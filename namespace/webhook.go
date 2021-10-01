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

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/google/uuid"

	"github.com/jmoiron/sqlx"
)

type hookURL struct {
	*url.URL
}

type Webhook struct {
	ID          int64      `db:"id"`
	UserID      int64      `db:"user_id"`
	AuthorID    int64      `db:"author_id"`
	NamespaceID int64      `db:"namespace_id"`
	PayloadURL  hookURL    `db:"payload_url"`
	Secret      []byte     `db:"secret"`
	SSL         bool       `db:"ssl"`
	Events      event.Type `db:"events"`
	Active      bool       `db:"active"`
	CreatedAt   time.Time  `db:"created_at"`

	User         *user.User       `db:"-"`
	Namespace    *Namespace       `db:"-"`
	LastDelivery *WebhookDelivery `db:"-"`
}

type WebhookDelivery struct {
	ID              int64                  `db:"id"`
	WebhookID       int64                  `db:"webhook_id"`
	EventID         uuid.UUID              `db:"event_id"`
	NamespaceID     sql.NullInt64          `db:"-"`
	Error           sql.NullString         `db:"error"`
	Event           event.Type             `db:"event"`
	Redelivery      bool                   `db:"redelivery"`
	RequestHeaders  sql.NullString         `db:"request_headers"`
	RequestBody     sql.NullString         `db:"request_body"`
	ResponseCode    sql.NullInt64          `db:"response_code"`
	ResponseStatus  string                 `db:"-"`
	ResponseHeaders sql.NullString         `db:"response_headers"`
	ResponseBody    sql.NullString         `db:"response_body"`
	Payload         map[string]interface{} `db:"-"`
	Duration        time.Duration          `db:"duration"`
	CreatedAt       time.Time              `db:"created_at"`

	Webhook *Webhook `db:"-"`
}

type WebhookStore struct {
	database.Store

	block *crypto.Block
	pool  *x509.CertPool

	User      *user.User
	Namespace *Namespace
}

var (
	_ database.Model  = (*Webhook)(nil)
	_ database.Binder = (*WebhookStore)(nil)

	_ sql.Scanner   = (*hookURL)(nil)
	_ driver.Valuer = (*hookURL)(nil)

	webhookTable         = "namespace_webhooks"
	webhookDeliveryTable = "namespace_webhook_deliveries"

	ErrUnknownWebhook = errors.New("unknown webhook")
)

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

func WebhookURL(url *url.URL) hookURL {
	return hookURL{
		URL: url,
	}
}

func NewWebhookStore(db *sqlx.DB, mm ...database.Model) *WebhookStore {
	s := &WebhookStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func NewWebhookStoreWithBlock(db *sqlx.DB, block *crypto.Block, mm ...database.Model) *WebhookStore {
	s := NewWebhookStore(db, mm...)
	s.block = block
	return s
}

func WebhookFromContext(ctx context.Context) (*Webhook, bool) {
	w, ok := ctx.Value("webhook").(*Webhook)
	return w, ok
}

func (e *WebhookDelivery) Endpoint(_ ...string) string {
	return e.Webhook.Endpoint("deliveries", strconv.FormatInt(e.ID, 10))
}

func (u hookURL) Value() (driver.Value, error) { return strings.ToLower(u.String()), nil }

func (u *hookURL) Scan(v interface{}) error {
	b, err := database.Scan(v)

	if err != nil {
		return errors.Err(err)
	}

	u.URL, err = url.Parse(string(b))

	return errors.Err(err)
}

func (w *Webhook) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *Namespace:
			w.Namespace = v
		case *user.User:
			w.User = v
		}
	}
}

func (w *Webhook) Endpoint(uri ...string) string {
	if w.Namespace.IsZero() {
		return ""
	}
	return w.Namespace.Endpoint(append([]string{"webhooks", strconv.FormatInt(w.ID, 10)}, uri...)...)
}

func (w *Webhook) IsZero() bool {
	return w == nil || w.ID == 0 &&
		w.AuthorID == 0 &&
		w.NamespaceID == 0 &&
		w.PayloadURL.URL == nil &&
		w.Secret == nil &&
		!w.SSL &&
		w.Events == event.Type(0) &&
		w.CreatedAt == time.Time{}
}

func (w *Webhook) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":           w.ID,
		"user_id":      w.UserID,
		"author_id":    w.AuthorID,
		"namespace_id": w.NamespaceID,
		"payload_url":  w.PayloadURL.String(),
		"ssl":          w.SSL,
		"active":       w.Active,
		"url":          addr + w.Endpoint(),
	}

	events := make([]string, 0, len(event.Types))

	for _, ev := range event.Types {
		if w.Events.Has(ev) {
			events = append(events, ev.String())
		}
	}

	json["events"] = events

	if !w.Namespace.IsZero() {
		json["namespace"] = w.Namespace.JSON(addr)
	}

	if w.LastDelivery != nil {
		var deliveryError interface{} = nil

		if w.LastDelivery.Error.Valid {
			deliveryError = w.LastDelivery.Error.String
		}

		json["last_response"] = map[string]interface{}{
			"error":      deliveryError,
			"code":       w.LastDelivery.ResponseCode,
			"duration":   w.LastDelivery.Duration,
			"created_at": w.LastDelivery.CreatedAt.Format(time.RFC3339),
		}
	}
	return json
}

func (w *Webhook) Primary() (string, int64) { return "id", w.ID }

func (w *Webhook) SetPrimary(id int64) { w.ID = id }

func (w *Webhook) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      w.UserID,
		"author_id":    w.AuthorID,
		"namespace_id": w.NamespaceID,
		"payload_url":  w.PayloadURL,
		"secret":       w.Secret,
		"ssl":          w.SSL,
		"events":       w.Events,
		"active":       w.Active,
	}
}

func (s *WebhookStore) New() *Webhook {
	w := &Webhook{
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		w.UserID = s.User.ID
	}
	if s.Namespace != nil {
		w.NamespaceID = s.Namespace.ID
	}
	return w
}

func (s *WebhookStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *Namespace:
			s.Namespace = v
		case *user.User:
			s.User = v
		}
	}
}

func (s *WebhookStore) Create(authorId int64, payloadUrl *url.URL, secret string, ssl bool, events event.Type, active bool) (*Webhook, error) {
	var (
		b   []byte
		err error
	)

	if secret != "" {
		if s.block == nil {
			return nil, errors.New("nil block cipher")
		}

		b, err = s.block.Encrypt([]byte(secret))

		if err != nil {
			return nil, errors.Err(err)
		}
	}

	w := s.New()
	w.AuthorID = authorId
	w.PayloadURL = hookURL{URL: payloadUrl}
	w.Secret = b
	w.SSL = ssl
	w.Events = events
	w.Active = active

	err = s.Store.Create(webhookTable, w)
	return w, errors.Err(err)
}

func (s *WebhookStore) Update(id int64, payloadUrl *url.URL, secret string, ssl bool, events event.Type, active bool) error {
	var (
		b   []byte
		err error
	)

	if secret != "" {
		if s.block == nil {
			return errors.New("nil block cipher")
		}

		b, err = s.block.Encrypt([]byte(secret))

		if err != nil {
			return errors.Err(err)
		}
	}

	q := query.Update(
		webhookTable,
		query.Set("payload_url", query.Arg(hookURL{URL: payloadUrl})),
		query.Set("secret", query.Arg(b)),
		query.Set("ssl", query.Arg(ssl)),
		query.Set("events", query.Arg(events)),
		query.Set("active", query.Arg(active)),
		query.Where("id", "=", query.Arg(id)),
	)

	_, err = s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

func (s *WebhookStore) All(opts ...query.Option) ([]*Webhook, error) {
	ww := make([]*Webhook, 0)

	opts = append([]query.Option{
		WhereCollaborator(s.User),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.All(&ww, webhookTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, w := range ww {
		w.User = s.User
		w.Namespace = s.Namespace
	}
	return ww, errors.Err(err)
}

func (s *WebhookStore) Delivery(hookId, id int64) (*WebhookDelivery, error) {
	d := &WebhookDelivery{}

	opts := []query.Option{
		query.Where("id", "=", query.Arg(id)),
		query.Where("webhook_id", "=", query.Arg(hookId)),
	}

	err := s.Store.Get(d, webhookDeliveryTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	if d.ResponseCode.Valid {
		d.ResponseStatus = strconv.FormatInt(d.ResponseCode.Int64, 10) + " " + http.StatusText(int(d.ResponseCode.Int64))
	}
	return d, errors.Err(err)
}

func (s *WebhookStore) Deliveries(id int64) ([]*WebhookDelivery, error) {
	dd := make([]*WebhookDelivery, 0)

	opts := []query.Option{
		query.Where("webhook_id", "=", query.Arg(id)),
		query.OrderDesc("created_at"),
		query.Limit(25),
	}

	err := s.Store.All(&dd, webhookDeliveryTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	w := &Webhook{
		ID:        id,
		Namespace: s.Namespace,
	}

	for _, d := range dd {
		d.Webhook = w
	}
	return dd, errors.Err(err)
}

func (s *WebhookStore) LastDelivery(id int64) (*WebhookDelivery, error) {
	d := &WebhookDelivery{}

	opts := []query.Option{
		query.Where("webhook_id", "=", query.Arg(id)),
		query.OrderDesc("created_at"),
	}

	err := s.Store.Get(d, webhookDeliveryTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return d, errors.Err(err)
}

func (s *WebhookStore) LoadLastDeliveries(ww ...*Webhook) error {
	mm := make([]database.Model, 0, len(ww))

	for _, w := range ww {
		mm = append(mm, w)
	}

	ids := database.MapKey("id", mm)

	opts := []query.Option{
		query.Where("webhook_id", "IN", database.List(ids...)),
		query.OrderDesc("created_at"),
	}

	dd := make([]*WebhookDelivery, 0, len(ww))

	err := s.Store.All(&dd, webhookDeliveryTable, opts...)

	if err != nil {
		return errors.Err(err)
	}

	deliveries := make(map[int64]*WebhookDelivery)

	for _, d := range dd {
		if _, ok := deliveries[d.WebhookID]; !ok {
			deliveries[d.WebhookID] = d
		}
	}

	for _, w := range ww {
		w.LastDelivery = deliveries[w.ID]
	}
	return nil
}

func (s *WebhookStore) Get(opts ...query.Option) (*Webhook, error) {
	w := &Webhook{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		WhereCollaborator(s.User),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(w, webhookTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return w, errors.Err(err)
}

// createDelivery records the given webhook delivery in the database. derr
// denotes any errors that occurred during delivery, such as the HTTP client
// not being able to communicate with the remote server.
func (s *WebhookStore) createDelivery(hookId int64, typ event.Type, eventId uuid.UUID, req *http.Request, resp *http.Response, dur time.Duration, derr error) error {
	var count int64

	q := query.Select(
		query.Count("webhook_id"),
		query.From(webhookDeliveryTable),
		query.Where("webhook_id", "=", query.Arg(hookId)),
		query.Where("event_id", "=", query.Arg(eventId)),
	)

	if err := s.DB.QueryRow(q.Build(), q.Args()...).Scan(&count); err != nil {
		return errors.Err(err)
	}

	redelivery := count > 0

	var (
		deliveryError sql.NullString

		reqHeaders  sql.NullString
		reqBody     sql.NullString

		respStatus  sql.NullInt64
		respHeaders sql.NullString
		respBody    sql.NullString
	)

	if derr != nil {
		deliveryError.Valid = true
		deliveryError.String = errors.Cause(derr).Error()
	}

	if req != nil {
		reqHeaders.Valid = true
		reqHeaders.String = encodeHeaders(req.Header)

		b, _ := io.ReadAll(req.Body)

		reqBody.Valid = true
		reqBody.String = string(b)
	}

	if resp != nil {
		respStatus.Valid = true
		respStatus.Int64 = int64(resp.StatusCode)

		respHeaders.Valid = true
		respHeaders.String = encodeHeaders(resp.Header)

		b, _ := io.ReadAll(resp.Body)

		respBody.Valid = true
		respBody.String = string(b)
	}

	q = query.Insert(
		webhookDeliveryTable,
		query.Columns("webhook_id", "event_id", "event", "error", "redelivery", "request_headers", "request_body", "response_code", "response_headers", "response_body", "duration"),
		query.Values(
			hookId,
			eventId,
			typ,
			deliveryError,
			redelivery,
			reqHeaders,
			reqBody,
			respStatus,
			respHeaders,
			respBody,
			dur,
		),
	)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

func (s *WebhookStore) realDeliver(w *Webhook, typ event.Type, r *bytes.Reader) (*http.Request, *http.Response, time.Duration, error) {
	cli := http.Client{
		Transport: http.DefaultTransport,
		Timeout:   time.Minute,
	}

	if w.SSL {
		if s.pool == nil {
			pool, err := x509.SystemCertPool()

			if err != nil {
				return nil, nil, 0, errors.Err(err)
			}
			s.pool = pool
		}

		cli.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: w.PayloadURL.Host,
				RootCAs:    s.pool,
			},
		}
	}

	var signature string

	if w.Secret != nil {
		if s.block == nil {
			return nil, nil, 0, errors.New("nil block cipher")
		}

		secret, err := s.block.Decrypt(w.Secret)

		if err != nil {
			return nil, nil, 0, errors.Err(err)
		}

		var buf bytes.Buffer

		hmac := hmac.New(sha256.New, secret)
		tee := io.TeeReader(r, &buf)

		io.Copy(hmac, tee)

		r = bytes.NewReader(buf.Bytes())
		signature = "sha256=" + hex.EncodeToString(hmac.Sum(nil))
	}

	req := &http.Request{
		Method: "POST",
		URL:    w.PayloadURL.URL,
		Header: map[string][]string{
			"Accept":             {"*/*"},
			"Content-Length":     {strconv.FormatInt(int64(r.Size()), 10)},
			"Content-Type":       {"application/json; charset=utf-8"},
			"User-Agent":         {"Djinn-CI-Hook"},
			"X-Djinn-CI-Event":   {typ.String()},
			"X-Djinn-CI-Hook-ID": {strconv.FormatInt(w.ID, 10)},
		},
		Body: io.NopCloser(r),
	}

	if signature != "" {
		req.Header["X-Djinn-CI-Signature"] = []string{signature}
	}

	start := time.Now()

	resp, err := cli.Do(req)

	if err != nil {
		return req, nil, 0, errors.Err(err)
	}
	return req, resp, time.Now().Sub(start), nil
}

func (s *WebhookStore) Redeliver(id int64, eventId uuid.UUID) error {
	w, err := s.Get(query.Where("id", "=", query.Arg(id)))

	if err != nil {
		return errors.Err(err)
	}

	if w.IsZero() {
		return database.ErrNotFound
	}

	var (
		headers string
		body    string
	)

	q := query.Select(
		query.Columns("request_headers", "request_body"),
		query.From(webhookDeliveryTable),
		query.Where("webhook_id", "=", query.Arg(w.ID)),
		query.Where("event_id", "=", query.Arg(eventId)),
	)

	if err := s.DB.QueryRow(q.Build(), q.Args()...).Scan(&headers, &body); err != nil {
		if err == sql.ErrNoRows {
			return database.ErrNotFound
		}
		return errors.Err(err)
	}

	hdr := decodeHeaders(headers)
	r := bytes.NewReader([]byte(body))

	event, _ := event.Lookup(hdr["X-Djinn-CI-Event"][0])

	req, resp, dur, derr := s.realDeliver(w, event, r)

	if req != nil {
		req.Body = io.NopCloser(strings.NewReader(body))
	}
	return s.createDelivery(w.ID, event, eventId, req, resp, dur, derr)
}

func (s *WebhookStore) Dispatch(ev *event.Event) error {
	if !ev.NamespaceID.Valid {
		return nil
	}

	ww, err := s.All(
		query.Where("namespace_id", "=", query.Arg(ev.NamespaceID)),
		query.Where("active", "=", query.Lit(true)),
	)

	if err != nil {
		return errors.Err(err)
	}

	var buf bytes.Buffer

	json.NewEncoder(&buf).Encode(ev.Data)

	r := bytes.NewReader(buf.Bytes())

	for _, w := range ww {
		if !w.Events.Has(ev.Type) {
			continue
		}

		req, resp, dur, derr := s.realDeliver(w, ev.Type, r)

		r.Seek(0, io.SeekStart)

		if req != nil {
			req.Body = io.NopCloser(&buf)
		}

		if err := s.createDelivery(w.ID, ev.Type, ev.ID, req, resp, dur, derr); err != nil {
			return errors.Err(err)
		}

		if resp != nil {
			resp.Body.Close()
		}
	}
	return nil
}

func (s *WebhookStore) Delete(id int64) error {
	q := query.Delete(webhookTable, query.Where("id", "=", query.Arg(id)))

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}
