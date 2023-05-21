package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"djinn-ci.com/auth"
	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/image"
	"djinn-ci.com/key"
	"djinn-ci.com/mail"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"
	"djinn-ci.com/runner"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/gorilla/mux"
)

type Handler struct {
	*server.Server

	Namespaces    namespace.Store
	Collaborators *database.Store[*namespace.Collaborator]
	Invites       namespace.InviteStore
	Webhooks      *namespace.WebhookStore
	Users         *database.Store[*auth.User]
	Builds        *build.Store
	Images        *image.Store
	Keys          *key.Store
	Objects       *object.Store
	Variables     *variable.Store
}

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server: srv,
		Namespaces: namespace.Store{
			Store: namespace.NewStore(srv.DB),
		},
		Collaborators: namespace.NewCollaboratorStore(srv.DB),
		Invites:       namespace.NewInviteStore(srv.DB),
		Webhooks: &namespace.WebhookStore{
			Store:  namespace.NewWebhookStore(srv.DB),
			AESGCM: srv.AESGCM,
		},
		Users: user.NewStore(srv.DB),
		Builds: &build.Store{
			Store: build.NewStore(srv.DB),
		},
		Keys:      &key.Store{Store: key.NewStore(srv.DB)},
		Objects:   &object.Store{Store: object.NewStore(srv.DB)},
		Images:    &image.Store{Store: image.NewStore(srv.DB)},
		Variables: &variable.Store{Store: variable.NewStore(srv.DB)},
	}
}

func (h *Handler) loadLastBuild(ctx context.Context, nn []*namespace.Namespace) error {
	if len(nn) == 0 {
		return nil
	}

	ids := database.Map[*namespace.Namespace, any](nn, func(n *namespace.Namespace) any {
		return n.ID
	})

	bb, err := h.Builds.Distinct(
		ctx,
		[]string{"namespace_id"},
		[]string{"*"},
		query.Where("namespace_id", "IN", query.List(ids...)),
		query.OrderDesc("namespace_id", "created_at"),
	)

	if err != nil {
		return errors.Err(err)
	}

	buildtab := make(map[int64]*build.Build)

	for _, b := range bb {
		if _, ok := buildtab[b.NamespaceID.Elem]; !ok {
			buildtab[b.NamespaceID.Elem] = b
		}
	}

	if err := build.LoadRelations(ctx, h.DB, bb...); err != nil {
		return errors.Err(err)
	}

	for _, n := range nn {
		if b, ok := buildtab[n.ID]; ok && n.Build == nil {
			n.Build = b
		}
	}
	return nil
}

type HandlerFunc func(*auth.User, *namespace.Namespace, http.ResponseWriter, *http.Request)

func (h *Handler) Namespace(fn HandlerFunc) auth.HandlerFunc {
	return func(u *auth.User, w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ctx := r.Context()

		owner, ok, err := h.Users.Get(ctx, user.WhereUsername(vars["username"]))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get namespace owner"))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		path := strings.TrimSuffix(vars["namespace"], "/")

		n, ok, err := h.Namespaces.Get(
			ctx,
			query.Where("user_id", "=", query.Arg(owner.ID)),
			query.Where("path", "=", query.Arg(path)),
		)

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get namespace"))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		base := webutil.BasePath(r.URL.Path)

		// Must be namespace owner to see these.
		if base == "collaborators" || base == "invites" {
			if u.ID != owner.ID {
				h.NotFound(w, r)
				return
			}
		}

		n.User = owner

		if n.ParentID.Valid {
			parent, _, err := h.Namespaces.Get(ctx, query.Where("id", "=", query.Arg(n.ParentID)))

			if err != nil {
				h.Error(w, r, errors.Wrap(err, "Failed to get parent"))
				return
			}

			parent.User, _, err = h.Users.Get(ctx, user.WhereID(parent.UserID))

			if err != nil {
				h.Error(w, r, errors.Wrap(err, "Failed to get parent"))
				return
			}
			n.Parent = parent
		}

		if err := n.HasAccess(ctx, h.DB, u); err != nil {
			if !errors.Is(err, database.ErrPermission) {
				h.Error(w, r, errors.Wrap(err, "Failed to get namespace"))
				return
			}

			h.NotFound(w, r)
			return
		}

		if n.Visibility != namespace.Public {
			if !u.Has("namespace:read") {
				h.NotFound(w, r)
				return
			}
		}
		fn(u, n, w, r)
	}
}

func (h *Handler) Index(u *auth.User, r *http.Request) (*database.Paginator[*namespace.Namespace], error) {
	ctx := r.Context()

	p, err := h.Namespaces.Index(ctx, r.URL.Query(), namespace.SharedWith(u.ID))

	if err != nil {
		return nil, errors.Err(err)
	}

	mm := make([]database.Model, 0, len(p.Items))

	for _, n := range p.Items {
		mm = append(mm, n)
	}

	if err := user.Loader(h.DB).Load(ctx, "user_id", "id", mm...); err != nil {
		return nil, errors.Err(err)
	}

	if err := h.loadLastBuild(ctx, p.Items); err != nil {
		return nil, errors.Err(err)
	}
	return p, nil
}

func (h *Handler) Store(u *auth.User, r *http.Request) (*namespace.Namespace, *Form, error) {
	f := Form{
		Pool: h.DB,
		User: u,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		return nil, &f, errors.Err(err)
	}

	ctx := r.Context()

	n, err := h.Namespaces.Create(ctx, &namespace.Params{
		User:        u,
		Parent:      f.Parent,
		Name:        f.Name,
		Description: f.Description,
		Visibility:  f.Visibility,
	})

	if err != nil {
		return nil, &f, errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &namespace.Event{
		Namespace: n,
		Action:    "created",
	})
	return n, &f, nil
}

func (h *Handler) Badge(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	username := vars["username"]
	path := strings.TrimSuffix(vars["namespace"], "/")

	n, ok, err := h.Namespaces.Get(
		ctx,
		query.Where("user_id", "=", user.Select(
			query.Columns("id"),
			query.Where("username", "=", query.Arg(username)),
		)),
		query.Where("path", "=", query.Arg(path)),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get namespace badge"))
		return
	}

	if !ok {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, badgeUnknown)
		return
	}

	b, ok, err := h.Builds.Get(
		ctx,
		query.Where("namespace_id", "=", query.Arg(n.ID)),
		query.OrderDesc("created_at"),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get namespace badge"))
		return
	}

	if !ok {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, badgeUnknown)
		return
	}

	if n.Visibility == namespace.Internal || n.Visibility == namespace.Private {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, badgeUnknown)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.WriteHeader(http.StatusOK)

	switch b.Status {
	case runner.Queued:
		io.WriteString(w, badgeQueued)
	case runner.Running:
		io.WriteString(w, badgeRunning)
	case runner.Passed:
		io.WriteString(w, badgePassed)
	case runner.PassedWithFailures:
		io.WriteString(w, badgePassedWithFailures)
	case runner.Failed:
		io.WriteString(w, badgeFailed)
	case runner.Killed:
		io.WriteString(w, badgeKilled)
	case runner.TimedOut:
		io.WriteString(w, badgeTimedOut)
	default:
		io.WriteString(w, badgeUnknown)
	}
}

func (h *Handler) Update(u *auth.User, n *namespace.Namespace, r *http.Request) (*namespace.Namespace, *Form, error) {
	f := Form{
		Pool:      h.DB,
		User:      u,
		Namespace: n,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		return nil, &f, errors.Err(err)
	}

	ctx := r.Context()

	n.Description = f.Description
	n.Visibility = f.Visibility

	if err := h.Namespaces.Update(ctx, n); err != nil {
		return nil, &f, errors.Err(err)
	}

	n.User = u

	h.Queues.Produce(ctx, "events", &namespace.Event{
		Namespace: n,
		Action:    "updated",
	})
	return n, &f, nil
}

func (h *Handler) DestroyCollaborator(u *auth.User, n *namespace.Namespace, r *http.Request) error {
	if u.ID != n.UserID {
		return database.ErrPermission
	}

	ctx := r.Context()

	c, ok, err := h.Collaborators.Get(
		ctx,
		query.Where("namespace_id", "=", query.Arg(n.ID)),
		query.Where("user_id", "=", user.Select(
			query.Columns("id"),
			user.WhereHandle(mux.Vars(r)["collaborator"]),
		)),
	)

	if err != nil {
		return errors.Err(err)
	}

	if !ok {
		return database.ErrNoRows
	}

	if err := h.Collaborators.Delete(ctx, c); err != nil {
		return errors.Err(err)
	}

	return nil
}

func (h *Handler) Destroy(ctx context.Context, n *namespace.Namespace) error {
	if err := h.Namespaces.Delete(ctx, n); err != nil {
		return errors.Err(err)
	}

	if err := user.Loader(h.DB).Load(ctx, "user_id", "id", n); err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &namespace.Event{
		Namespace: n,
		Action:    "deleted",
	})
	return nil
}

type InviteHandler struct {
	*server.Server

	Namespaces    *database.Store[*namespace.Namespace]
	Invites       namespace.InviteStore
	Collaborators *database.Store[*namespace.Collaborator]
	Users         *database.Store[*auth.User]
}

func NewInviteHandler(srv *server.Server) *InviteHandler {
	return &InviteHandler{
		Server:        srv,
		Namespaces:    namespace.NewStore(srv.DB),
		Invites:       namespace.NewInviteStore(srv.DB),
		Collaborators: namespace.NewCollaboratorStore(srv.DB),
		Users:         user.NewStore(srv.DB),
	}
}

var inviteMail = `%s has invited you to be a collaborator in %s. You can accept this invite via
your Invites list,

    %s/invites`

func (h *InviteHandler) Store(u *auth.User, n *namespace.Namespace, r *http.Request) (*namespace.Invite, *InviteForm, error) {
	var f InviteForm

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		return nil, &f, errors.Err(err)
	}

	ctx := r.Context()

	i, err := h.Invites.Create(ctx, &namespace.InviteParams{
		Handle:    f.Handle,
		Inviter:   u,
		Namespace: n,
	})

	if err != nil {
		if errors.Is(err, namespace.ErrSelfInvite) {
			return nil, &f, err
		}
		if errors.Is(err, namespace.ErrInviteSent) {
			return nil, &f, err
		}
		if errors.Is(err, namespace.ErrCollaborator) {
			return nil, &f, err
		}
		if errors.Is(err, database.ErrNoRows) {
			return nil, &f, err
		}
		return nil, &f, errors.Err(err)
	}

	if h.SMTP.Client != nil {
		h.Queues.Produce(ctx, "email", &mail.Mail{
			From:    h.SMTP.From,
			To:      []string{i.Invitee.Email},
			Subject: fmt.Sprintf("Djinn CI - %s invited you to %s", i.Inviter.Username, n.Path),
			Body:    fmt.Sprintf(inviteMail, i.Inviter.Username, n.Path, webutil.BaseAddress(r)),
		})
	}

	h.Queues.Produce(ctx, "events", &namespace.InviteEvent{
		Action:    "sent",
		Namespace: n,
		Inviter:   i.Inviter,
		Invitee:   i.Invitee,
	})
	return i, &f, nil
}

func (h *InviteHandler) Accept(ctx context.Context, u *auth.User, i *namespace.Invite) (*namespace.Namespace, *auth.User, *auth.User, error) {
	n, inviter, invitee, err := h.Invites.Accept(ctx, u, i)

	if err != nil {
		return nil, nil, nil, errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &namespace.InviteEvent{
		Action:    "accepted",
		Namespace: n,
		Invitee:   invitee,
	})
	return n, inviter, invitee, nil
}

func (h *InviteHandler) Destroy(ctx context.Context, u *auth.User, i *namespace.Invite) error {
	if u.ID != i.InviteeID && u.ID != i.InviterID {
		return database.ErrNoRows
	}

	if err := h.Invites.Delete(ctx, i); err != nil {
		return errors.Err(err)
	}

	n, _, err := h.Namespaces.Get(ctx, query.Where("id", "=", query.Arg(i.NamespaceID)))

	if err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &namespace.InviteEvent{
		Action:    "rejected",
		Namespace: n,
		Invitee:   u,
	})
	return nil
}

type WebhookHandler struct {
	*server.Server

	Webhooks *namespace.WebhookStore
	Users    *database.Store[*auth.User]
}

func NewWebhookHandler(srv *server.Server) *WebhookHandler {
	return &WebhookHandler{
		Server: srv,
		Webhooks: &namespace.WebhookStore{
			Store:  namespace.NewWebhookStore(srv.DB),
			AESGCM: srv.AESGCM,
		},
		Users: user.NewStore(srv.DB),
	}
}

func (h *WebhookHandler) Store(u *auth.User, n *namespace.Namespace, r *http.Request) (*namespace.Webhook, *WebhookForm, error) {
	f := WebhookForm{
		Pool: h.DB,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		return nil, &f, errors.Err(err)
	}

	url, _ := url.Parse(f.PayloadURL)
	events, _ := event.UnmarshalType(f.Events...)

	wh, err := h.Webhooks.Create(r.Context(), &namespace.WebhookParams{
		User:       u,
		Namespace:  n,
		PayloadURL: url,
		Secret:     f.Secret,
		SSL:        f.SSL,
		Events:     events,
		Active:     f.Active,
	})

	if err != nil {
		return nil, &f, errors.Err(err)
	}
	return wh, &f, nil
}

func (h *WebhookHandler) Update(wh *namespace.Webhook, r *http.Request) (*namespace.Webhook, *WebhookForm, error) {
	f := WebhookForm{
		Pool:    h.DB,
		Webhook: wh,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		return nil, &f, errors.Err(err)
	}

	if f.Secret == "" {
		if f.RemoveSecret {
			wh.Secret = nil
		}

		if !f.RemoveSecret && wh.Secret != nil {
			secret, err := h.AESGCM.Decrypt(wh.Secret)

			if err != nil {
				return nil, &f, errors.Err(err)
			}
			wh.Secret = secret
		}
	}

	url, _ := url.Parse(f.PayloadURL)
	events, _ := event.UnmarshalType(f.Events...)

	wh.PayloadURL = namespace.WebhookURL(url)
	wh.SSL = f.SSL
	wh.Events = events
	wh.Active = f.Active

	if err := h.Webhooks.Update(r.Context(), wh); err != nil {
		return nil, &f, errors.Err(err)
	}
	return wh, &f, nil
}
