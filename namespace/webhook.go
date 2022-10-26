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
	"djinn-ci.com/log"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v4"
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

	Author       *user.User
	User         *user.User
	Namespace    *Namespace
	LastDelivery *WebhookDelivery
}

var _ database.Model = (*Webhook)(nil)

func (w *Webhook) Dest() []interface{} {
	return []interface{}{
		&w.ID,
		&w.UserID,
		&w.AuthorID,
		&w.NamespaceID,
		&w.PayloadURL,
		&w.Secret,
		&w.SSL,
		&w.Events,
		&w.Active,
		&w.CreatedAt,
	}
}

// PruneWebhookDeliveries will delete all webhook deliveries that are older than
// 30 days. This is a continuous process, and will stop when the given context
// is cancelled.
func PruneWebhookDeliveries(ctx context.Context, log *log.Logger, db database.Pool) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
			q := query.Delete(
				webhookDeliveryTable,
				query.Where("id", "IN", query.Select(
					query.Columns("id"),
					query.From(webhookDeliveryTable),
					query.Where("created_at", ">", query.Lit("CURRENT_DATE + INTERVAL '30 days'")),
				)),
			)

			res, err := db.Exec(q.Build(), q.Args()...)

			if err != nil {
				log.Error.Println("failed to prune old webhook deliveries", err)
				continue
			}
			log.Debug.Println("deleted", res.RowsAffected(), "old webhook deliveries")
		}
	}
}

func (w *Webhook) Bind(m database.Model) {
	switch v := m.(type) {
	case *Namespace:
		if w.NamespaceID == v.ID {
			w.Namespace = v
		}
	case *user.User:
		w.Author = v

		if w.UserID == v.ID {
			w.User = v
		}
	}
}

func (w *Webhook) Endpoint(uri ...string) string {
	if w.Namespace == nil {
		return ""
	}
	return w.Namespace.Endpoint(append([]string{"webhooks", strconv.FormatInt(w.ID, 10)}, uri...)...)
}

func (w *Webhook) JSON(addr string) map[string]interface{} {
	if w == nil {
		return nil
	}

	events := make([]string, 0, len(event.Types))

	for _, ev := range event.Types {
		if w.Events.Has(ev) {
			events = append(events, ev.String())
		}
	}

	json := map[string]interface{}{
		"id":           w.ID,
		"user_id":      w.UserID,
		"author_id":    w.AuthorID,
		"namespace_id": w.NamespaceID,
		"payload_url":  w.PayloadURL.String(),
		"ssl":          w.SSL,
		"events":       events,
		"active":       w.Active,
		"url":          addr + w.Endpoint(),
	}

	if w.Namespace != nil {
		if v := w.Namespace.JSON(addr); v != nil {
			json["namespace"] = w.Namespace.JSON(addr)
		}
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

func (w *Webhook) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":           w.ID,
		"user_id":      w.UserID,
		"author_id":    w.AuthorID,
		"namespace_id": w.NamespaceID,
		"payload_url":  w.PayloadURL,
		"secret":       w.Secret,
		"ssl":          w.SSL,
		"events":       w.Events,
		"active":       w.Active,
		"created_at":   w.CreatedAt,
	}
}

type WebhookDelivery struct {
	ID              int64
	WebhookID       int64
	EventID         uuid.UUID
	NamespaceID     sql.NullInt64
	Error           sql.NullString
	Event           event.Type
	Redelivery      bool
	RequestHeaders  sql.NullString
	RequestBody     sql.NullString
	ResponseCode    sql.NullInt64
	ResponseStatus  string
	ResponseHeaders sql.NullString
	ResponseBody    sql.NullString
	Payload         map[string]interface{}
	Duration        time.Duration
	CreatedAt       time.Time

	Webhook *Webhook
}

var _ database.Model = (*WebhookDelivery)(nil)

func (d *WebhookDelivery) Dest() []interface{} {
	return []interface{}{
		&d.ID,
		&d.WebhookID,
		&d.EventID,
		&d.Error,
		&d.Event,
		&d.Redelivery,
		&d.RequestHeaders,
		&d.RequestBody,
		&d.ResponseCode,
		&d.ResponseHeaders,
		&d.ResponseBody,
		&d.Duration,
		&d.CreatedAt,
	}
}

func (*WebhookDelivery) Bind(_ database.Model)                {}
func (*WebhookDelivery) JSON(_ string) map[string]interface{} { return nil }
func (*WebhookDelivery) Values() map[string]interface{}       { return nil }

func (d *WebhookDelivery) Endpoint(_ ...string) string {
	return d.Webhook.Endpoint("deliveries", strconv.FormatInt(d.ID, 10))
}

type WebhookStore struct {
	database.Pool

	AESGCM   *crypto.AESGCM
	CertPool *x509.CertPool
}

var (
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

type WebhookParams struct {
	UserID      int64
	NamespaceID int64
	PayloadURL  *url.URL
	Secret      string
	SSL         bool
	Events      event.Type
	Active      bool
}

func (s *WebhookStore) Create(p WebhookParams) (*Webhook, error) {
	var (
		b   []byte
		err error
	)

	if p.Secret != "" {
		if s.AESGCM == nil {
			return nil, crypto.ErrNilAESGCM
		}

		b, err = s.AESGCM.Encrypt([]byte(p.Secret))

		if err != nil {
			return nil, errors.Err(err)
		}
	}

	ownerId, err := getNamespaceOwnerId(s.Pool, p.NamespaceID)

	if err != nil {
		return nil, errors.Err(err)
	}

	now := time.Now()

	url := hookURL{
		URL: p.PayloadURL,
	}

	q := query.Insert(
		webhookTable,
		query.Columns("user_id", "author_id", "namespace_id", "payload_url", "secret", "ssl", "events", "active", "created_at"),
		query.Values(ownerId, p.UserID, p.NamespaceID, url, b, p.SSL, p.Events, p.Active, now),
		query.Returning("id"),
	)

	var id int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}

	return &Webhook{
		ID:          id,
		UserID:      ownerId,
		AuthorID:    p.UserID,
		NamespaceID: p.NamespaceID,
		PayloadURL:  url,
		Secret:      b,
		SSL:         p.SSL,
		Events:      p.Events,
		Active:      p.Active,
		CreatedAt:   now,
	}, nil
}

func (s *WebhookStore) Update(id int64, p WebhookParams) error {
	var (
		b   []byte
		err error
	)

	if p.Secret != "" {
		if s.AESGCM == nil {
			return crypto.ErrNilAESGCM
		}

		b, err = s.AESGCM.Encrypt([]byte(p.Secret))

		if err != nil {
			return errors.Err(err)
		}
	}

	q := query.Update(
		webhookTable,
		query.Set("payload_url", query.Arg(hookURL{URL: p.PayloadURL})),
		query.Set("secret", query.Arg(b)),
		query.Set("ssl", query.Arg(p.SSL)),
		query.Set("events", query.Arg(p.Events)),
		query.Set("active", query.Arg(p.Active)),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *WebhookStore) Get(opts ...query.Option) (*Webhook, bool, error) {
	var wh Webhook

	ok, err := s.Pool.Get(webhookTable, &wh, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &wh, ok, nil
}

func (s *WebhookStore) All(opts ...query.Option) ([]*Webhook, error) {
	ww := make([]*Webhook, 0)

	new := func() database.Model {
		wh := &Webhook{}
		ww = append(ww, wh)
		return wh
	}

	if err := s.Pool.All(webhookTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return ww, nil
}

func (s *WebhookStore) Delivery(hookId, id int64) (*WebhookDelivery, bool, error) {
	var d WebhookDelivery

	opts := []query.Option{
		query.Where("id", "=", query.Arg(id)),
		query.Where("webhook_id", "=", query.Arg(hookId)),
	}

	ok, err := s.Pool.Get(webhookDeliveryTable, &d, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if d.ResponseCode.Valid {
		status := strconv.FormatInt(d.ResponseCode.Int64, 10) + " " + http.StatusText(int(d.ResponseCode.Int64))
		d.ResponseStatus = status
	}
	return &d, ok, nil
}

func (s *WebhookStore) Deliveries(id int64) ([]*WebhookDelivery, error) {
	dd := make([]*WebhookDelivery, 0)

	opts := []query.Option{
		query.Where("webhook_id", "=", query.Arg(id)),
		query.OrderDesc("created_at"),
		query.Limit(25),
	}

	new := func() database.Model {
		d := &WebhookDelivery{}
		dd = append(dd, d)
		return d
	}
	if err := s.Pool.All(webhookDeliveryTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return dd, nil
}

func (s *WebhookStore) LastDelivery(id int64) (*WebhookDelivery, error) {
	var d WebhookDelivery

	opts := []query.Option{
		query.Where("webhook_id", "=", query.Arg(id)),
		query.OrderDesc("created_at"),
	}

	if _, err := s.Pool.Get(webhookDeliveryTable, &d, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return &d, nil
}

func (s *WebhookStore) LoadLastDeliveries(ww ...*Webhook) error {
	mm := make([]database.Model, 0, len(ww))

	for _, w := range ww {
		mm = append(mm, w)
	}

	ids := database.Values("id", mm)

	opts := []query.Option{
		query.Where("webhook_id", "IN", database.List(ids...)),
		query.OrderDesc("created_at"),
	}

	deliveries := make(map[int64]*WebhookDelivery)
	dd := make([]*WebhookDelivery, 0)

	new := func() database.Model {
		d := &WebhookDelivery{}
		dd = append(dd, d)
		return d
	}

	if err := s.Pool.All(webhookDeliveryTable, new, opts...); err != nil {
		return errors.Err(err)
	}

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

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&count); err != nil {
		return errors.Err(err)
	}

	redelivery := count > 0

	var (
		deliveryError sql.NullString

		reqHeaders sql.NullString
		reqBody    sql.NullString

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

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *WebhookStore) realDeliver(w *Webhook, typ event.Type, r *bytes.Reader) (*http.Request, *http.Response, time.Duration, error) {
	cli := http.Client{
		Transport: http.DefaultTransport,
		Timeout:   time.Minute,
	}

	if w.SSL {
		if s.CertPool == nil {
			pool, err := x509.SystemCertPool()

			if err != nil {
				return nil, nil, 0, errors.Err(err)
			}
			s.CertPool = pool
		}

		cli.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: w.PayloadURL.Host,
				RootCAs:    s.CertPool,
			},
		}
	}

	var signature string

	if w.Secret != nil {
		if s.AESGCM == nil {
			return nil, nil, 0, crypto.ErrNilAESGCM
		}

		secret, err := s.AESGCM.Decrypt(w.Secret)

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
	w, ok, err := s.Get(query.Where("id", "=", query.Arg(id)))

	if err != nil {
		return errors.Err(err)
	}

	if !ok {
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

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&headers, &body); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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

	if err := s.createDelivery(w.ID, event, eventId, req, resp, dur, derr); err != nil {
		return errors.Err(err)
	}
	return nil
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

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}
