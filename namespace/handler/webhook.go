package handler

import (
	"net/http"
	"net/url"

	"djinn-ci.com/crypto"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/namespace"
	"djinn-ci.com/user"
	"djinn-ci.com/web"

	"github.com/andrewpillar/webutil"
)

type Webhook struct {
	web.Handler

	block *crypto.Block
}

func NewWebhook(h web.Handler, block *crypto.Block) Webhook {
	return Webhook{
		Handler: h,
		block:   block,
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

	webhooks := namespace.NewWebhookStoreWithBlock(h.DB, h.block, n, u)

	f.Webhooks = webhooks

	if err := webutil.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	url, _ := url.Parse(f.PayloadURL)

	if len(f.Events) == 0 {
		for _, ev := range event.Types {
			f.Events = append(f.Events, ev.String())
		}
	}

	events, _ := event.UnmarshalType(f.Events...)

	w, err := webhooks.Create(u.ID, url, f.Secret, f.SSL, events, f.Active)

	if err != nil {
		return nil, f, errors.Err(err)
	}
	return w, f, nil
}

func (h Webhook) UpdateModel(r *http.Request) (*namespace.Webhook, namespace.WebhookForm, error) {
	ctx := r.Context()

	f := namespace.WebhookForm{}

	u, ok := user.FromContext(ctx)

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	w, ok := namespace.WebhookFromContext(ctx)

	if !ok {
		return nil, f, errors.New("no webhook in request context")
	}

	n, ok := namespace.FromContext(ctx)

	if !ok {
		return nil, f, errors.New("no namespace in request context")
	}

	webhooks := namespace.NewWebhookStoreWithBlock(h.DB, h.block, n, u)

	f.Webhooks = webhooks
	f.Webhook = w

	if err := webutil.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	url, _ := url.Parse(f.PayloadURL)

	events, _ := event.UnmarshalType(f.Events...)

	secret := f.Secret

	if !f.RemoveSecret && secret == "" {
		secret0, err := h.block.Decrypt(w.Secret)

		if err != nil {
			return nil, f, errors.Err(err)
		}
		secret = string(secret0)
	}

	if err := webhooks.Update(w.ID, url, secret, f.SSL, events, f.Active); err != nil {
		return nil, f, errors.Err(err)
	}

	w.PayloadURL = namespace.WebhookURL(url)
	w.SSL = f.SSL
	w.Events = events
	w.Active = f.Active

	return w, f, nil
}
