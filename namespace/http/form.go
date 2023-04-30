package http

import (
	"context"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"
)

type Form struct {
	Pool      *database.Pool       `json:"-" schema:"-"`
	User      *auth.User           `json:"-" schema:"-"`
	Namespace *namespace.Namespace `json:"-" schema:"-"`

	Parent      string
	Name        string
	Description string
	Visibility  namespace.Visibility
}

var _ webutil.Form = (*Form)(nil)

func (f *Form) Fields() map[string]string {
	return map[string]string{
		"name":        f.Name,
		"description": f.Description,
	}
}

var reName = regexp.MustCompile("^[a-zA-Z0-9]+$")

func (f *Form) Validate(ctx context.Context) error {
	var v webutil.Validator

	v.WrapError(func(name string, err error) error {
		first := rune(name[0])

		name = string(unicode.ToUpper(first)) + name[1:]

		if _, ok := err.(webutil.MatchError); ok {
			err = errors.New("can only have letters, and numbers")
		}
		return webutil.WrapFieldError(name, err)
	})

	if f.Namespace != nil {
		if f.Name == "" {
			f.Name = f.Namespace.Name
		}
		if f.Description == "" {
			f.Description = f.Namespace.Description
		}
	}

	v.Add("name", f.Name, webutil.FieldRequired)
	v.Add("name", f.Name, webutil.FieldLen(3, 32))
	v.Add("name", f.Name, webutil.FieldMatches(reName))
	v.Add("name", f.Name, func(ctx context.Context, val any) error {
		name := val.(string)

		if f.Namespace == nil || (f.Namespace != nil && name != f.Namespace.Name) {
			_, ok, err := namespace.NewStore(f.Pool).Get(
				ctx,
				query.Where("user_id", "=", query.Arg(f.User.ID)),
				query.Where("name", "=", query.Arg(name)),
			)

			if err != nil {
				return err
			}

			if ok {
				return webutil.ErrFieldExists
			}
		}
		return nil
	})

	v.Add("description", f.Description, webutil.FieldMaxLen(255))

	errs := v.Validate(ctx)

	return errs.Err()
}

type InviteForm struct {
	Handle string
}

var _ webutil.Form = (*InviteForm)(nil)

func (f *InviteForm) Fields() map[string]string {
	return map[string]string{
		"handle": f.Handle,
	}
}

func (f *InviteForm) Validate(ctx context.Context) error {
	var v webutil.Validator

	v.WrapError(func(name string, err error) error {
		return webutil.WrapFieldError("Username or email", err)
	})

	v.Add("handle", f.Handle, webutil.FieldRequired)

	errs := v.Validate(ctx)

	return errs.Err()
}

type WebhookForm struct {
	Pool    *database.Pool     `json:"-" schema:"-"`
	Webhook *namespace.Webhook `json:"-" schema:"-"`

	PayloadURL   string `json:"payload_url" schema:"payload_url"`
	Secret       string
	RemoveSecret bool `json:"remove_secret" schema:"remove_secret"`
	SSL          bool
	Active       bool
	Events       []string
}

var _ webutil.Form = (*WebhookForm)(nil)

func (f *WebhookForm) Fields() map[string]string {
	return map[string]string{
		"payload_url": f.PayloadURL,
		"events":      strings.Join(f.Events, " "),
	}
}

func (f *WebhookForm) Validate(ctx context.Context) error {
	var v webutil.Validator

	v.WrapError(func(name string, err error) error {
		if name == "payload_url" {
			name = "Payload URL"
		}
		return webutil.WrapFieldError(name, err)
	})

	if f.Webhook != nil {
		if f.PayloadURL == "" {
			f.PayloadURL = f.Webhook.PayloadURL.String()
		}
	}

	v.Add("payload_url", f.PayloadURL, webutil.FieldRequired)
	v.Add("payload_url", f.PayloadURL, func(ctx context.Context, val any) error {
		if f.Webhook == nil || (f.Webhook != nil && f.Webhook.PayloadURL.String() != f.PayloadURL) {
			_, ok, err := namespace.NewWebhookStore(f.Pool).Get(
				ctx, query.Where("payload_url", "=", query.Arg(val)),
			)

			if err != nil {
				return err
			}
			if ok {
				return webutil.ErrFieldExists
			}
		}
		return nil
	})
	v.Add("payload_url", f.PayloadURL, func(ctx context.Context, val any) error {
		localhosts := map[string]struct{}{
			"localhost": {},
			"127.0.0.1": {},
			"::1":       {},
		}

		s := val.(string)

		url, err := url.Parse(s)

		if err != nil {
			return err
		}

		if url.Scheme != "http" && url.Scheme != "https" {
			return errors.New("missing http or https scheme")
		}

		if url.Host == "" {
			return errors.New("missing host")
		}

		if _, _, err := net.SplitHostPort(url.Host); err != nil {
			if addrErr, ok := err.(*net.AddrError); ok {
				if addrErr.Err == "missing port in address" {
					goto cont
				}
			}
			return err
		}

	cont:
		if _, ok := localhosts[url.Host]; ok {
			return errors.New("cannot be localhost")
		}

		if f.SSL {
			if url.Scheme == "http" {
				return errors.New("cannot use SSL for plain http URL")
			}
		}
		return nil
	})

	errs := v.Validate(ctx)

	return errs.Err()
}
