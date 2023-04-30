package namespace

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"database/sql/driver"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/log"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/google/uuid"
)

type hookURL struct {
	*url.URL
}

var (
	_ sql.Scanner   = (*hookURL)(nil)
	_ driver.Valuer = (*hookURL)(nil)
)

func WebhookURL(url *url.URL) hookURL {
	return hookURL{
		URL: url,
	}
}

func (u hookURL) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(u.String())

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (u hookURL) Value() (driver.Value, error) { return strings.ToLower(u.String()), nil }

func (u *hookURL) Scan(val interface{}) error {
	v, err := driver.String.ConvertValue(val)

	if err != nil {
		return errors.Err(err)
	}

	s, ok := v.(string)

	if !ok {
		return errors.New("namespace: could not type assert hookURL to string")
	}

	u.URL, err = url.Parse(s)

	if err != nil {
		return errors.Err(err)
	}
	return nil
}

type Webhook struct {
	loaded []string

	ID          int64
	UserID      int64
	AuthorID    int64
	NamespaceID int64
	PayloadURL  hookURL
	Secret      []byte
	SSL         bool
	Events      event.Type
	Active      bool
	CreatedAt   time.Time

	Author       *auth.User
	User         *auth.User
	Namespace    *Namespace
	LastDelivery *WebhookDelivery
}

var _ database.Model = (*Webhook)(nil)

func (w *Webhook) Primary() (string, any) { return "id", w.ID }

func (w *Webhook) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":           &w.ID,
		"user_id":      &w.UserID,
		"author_id":    &w.AuthorID,
		"namespace_id": &w.NamespaceID,
		"payload_url":  &w.PayloadURL,
		"secret":       &w.Secret,
		"ssl":          &w.SSL,
		"events":       &w.Events,
		"active":       &w.Active,
		"created_at":   &w.CreatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}

	w.loaded = r.Columns
	return nil
}

func (w *Webhook) Params() database.Params {
	params := database.Params{
		"id":           database.ImmutableParam(w.ID),
		"user_id":      database.CreateOnlyParam(w.UserID),
		"author_id":    database.CreateOnlyParam(w.AuthorID),
		"namespace_id": database.CreateOnlyParam(w.NamespaceID),
		"payload_url":  database.CreateUpdateParam(w.PayloadURL),
		"secret":       database.CreateUpdateParam(w.Secret),
		"ssl":          database.CreateUpdateParam(w.SSL),
		"events":       database.CreateUpdateParam(w.Events),
		"active":       database.CreateUpdateParam(w.Active),
		"created_at":   database.CreateOnlyParam(w.CreatedAt),
	}

	if len(w.loaded) > 0 {
		params.Only(w.loaded...)
	}
	return params
}

func (w *Webhook) Bind(m database.Model) {
	switch v := m.(type) {
	case *Namespace:
		if w.NamespaceID == v.ID {
			w.Namespace = v
		}
	case *auth.User:
		w.Author = v

		if w.UserID == v.ID {
			w.User = v
		}
	}
}

func (w *Webhook) MarshalJSON() ([]byte, error) {
	if w == nil {
		return []byte("null"), nil
	}

	events := make([]string, 0, len(event.Types))

	for _, ev := range event.Types {
		if w.Events.Has(ev) {
			events = append(events, ev.String())
		}
	}

	b, err := json.Marshal(map[string]any{
		"id":            w.ID,
		"user_id":       w.UserID,
		"author_id":     w.AuthorID,
		"namespace_id":  w.NamespaceID,
		"payload_url":   w.PayloadURL,
		"ssl":           w.SSL,
		"events":        events,
		"active":        w.Active,
		"url":           env.DJINN_API_SERVER + w.Endpoint(),
		"namespace":     w.Namespace,
		"last_response": w.LastDelivery,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (w *Webhook) Endpoint(elems ...string) string {
	if w.Namespace == nil {
		return ""
	}
	return w.Namespace.Endpoint(append([]string{"webhooks", strconv.FormatInt(w.ID, 10)}, elems...)...)
}

// PruneWebhookDeliveries will delete all webhook deliveries that are older than
// 30 days. This is a continuous process, and will stop when the given context
// is cancelled.
func PruneWebhookDeliveries(ctx context.Context, log *log.Logger, db *database.Pool) {
	t := time.NewTicker(time.Minute)

	for {
		select {
		case <-ctx.Done():
			log.Info.Println("stopping pruning of webhooks")
			t.Stop()
			return
		case <-t.C:
			q := query.Delete(
				webhookDeliveryTable,
				query.Where("id", "IN", query.Select(
					query.Columns("id"),
					query.From(webhookDeliveryTable),
					query.Where("created_at", ">", query.Lit("CURRENT_DATE + INTERVAL '30 days'")),
				)),
			)

			res, err := db.Exec(ctx, q.Build(), q.Args()...)

			if err != nil {
				log.Error.Println("failed to prune old webhook deliveries", err)
				continue
			}
			log.Debug.Println("deleted", res.RowsAffected(), "old webhook deliveries")
		}
	}
}

type WebhookStore struct {
	*database.Store[*Webhook]

	AESGCM *crypto.AESGCM
	Certs  *x509.CertPool
}

func NewWebhookStore(pool *database.Pool) *database.Store[*Webhook] {
	return database.NewStore[*Webhook](pool, webhookTable, func() *Webhook {
		return &Webhook{}
	})
}

const webhookTable = "namespace_webhooks"

var ErrUnknownWebhook = errors.New("unknown webhook")

func decodeHeaders(s string) http.Header {
	m := make(map[string][]string)

	buf := make([]rune, 0)

	var (
		key string
		r0  rune
	)

	for _, r := range s {
		switch r {
		case ' ':
			if r0 == ';' || r0 == ':' {
				continue
			}
		case ':':
			key = string(buf)
			buf = buf[0:0]
		case ';':
			m[key] = append(m[key], strings.TrimSpace(string(buf)))
			buf = buf[0:0]
		case '\n':
			m[key] = append(m[key], strings.TrimSpace(string(buf)))

			buf = buf[0:0]
			key = key[0:0]
		default:
			buf = append(buf, r)
		}
		r0 = r
	}

	if key != "" {
		m[key] = append(m[key], string(buf))
	}
	return http.Header(m)
}

func encodeHeaders(headers http.Header) string {
	var buf bytes.Buffer
	headers.Write(&buf)
	return buf.String()
}

type WebhookParams struct {
	User       *auth.User
	Namespace  *Namespace
	PayloadURL *url.URL
	Secret     string
	SSL        bool
	Events     event.Type
	Active     bool
}

func (s *WebhookStore) Create(ctx context.Context, p *WebhookParams) (*Webhook, error) {
	var secret []byte

	if p.Secret != "" {
		if s.AESGCM == nil {
			return nil, crypto.ErrNilAESGCM
		}

		b, err := s.AESGCM.Encrypt([]byte(p.Secret))

		if err != nil {
			return nil, errors.Err(err)
		}
		secret = b
	}

	u, ok, err := user.NewStore(s.Pool).Get(ctx, user.WhereID(p.Namespace.UserID))

	if err != nil {
		return nil, errors.Err(err)
	}

	if !ok {
		return nil, database.ErrNoRows
	}

	wh := Webhook{
		UserID:      p.Namespace.UserID,
		AuthorID:    p.User.ID,
		NamespaceID: p.Namespace.ID,
		PayloadURL:  hookURL{URL: p.PayloadURL},
		Secret:      secret,
		SSL:         p.SSL,
		Events:      p.Events,
		Active:      p.Active,
		CreatedAt:   time.Now(),
		Namespace:   p.Namespace,
		Author:      p.User,
		User:        u,
	}

	if err := s.Store.Create(ctx, &wh); err != nil {
		return nil, errors.Err(err)
	}
	return &wh, nil
}

func (s *WebhookStore) Update(ctx context.Context, w *Webhook) error {
	loaded := w.loaded
	w.loaded = []string{"payload_url", "secret", "ssl", "events", "active"}

	if w.Secret != nil {
		if s.AESGCM == nil {
			return crypto.ErrNilAESGCM
		}

		b, err := s.AESGCM.Encrypt(w.Secret)

		if err != nil {
			return errors.Err(err)
		}
		w.Secret = b
	}

	if err := s.Store.Update(ctx, w); err != nil {
		return errors.Err(err)
	}

	w.loaded = loaded
	return nil
}

type WebhookDelivery struct {
	ID              int64
	WebhookID       int64
	EventID         uuid.UUID
	NamespaceID     database.Null[int64]
	Error           database.Null[string]
	Event           event.Type
	Redelivery      bool
	RequestHeaders  database.Null[string]
	RequestBody     database.Null[string]
	ResponseCode    database.Null[int64]
	ResponseStatus  string
	ResponseHeaders database.Null[string]
	ResponseBody    database.Null[string]
	Payload         map[string]any
	Duration        time.Duration
	CreatedAt       time.Time

	Webhook *Webhook
}

var _ database.Model = (*WebhookDelivery)(nil)

func (d *WebhookDelivery) Primary() (string, any) { return "id", d.ID }

func (d *WebhookDelivery) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":               &d.ID,
		"webhook_id":       &d.WebhookID,
		"event_id":         &d.EventID,
		"error":            &d.Error,
		"event":            &d.Event,
		"redelivery":       &d.Redelivery,
		"request_headers":  &d.RequestHeaders,
		"request_body":     &d.RequestBody,
		"response_code":    &d.ResponseCode,
		"response_headers": &d.ResponseHeaders,
		"response_body":    &d.ResponseBody,
		"duration":         &d.Duration,
		"created_at":       &d.CreatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (d *WebhookDelivery) Params() database.Params {
	return database.Params{
		"id":               database.ImmutableParam(d.ID),
		"webhook_id":       database.CreateOnlyParam(d.WebhookID),
		"event_id":         database.CreateOnlyParam(d.EventID),
		"error":            database.CreateOnlyParam(d.Error),
		"event":            database.CreateOnlyParam(d.Event),
		"redelivery":       database.CreateOnlyParam(d.Redelivery),
		"request_headers":  database.CreateOnlyParam(d.RequestHeaders),
		"request_body":     database.CreateOnlyParam(d.RequestBody),
		"response_code":    database.CreateOnlyParam(d.ResponseCode),
		"response_headers": database.CreateOnlyParam(d.ResponseHeaders),
		"response_body":    database.CreateOnlyParam(d.ResponseBody),
		"duration":         database.CreateOnlyParam(d.Duration),
		"created_at":       database.CreateOnlyParam(d.CreatedAt),
	}
}

func (*WebhookDelivery) Bind(database.Model) {}

func (d *WebhookDelivery) MarshalJSON() ([]byte, error) {
	if d == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"error":      d.Error,
		"code":       d.ResponseCode,
		"duration":   d.Duration,
		"created_at": d.CreatedAt,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (d *WebhookDelivery) Endpoint(...string) string {
	return d.Webhook.Endpoint("deliveries", strconv.FormatInt(d.ID, 10))
}

const webhookDeliveryTable = "namespace_webhook_deliveries"

type deliveryStore struct {
	*database.Store[*WebhookDelivery]
}

func newDeliveryStore(pool *database.Pool) deliveryStore {
	return deliveryStore{
		Store: database.NewStore[*WebhookDelivery](pool, webhookDeliveryTable, func() *WebhookDelivery {
			return &WebhookDelivery{}
		}),
	}
}

func (s deliveryStore) Create(ctx context.Context, d *WebhookDelivery) error {
	var count int64

	q := query.Select(
		query.Count("webhook_id"),
		query.From(webhookDeliveryTable),
		query.Where("webhook_id", "=", query.Arg(d.WebhookID)),
		query.Where("event_id", "=", query.Arg(d.EventID)),
	)

	if err := s.QueryRow(ctx, q.Build(), q.Args()...).Scan(&count); err != nil {
		return errors.Err(err)
	}

	d.Redelivery = count > 0

	if err := s.Store.Create(ctx, d); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *WebhookStore) Delivery(ctx context.Context, w *Webhook, id int64) (*WebhookDelivery, bool, error) {
	d, ok, err := newDeliveryStore(s.Pool).Get(
		ctx,
		query.Where("id", "=", query.Arg(id)),
		query.Where("webhook_id", "=", query.Arg(w.ID)),
	)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if d.ResponseCode.Valid {
		status := strconv.FormatInt(d.ResponseCode.Elem, 10) + " " + http.StatusText(int(d.ResponseCode.Elem))
		d.ResponseStatus = status
	}
	return d, ok, nil
}

func (s *WebhookStore) Deliveries(ctx context.Context, w *Webhook) ([]*WebhookDelivery, error) {
	dd, err := newDeliveryStore(s.Pool).All(
		ctx,
		query.Where("webhook_id", "=", query.Arg(w.ID)),
		query.OrderDesc("created_at"),
		query.Limit(25),
	)

	if err != nil {
		return nil, errors.Err(err)
	}
	return dd, nil
}

func (s *WebhookStore) LastDelivery(ctx context.Context, w *Webhook) (*WebhookDelivery, error) {
	d, _, err := newDeliveryStore(s.Pool).Get(
		ctx,
		query.Where("webhook_id", "=", query.Arg(w.ID)),
		query.OrderDesc("created_at"),
	)

	if err != nil {
		return nil, errors.Err(err)
	}
	return d, nil
}

func (s *WebhookStore) LoadLastDeliveries(ctx context.Context, ww ...*Webhook) error {
	if len(ww) == 0 {
		return nil
	}

	ids := database.Map[*Webhook, any](ww, func(w *Webhook) any {
		return w.ID
	})

	dd, err := newDeliveryStore(s.Pool).All(ctx, query.Where("webhook_id", "IN", query.List(ids...)))

	if err != nil {
		return errors.Err(err)
	}

	deliverytab := make(map[int64]*WebhookDelivery)

	for _, d := range dd {
		if _, ok := deliverytab[d.WebhookID]; !ok {
			deliverytab[d.WebhookID] = d
		}
	}

	for _, w := range ww {
		w.LastDelivery = deliverytab[w.ID]
	}
	return nil
}

func (s *WebhookStore) storeDelivery(ctx context.Context, d *WebhookDelivery) error {
	var count int64

	q := query.Select(
		query.Count("webhook_id"),
		query.From(webhookDeliveryTable),
		query.Where("webhook_id", "=", query.Arg(d.WebhookID)),
		query.Where("event_id", "=", query.Arg(d.EventID)),
	)

	if err := s.QueryRow(ctx, q.Build(), q.Args()...).Scan(&count); err != nil {
		return errors.Err(err)
	}

	d.Redelivery = count > 0

	store := newDeliveryStore(s.Pool)

	if err := store.Create(ctx, d); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *WebhookStore) handleDelivery(ctx context.Context, w *Webhook, e *event.Event) error {
	cli := http.Client{
		Transport: http.DefaultTransport,
		Timeout:   time.Minute,
	}

	if w.SSL {
		if s.Certs == nil {
			pool, err := x509.SystemCertPool()

			if err != nil {
				return errors.Err(err)
			}
			s.Certs = pool
		}

		cli.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: w.PayloadURL.Host,
				RootCAs:    s.Certs,
			},
		}
	}

	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(e.Data)

	r := bytes.NewReader(buf.Bytes())

	var signature string

	if w.Secret != nil {
		if s.AESGCM == nil {
			return crypto.ErrNilAESGCM
		}

		secret, err := s.AESGCM.Decrypt(w.Secret)

		if err != nil {
			return errors.Err(err)
		}

		hmac := hmac.New(sha256.New, secret)

		io.Copy(hmac, r)

		r.Seek(0, io.SeekStart)
		signature = "sha256=" + hex.EncodeToString(hmac.Sum(nil))
	}

	req := http.Request{
		Method: "POST",
		URL:    w.PayloadURL.URL,
		Header: http.Header{
			"Accept":             {"*/*"},
			"Content-Length":     {strconv.FormatInt(int64(r.Size()), 10)},
			"Content-Type":       {"application/json; charset=utf-8"},
			"User-Agent":         {"Djinn-CI-Hook"},
			"X-Djinn-CI-Event":   {e.Type.String()},
			"X-Djinn-CI-Hook-ID": {strconv.FormatInt(w.ID, 10)},
		},
		Body: io.NopCloser(r),
	}

	if signature != "" {
		req.Header.Set("X-Djinn-CI-Signature", signature)
	}

	start := time.Now()

	resp, resperr := cli.Do(&req)

	defer func() {
		if resp != nil {
			resp.Body.Close()
		}
	}()

	errmsg := ""

	if resperr != nil {
		errmsg = resperr.Error()
	}

	r.Seek(0, io.SeekStart)
	reqBody, _ := io.ReadAll(req.Body)

	reqHeaders := encodeHeaders(req.Header)

	var (
		respCode    int64
		respHeaders string
		respBody    string
	)

	if resp != nil {
		respCode = int64(resp.StatusCode)
		respHeaders = encodeHeaders(resp.Header)

		b, _ := io.ReadAll(resp.Body)
		respBody = string(b)
	}

	d := WebhookDelivery{
		WebhookID: w.ID,
		EventID:   e.ID,
		NamespaceID: database.Null[int64]{
			Elem:  w.NamespaceID,
			Valid: w.NamespaceID > 0,
		},
		Event: e.Type,
		Error: database.Null[string]{
			Elem:  errmsg,
			Valid: resperr != nil,
		},
		RequestHeaders: database.Null[string]{
			Elem:  reqHeaders,
			Valid: reqHeaders != "",
		},
		RequestBody: database.Null[string]{
			Elem:  string(reqBody),
			Valid: reqBody != nil,
		},
		ResponseCode: database.Null[int64]{
			Elem:  respCode,
			Valid: respCode > 0,
		},
		ResponseHeaders: database.Null[string]{
			Elem:  respHeaders,
			Valid: respHeaders != "",
		},
		ResponseBody: database.Null[string]{
			Elem:  respBody,
			Valid: respBody != "",
		},
		Duration: time.Now().Sub(start),
	}

	if err := newDeliveryStore(s.Pool).Create(ctx, &d); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *WebhookStore) Redeliver(ctx context.Context, w *Webhook, eventId uuid.UUID) error {
	d, _, err := newDeliveryStore(s.Pool).Get(
		ctx,
		query.Where("webhook_id", "=", query.Arg(w.ID)),
		query.Where("event_id", "=", query.Arg(eventId)),
	)

	if err != nil {
		return errors.Err(err)
	}

	hdr := decodeHeaders(d.RequestHeaders.Elem)
	typ, _ := event.Lookup(hdr.Get("X-Djinn-CI-Event"))

	data := make(map[string]any)

	if err := json.NewDecoder(strings.NewReader(d.RequestBody.Elem)).Decode(&data); err != nil {
		return errors.Err(err)
	}

	ev := event.Event{
		ID:        eventId,
		Type:      typ,
		Data:      data,
		CreatedAt: time.Now(),
	}

	if err := s.handleDelivery(ctx, w, &ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *WebhookStore) Dispatch(ev *event.Event) error {
	if !ev.NamespaceID.Valid {
		return nil
	}

	ctx := context.Background()

	ww, err := s.All(
		ctx,
		query.Where("namespace_id", "=", query.Arg(ev.NamespaceID)),
		query.Where("active", "=", query.Lit(true)),
	)

	if err != nil {
		return errors.Err(err)
	}

	for _, w := range ww {
		if !w.Events.Has(ev.Type) {
			continue
		}

		if err := s.handleDelivery(ctx, w, ev); err != nil {
			return errors.Err(err)
		}
	}
	return nil
}
