package handler

import (
	"net/http"
	"net/url"

	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/user"
	"djinn-ci.com/web"

	"github.com/andrewpillar/webutil"
)

type Webhook struct {
	web.Handler
}

func NewWebhook(h web.Handler) Webhook {
	return Webhook{
		Handler: h,
	}
}

func (h Webhook) StoreModel(r *http.Request) (*namespace.Webhook, namespace.WebhookForm, error) {
	ctx := r.Context()

	f := namespace.WebhookForm{}

	u, ok := user.FromContext(ctx)

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	n, ok := namespace.FromContext(ctx)

	if !ok {
		return nil, f, errors.New("no namespace in request context")
	}

	webhooks := namespace.NewWebhookStore(h.DB, n)

	f.Webhooks = webhooks

	if err := webutil.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	url, _ := url.Parse(f.PayloadURL)

	if len(f.Events) == 0 {
		for _, ev := range namespace.WebhookEvents {
			f.Events = append(f.Events, ev.String())
		}
	}

	events, _ := namespace.UnmarshalWebhookEvents(f.Events...)

	w, err := webhooks.Create(u.ID, url, f.Secret, f.SSL, events, f.Active)

	if err != nil {
		return nil, f, errors.Err(err)
	}
	return w, f, nil
}
