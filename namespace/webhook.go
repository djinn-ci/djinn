package namespace

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type hookURL struct {
	*url.URL
}

type WebhookEvent int64

type Webhook struct {
	ID          int64        `db:"id"`
	UserID      int64        `db:"user_id"`
	AuthorID    int64        `db:"author_id"`
	NamespaceID int64        `db:"namespace_id"`
	PayloadURL  hookURL      `db:"payload_url"`
	Secret      string       `db:"secret"`
	SSL         bool         `db:"ssl"`
	Events      WebhookEvent `db:"events"`
	Active      bool         `db:"active"`
	CreatedAt   time.Time    `db:"created_at"`

	User         *user.User       `db:"-"`
	Namespace    *Namespace       `db:"-"`
	LastDelivery *WebhookDelivery `db:"-"`
}

type WebhookDelivery struct {
	ID              string         `db:"id"`
	WebhookID       int64          `db:"webhook_id"`
	DeliveryID      string         `db:"delivery_id"`
	DeliveryErr     bool           `db:"delivery_err"`
	RequestHeaders  string         `db:"request_headers"`
	RequestBody     string         `db:"request"`
	ResponseCode    int            `db:"response_code"`
	ResponseHeaders string         `db:"response_headers"`
	ResponseBody    sql.NullString `db:"response"`
	Duration        time.Duration  `db:"duration"`
	CreatedAt       time.Time      `db:"created_at"`
}

type WebhookStore struct {
	database.Store

	pool *x509.CertPool

	User      *user.User
	Namespace *Namespace
}

//go:generate stringer -type WebhookEvent -linecomment
const (
	BuildSubmitted WebhookEvent = 1 << iota // build_submitted
	BuildStarted                            // build_started
	BuildFinished                           // build_finished
	BuildTagged                             // build_tagged
	CollaboratorJoined                      // collaborator_joined
	Cron                                    // cron
	Images                                  // images
	Objects                                 // objects
	Variables                               // variables
	SSHKeys                                 // ssh_keys
)

var (
	_ database.Model  = (*Webhook)(nil)
	_ database.Binder = (*WebhookStore)(nil)

	_ sql.Scanner   = (*hookURL)(nil)
	_ driver.Valuer = (*hookURL)(nil)

	_ sql.Scanner   = (*WebhookEvent)(nil)
	_ driver.Valuer = (*WebhookEvent)(nil)

	webhookTable         = "namespace_webhooks"
	webhookDeliveryTable = "namespace_webhook_deliveries"

	webhookEventsMap = map[string]WebhookEvent{
		"build_submitted":     BuildSubmitted,
		"build_started":       BuildStarted,
		"build_finished":      BuildFinished,
		"build_tagged":        BuildTagged,
		"collaborator_joined": CollaboratorJoined,
		"cron":                Cron,
		"images":              Images,
		"objects":             Objects,
		"variables":           Variables,
		"ssh_keys":            SSHKeys,
	}

	WebhookEvents = []WebhookEvent{
		BuildSubmitted,
		BuildStarted,
		BuildFinished,
		BuildTagged,
		CollaboratorJoined,
		Cron,
		Images,
		Objects,
		Variables,
		SSHKeys,
	}

	ErrUnknownEvent   = errors.New("unknown event")
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
			m[key] = append(m[key], string(buf))
			buf = buf[0:0]
		case '\n':
			m[key] = append(m[key], string(buf))

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

	for header, vals := range headers {
		buf.WriteString(header + ": " + strings.Join(vals, "; "))
	}
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

func WebhookFromContext(ctx context.Context) (*Webhook, bool) {
	w, ok := ctx.Value("webhook").(*Webhook)
	return w, ok
}

func UnmarshalWebhookEvents(names ...string) (WebhookEvent, error) {
	var events WebhookEvent

	for _, name := range names {
		ev, ok := webhookEventsMap[name]

		if !ok {
			return 0, errors.New("unknown webhook event: " + name)
		}
		events |= ev
	}
	return events, nil
}

func (e WebhookEvent) Has(mask WebhookEvent) bool { return (e & mask) == mask }

func (e WebhookEvent) Value() (driver.Value, error) {
	size := binary.Size(e)

	if size < 0 {
		return nil, errors.New("invalid webhook event")
	}

	buf := make([]byte, size)

	binary.PutVarint(buf, int64(e))
	return buf, nil
}

func (e *WebhookEvent) Scan(v interface{}) error {
	if v == nil {
		return nil
	}

	b, err := database.Scan(v)

	if err != nil {
		return errors.Err(err)
	}

	i, n := binary.Varint(b)

	if n < 0 {
		return errors.New("64 bit overflow")
	}

	(*e) = WebhookEvent(i)
	return nil
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
		w.Secret == "" &&
		!w.SSL &&
		w.Events == WebhookEvent(0) &&
		w.CreatedAt == time.Time{}
}

func (w *Webhook) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"user_id":      w.UserID,
		"author_id":    w.AuthorID,
		"namespace_id": w.NamespaceID,
		"payload_url":  w.PayloadURL,
		"secret":       w.Secret,
		"ssl":          w.SSL,
		"url":          addr + w.Endpoint(),
	}

	events := make([]string, 0, len(WebhookEvents))

	for _, ev := range WebhookEvents {
		if w.Events.Has(ev) {
			events = append(events, ev.String())
		}
	}

	json["events"] = events

	if !w.Namespace.IsZero() {
		json["namespace"] = w.Namespace.JSON(addr)
	}

	if w.LastDelivery != nil {
		json["last_response"] = map[string]interface{}{
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

func (s *WebhookStore) Create(authorId int64, payloadUrl *url.URL, secret string, ssl bool, events WebhookEvent, active bool) (*Webhook, error) {
	w := s.New()
	w.AuthorID = authorId
	w.PayloadURL = hookURL{URL: payloadUrl}
	w.Secret = secret
	w.SSL = ssl
	w.Events = events
	w.Active = active

	err := s.Store.Create(webhookTable, w)
	return w, errors.Err(err)
}

func (s *WebhookStore) Update(id int64, payloadUrl *url.URL, secret string, ssl bool, events WebhookEvent, active bool) error {
	q := query.Update(
		webhookTable,
		query.Set("payload_url", query.Arg(hookURL{URL:payloadUrl})),
		query.Set("secret", query.Arg(secret)),
		query.Set("ssl", query.Arg(ssl)),
		query.Set("events", query.Arg(events)),
		query.Set("active", query.Arg(active)),
		query.Where("id", "=", query.Arg(id)),
	)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
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
func (s *WebhookStore) createDelivery(hookId int64, deliveryId string, req *http.Request, resp *http.Response, dur time.Duration, derr error) error {
	var count int64

	q := query.Select(
		query.Count("webhook_id"),
		query.From(webhookDeliveryTable),
		query.Where("webhook_id", "=", query.Arg(hookId)),
		query.Where("delivery_id", "=", query.Arg(deliveryId)),
	)

	if err := s.DB.QueryRow(q.Build(), q.Args()...).Scan(&count); err != nil {
		return errors.Err(err)
	}

	redelivery := count > 0

	var (
		reqHeaders  string
		respHeaders string
		respStatus  int

		reqBody  []byte
		respBody []byte

		deliveryErr string
	)

	if derr != nil {
		deliveryErr = errors.Cause(derr).Error()
	} else {
		reqHeaders = encodeHeaders(req.Header)
		respHeaders = encodeHeaders(resp.Header)
		respStatus = resp.StatusCode

		reqBody, _ = io.ReadAll(req.Body)
		respBody, _ = io.ReadAll(resp.Body)
	}

	q = query.Insert(
		webhookDeliveryTable,
		query.Columns(
			"webhook_id",
			"delivery_id",
			"delivery_err",
			"redelivery",
			"request_headers",
			"request_body",
			"response_code",
			"response_headers",
			"response_body",
			"duration",
		),
		query.Values(
			hookId,
			deliveryId,
			sql.NullString{
				Valid:  derr != nil,
				String: deliveryErr,
			},
			redelivery,
			reqHeaders,
			string(reqBody),
			respStatus,
			respHeaders,
			string(respBody),
			dur,
		),
	)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

func (s *WebhookStore) realDeliver(w *Webhook, event WebhookEvent, r *bytes.Reader) (*http.Request, *http.Response, time.Duration, error) {
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

	if w.Secret != "" {
		var buf bytes.Buffer

		hmac := hmac.New(sha256.New, []byte(w.Secret))
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
			"Content-Type":       {"application/json", "charset=utf-8"},
			"User-Agent":         {"Djinn-CI-Hook"},
			"X-Djinn-CI-Event":   {event.String()},
			"X-Djinn-CI-Hook-ID": {strconv.FormatInt(w.ID, 10)},
		},
		Body: io.NopCloser(r),
	}

	if signature != "" {
		req.Header.Set("X-Djinn-CI-Signature", signature)
	}

	start := time.Now()

	resp, err := cli.Do(req)

	if err != nil {

		return nil, nil, 0, errors.Err(err)
	}
	return req, resp, time.Now().Sub(start), nil
}

func (s *WebhookStore) Redeliver(id int64, deliveryId string) error {
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
		query.Where("delivery_id", "=", query.Arg(deliveryId)),
	)

	if err := s.DB.QueryRow(q.Build(), q.Args()...).Scan(&headers, &body); err != nil {
		if err == sql.ErrNoRows {
			return database.ErrNotFound
		}
		return errors.Err(err)
	}

	hdr := decodeHeaders(headers)
	r := bytes.NewReader([]byte(body))

	event := webhookEventsMap[hdr["X-Djinn-CI-Event"][0]]

	req, resp, dur, derr := s.realDeliver(w, event, r)
	req.Body = io.NopCloser(strings.NewReader(body))

	return s.createDelivery(w.ID, deliveryId, req, resp, dur, derr)
}

func (s *WebhookStore) Deliver(event string, payload map[string]interface{}) error {
	ww, err := s.All()

	if err != nil {
		return errors.Err(err)
	}

	ev, ok := webhookEventsMap[event]

	if !ok {
		return ErrUnknownEvent
	}

	var buf bytes.Buffer

	json.NewEncoder(&buf).Encode(payload)

	r := bytes.NewReader(buf.Bytes())

	for _, w := range ww {
		if !w.Active {
			continue
		}

		if !w.Events.Has(ev) {
			continue
		}

		deliveryId := make([]byte, 16)
		rand.Read(deliveryId)

		req, resp, dur, derr := s.realDeliver(w, ev, r)

		r.Seek(0, io.SeekStart)

		if req != nil {
			req.Body = io.NopCloser(&buf)
		}

		if err := s.createDelivery(w.ID, hex.EncodeToString(deliveryId), req, resp, dur, derr); err != nil {
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
