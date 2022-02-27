package http

import (
	"net"
	"net/url"
	"regexp"
	"strings"

	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

type Form struct {
	Parent      string
	Name        string
	Description string
	Visibility  namespace.Visibility
}

var _ webutil.Form = (*Form)(nil)

func (f Form) Fields() map[string]string {
	return map[string]string{
		"name":        f.Name,
		"description": f.Description,
	}
}

type Validator struct {
	UserID     int64
	Namespaces namespace.Store
	Namespace  *namespace.Namespace
	Form       Form
}

var (
	_ webutil.Validator = (*Validator)(nil)

	rename = regexp.MustCompile("^[a-zA-Z0-9]+$")
)

func (v Validator) Validate(errs webutil.ValidationErrors) {
	if v.Namespace != nil {
		if v.Form.Name == "" {
			v.Form.Name = v.Namespace.Name
		}
		if v.Form.Description == "" {
			v.Form.Description = v.Namespace.Description
		}
	}

	if v.Form.Name == "" {
		errs.Add("name", webutil.ErrFieldRequired("Name"))
	}

	if len(v.Form.Name) < 3 || len(v.Form.Name) > 32 {
		errs.Add("name", errors.New("Name must be between 3 and 32 characters in length"))
	}

	if !rename.Match([]byte(v.Form.Name)) {
		errs.Add("name", errors.New("Name can only contain letters and numbers"))
	}

	if v.Namespace == nil || (v.Namespace != nil && v.Form.Name != v.Namespace.Name) {
		opts := []query.Option{
			query.Where("user_id", "=", query.Arg(v.UserID)),
			query.Where("name", "=", query.Arg(v.Form.Name)),
		}

		_, ok, err := v.Namespaces.Get(opts...)

		if err != nil {
			errs.Add("fatal", err)
			return
		}

		if ok {
			errs.Add("name", webutil.ErrFieldExists("Name"))
		}
	}

	if len(v.Form.Description) > 255 {
		errs.Add("description", errors.New("Description must be shorter than 255 characters in length"))
	}
}

type InviteForm struct {
	Handle string
}

var _ webutil.Form = (*InviteForm)(nil)

func (f InviteForm) Fields() map[string]string {
	return map[string]string{
		"handle": f.Handle,
	}
}

type WebhookForm struct {
	PayloadURL   string `json:"payload_url" schema:"payload_url"`
	Secret       string
	RemoveSecret bool `json:"remove_secret" schema:"remove_secret"`
	SSL          bool
	Active       bool
	Events       []string
}

var _ webutil.Form = (*WebhookForm)(nil)

func (f WebhookForm) Fields() map[string]string {
	return map[string]string{
		"payload_url": f.PayloadURL,
		"events":      strings.Join(f.Events, " "),
	}
}

type WebhookValidator struct {
	Webhooks *namespace.WebhookStore
	Webhook  *namespace.Webhook
	Form     WebhookForm
}

var (
	_ webutil.Validator = (*WebhookValidator)(nil)

	localhosts = map[string]struct{}{
		"localhost": {},
		"127.0.0.1": {},
		"::1":       {},
	}
)

func (v WebhookValidator) Validate(errs webutil.ValidationErrors) {
	if v.Form.PayloadURL == "" {
		errs.Add("payload_url", webutil.ErrFieldRequired("Payload URL"))
	}

	v.Form.PayloadURL = strings.ToLower(v.Form.PayloadURL)

	if v.Webhook == nil || (v.Webhook != nil && v.Webhook.PayloadURL.String() != v.Form.PayloadURL) {
		_, ok, err := v.Webhooks.Get(query.Where("payload_url", "=", query.Arg(v.Form.PayloadURL)))

		if err != nil {
			errs.Add("webhook", err)
			return
		}

		if ok {
			errs.Add("payload_url", webutil.ErrFieldExists("Payload URL"))
		}
	}

	url, err := url.Parse(v.Form.PayloadURL)

	if err != nil {
		errs.Add("payload_url", errors.New("Invalid payload URL"))
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		errs.Add("payload_url", errors.New("Invalid payload URL"))
	}

	host, _, _ := net.SplitHostPort(url.Host)

	if host == "" {
		errs.Add("payload_url", errors.New("Invalid payload URL, missing host"))
	}

	if _, ok := localhosts[host]; ok {
		errs.Add("payload_url", errors.New("Invalid payload URL, cannot be localhost"))
	}

	if v.Form.SSL {
		if url.Scheme == "http" {
			errs.Add("payload_url", errors.New("Cannot use SSL for a plain http URL"))
		}
	}
}
