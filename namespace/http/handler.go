package http

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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
	userhttp "djinn-ci.com/user/http"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/mux"
)

type Handler struct {
	*server.Server

	Namespaces    namespace.Store
	Collaborators namespace.CollaboratorStore
	Invites       namespace.InviteStore
	Webhooks      *namespace.WebhookStore
	Users         user.Store
	Builds        *build.Store
	Images        image.Store
	Keys          *key.Store
	Objects       object.Store
	Variables     variable.Store
}

type HandlerFunc func(*user.User, *namespace.Namespace, http.ResponseWriter, *http.Request)

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server:        srv,
		Namespaces:    namespace.Store{Pool: srv.DB},
		Collaborators: namespace.CollaboratorStore{Pool: srv.DB},
		Invites:       namespace.InviteStore{Pool: srv.DB},
		Webhooks: &namespace.WebhookStore{
			Pool:   srv.DB,
			AESGCM: srv.AESGCM,
		},
		Users:     user.Store{Pool: srv.DB},
		Builds:    &build.Store{Pool: srv.DB},
		Keys:      &key.Store{Pool: srv.DB},
		Objects:   object.Store{Pool: srv.DB},
		Images:    image.Store{Pool: srv.DB},
		Variables: variable.Store{Pool: srv.DB},
	}
}

func (h *Handler) loadParent(n *namespace.Namespace) error {
	if err := h.Users.Load("user_id", "id", n); err != nil {
		return errors.Err(err)
	}

	if !n.ParentID.Valid {
		return nil
	}

	p, _, err := h.Namespaces.Get(query.Where("id", "=", query.Arg(n.ParentID)))

	if err != nil {
		return errors.Err(err)
	}

	if err := h.Users.Load("user_id", "id", p); err != nil {
		return errors.Err(err)
	}

	n.Parent = p
	return nil
}

func (h *Handler) loadLastBuild(nn []*namespace.Namespace) error {
	mm := make([]database.Model, 0, len(nn))

	for _, n := range nn {
		mm = append(mm, n)
	}

	bb, err := h.Builds.Distinct(
		[]string{"namespace_id"},
		query.Where("namespace_id", "IN", database.List(database.Values("id", mm)...)),
		query.OrderDesc("namespace_id", "created_at"),
	)

	if err != nil {
		return errors.Err(err)
	}

	builds := make(map[int64]*build.Build)

	for _, b := range bb {
		if _, ok := builds[b.NamespaceID.Int64]; !ok {
			builds[b.NamespaceID.Int64] = b
		}
	}

	if err := build.LoadRelations(h.DB, bb...); err != nil {
		return errors.Err(err)
	}

	for _, n := range nn {
		if b, ok := builds[n.ID]; ok && n.Build == nil {
			n.Build = b
		}
	}
	return nil
}

func (h *Handler) WithNamespace(fn HandlerFunc) userhttp.HandlerFunc {
	ownerPaths := map[string]struct{}{
		"collaborators": {},
		"invites":       {},
	}

	return func(u *user.User, w http.ResponseWriter, r *http.Request) {
		var ok bool

		switch r.Method {
		case "GET":
			_, ok = u.Permissions["namespace:read"]
		case "POST", "PATCH":
			_, ok = u.Permissions["namespace:write"]
		case "DELETE":
			_, ok = u.Permissions["namespace:delete"]
		}

		vars := mux.Vars(r)

		owner, ok, err := h.Users.Get(query.Where("username", "=", query.Arg(vars["username"])))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		path := strings.TrimSuffix(vars["namespace"], "/")

		n, ok, err := h.Namespaces.Get(
			query.Where("user_id", "=", query.Arg(owner.ID)),
			query.Where("path", "=", query.Arg(path)),
		)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if !ok {
			h.NotFound(w, r)
			return
		}

		n.User = owner

		base := webutil.BasePath(r.URL.Path)

		if _, ok := ownerPaths[base]; ok {
			if u.ID != owner.ID {
				h.NotFound(w, r)
				return
			}
		}

		if r.Method == "POST" && base == "webhooks" {
			goto checkAccess
		}

		if r.Method == "POST" || r.Method == "PATCH" || r.Method == "DELETE" {
			if u.ID != owner.ID {
				h.NotFound(w, r)
				return
			}
		}

checkAccess:
		if err := n.HasAccess(h.DB, u.ID); err != nil {
			if !errors.Is(err, namespace.ErrPermission) {
				h.InternalServerError(w, r, errors.Err(err))
				return
			}

			h.NotFound(w, r)
			return
		}

		if err := h.loadParent(n); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}
		fn(u, n, w, r)
	}
}

func (h *Handler) CanAccessResource(name string, get database.GetModelFunc, u *user.User, r *http.Request) (bool, error) {
	var ok bool

	switch r.Method {
	case "GET":
		_, ok = u.Permissions[name+":read"]
	case "POST", "PATCH":
		_, ok = u.Permissions[name+":write"]
	case "DELETE":
		_, ok = u.Permissions[name+":delete"]
	}

	if !ok {
		return false, nil
	}

	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars[name], 10, 64)

	m, ok, err := get(id)

	if err != nil {
		return false, errors.Err(err)
	}

	if !ok {
		return false, nil
	}

	vals := m.Values()

	namespaceId, _ := vals["namespace_id"].(sql.NullInt64)

	if !namespaceId.Valid {
		userId, _ := vals["user_id"].(int64)

		return u.ID == userId, nil
	}

	root, _, err := h.Namespaces.Get(
		query.Where("root_id", "=", namespace.SelectRootID(namespaceId.Int64)),
		query.Where("id", "=", namespace.SelectRootID(namespaceId.Int64)),
	)

	if err != nil {
		return false, errors.Err(err)
	}

	if r.Method == "GET" {
		if err := root.HasAccess(h.DB, u.ID); err != nil {
			if errors.Is(err, namespace.ErrPermission) {
				return false, nil
			}
			return false, errors.Err(err)
		}
	}

	if err := root.IsCollaborator(h.DB, u.ID); err != nil {
		if errors.Is(err, namespace.ErrPermission) {
			return false, nil
		}
		return false, errors.Err(err)
	}
	return true, nil
}

func (h *Handler) IndexWithRelations(u *user.User, r *http.Request) ([]*namespace.Namespace, database.Paginator, error) {
	nn, paginator, err := h.Namespaces.Index(r.URL.Query(), namespace.SharedWith(u.ID))

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	mm := make([]database.Model, 0, len(nn))

	for _, n := range nn {
		if u.ID != n.UserID {
			mm = append(mm, n)
			continue
		}
		n.Bind(u)
	}

	if len(mm) > 0 {
		if err := h.Users.Load("user_id", "id", mm...); err != nil {
			return nil, paginator, errors.Err(err)
		}
	}

	if err := h.loadLastBuild(nn); err != nil {
		return nil, paginator, errors.Err(err)
	}
	return nn, paginator, nil
}

func (h *Handler) StoreModel(u *user.User, r *http.Request) (*namespace.Namespace, Form, error) {
	var (
		f     Form
		verrs webutil.ValidationErrors
	)

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		if verrs0, ok := err.(webutil.ValidationErrors); ok {
			verrs = verrs0
			goto validate
		}
		return nil, f, errors.Err(err)
	}

validate:
	v := Validator{
		UserID:     u.ID,
		Namespaces: h.Namespaces,
		Form:       f,
	}

	if err := webutil.Validate(v); err != nil {
		err.(webutil.ValidationErrors).Merge(verrs)

		return nil, f, errors.Err(err)
	}

	n, err := h.Namespaces.Create(namespace.Params{
		UserID:      u.ID,
		Parent:      f.Parent,
		Name:        f.Name,
		Description: f.Description,
		Visibility:  f.Visibility,
	})

	if err != nil {
		return nil, f, errors.Err(err)
	}

	n.User = u

	h.Queues.Produce(r.Context(), "events", &namespace.Event{
		Namespace: n,
		Action:    "created",
	})
	return n, f, nil
}

func (h *Handler) Badge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	owner, ok, err := h.Users.Get(query.Where("username", "=", query.Arg(vars["username"])))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	path := strings.TrimSuffix(vars["namespace"], "/")

	n, ok, err := h.Namespaces.Get(
		query.Where("user_id", "=", query.Arg(owner.ID)),
		query.Where("path", "=", query.Arg(path)),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
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

	b, ok, err := h.Builds.Get(
		query.Where("namespace_id", "=", query.Arg(n.ID)),
		query.OrderDesc("created_at"),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
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

func (h *Handler) UpdateModel(n *namespace.Namespace, r *http.Request) (*namespace.Namespace, Form, error) {
	var (
		f     Form
		verrs webutil.ValidationErrors
	)

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		if verrs0, ok := err.(webutil.ValidationErrors); ok {
			verrs = verrs0
			goto validate
		}
		return nil, f, errors.Err(err)
	}

validate:
	v := Validator{
		UserID:     n.UserID,
		Namespaces: h.Namespaces,
		Namespace:  n,
		Form:       f,
	}

	if err := webutil.Validate(v); err != nil {
		err.(webutil.ValidationErrors).Merge(verrs)

		return nil, f, errors.Err(err)
	}

	params := namespace.Params{
		Description: f.Description,
		Visibility:  f.Visibility,
	}

	if err := h.Namespaces.Update(n.ID, params); err != nil {
		return nil, f, errors.Err(err)
	}

	n.Description = f.Description
	n.Visibility = f.Visibility

	if err := h.Users.Load("user_id", "id", n); err != nil {
		return nil, f, errors.Err(err)
	}

	h.Queues.Produce(r.Context(), "events", &namespace.Event{
		Namespace: n,
		Action:    "updated",
	})
	return n, f, nil
}

func (h *Handler) deleteCollab(u *user.User, n *namespace.Namespace, r *http.Request) error {
	if u.ID != n.UserID {
		return namespace.ErrPermission
	}

	u2, ok, err := h.Users.Get(user.WhereHandle(mux.Vars(r)["collaborator"]))

	if err != nil {
		return errors.Err(err)
	}

	if !ok {
		return database.ErrNotFound
	}

	c, ok, err := h.Collaborators.Get(
		query.Where("namespace_id", "=", query.Arg(n.ID)),
		query.Where("user_id", "=", query.Arg(u2.ID)),
	)

	if err != nil {
		return errors.Err(err)
	}

	if !ok {
		return database.ErrNotFound
	}

	chowns := []namespace.ChownFunc{
		image.Chown,
		key.Chown,
		object.Chown,
		variable.Chown,
	}

	if err := h.Collaborators.Delete(n.ID, c.UserID, chowns...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (h *Handler) DeleteModel(ctx context.Context, n *namespace.Namespace) error {
	if err := h.Namespaces.Delete(n.ID); err != nil {
		return errors.Err(err)
	}

	if err := h.Users.Load("user_id", "id", n); err != nil {
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

	Namespaces    namespace.Store
	Invites       namespace.InviteStore
	Collaborators namespace.CollaboratorStore
	Users         user.Store
}

func NewInviteHandler(srv *server.Server) *InviteHandler {
	return &InviteHandler{
		Server:        srv,
		Namespaces:    namespace.Store{Pool: srv.DB},
		Invites:       namespace.InviteStore{Pool: srv.DB},
		Collaborators: namespace.CollaboratorStore{Pool: srv.DB},
		Users:         user.Store{Pool: srv.DB},
	}
}

func (h *InviteHandler) inviteFromRequest(u *user.User, r *http.Request) (*namespace.Invite, bool, error) {
	var (
		i   *namespace.Invite
		ok  bool
		err error
	)

	switch r.Method {
	case "GET":
		_, ok = u.Permissions["invite:read"]
	case "PATCH":
		_, ok = u.Permissions["invite:write"]
	case "DELETE":
		_, ok = u.Permissions["invite:delete"]
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["invite"], 10, 64)

	i, ok, err = h.Invites.Get(query.Where("id", "=", query.Arg(id)))

	if err != nil {
		return nil, false, errors.Err(err)
	}
	return i, ok, nil
}

func (h *InviteHandler) invitesWithRelations(u *user.User) ([]*namespace.Invite, error) {
	ii, err := h.Invites.All(query.Where("invitee_id", "=", query.Arg(u.ID)))

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := namespace.LoadInviteRelations(h.DB, ii...); err != nil {
		return nil, errors.Err(err)
	}

	mm := make([]database.Model, 0, len(ii))

	for _, i := range ii {
		mm = append(mm, i.Namespace)
	}

	if err := h.Users.Load("user_id", "id", mm...); err != nil {
		return nil, errors.Err(err)
	}
	return ii, nil
}

var inviteMail = `%s has invited you to be a collaborator in %s. You can accept this invite via
your Invites list,

    %s/invites`

func (h *InviteHandler) StoreModel(n *namespace.Namespace, r *http.Request) (*namespace.Invite, InviteForm, error) {
	var f InviteForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	invitee, ok, err := h.Users.Get(user.WhereHandle(f.Handle))

	if err != nil {
		return nil, f, errors.Err(err)
	}

	if !ok {
		errs := webutil.NewValidationErrors()
		errs.Add("handle", errors.New("No such user"))

		return nil, f, errs
	}

	if invitee.ID == n.UserID {
		errs := webutil.NewValidationErrors()
		errs.Add("handle", errors.New("Cannot invite yourself"))

		return nil, f, errs
	}

	i, ok, err := h.Invites.Get(query.Where("invitee_id", "=", query.Arg(invitee.ID)))

	if err != nil {
		return nil, f, errors.Err(err)
	}

	if ok {
		errs := webutil.NewValidationErrors()
		errs.Add("handle", errors.New("User already invited"))

		return nil, f, errs
	}

	_, ok, err = h.Collaborators.Get(query.Where("user_id", "=", query.Arg(invitee.ID)))

	if err != nil {
		return nil, f, errors.Err(err)
	}

	if ok {
		errs := webutil.NewValidationErrors()
		errs.Add("handle", errors.New("User already a collaborator"))

		return nil, f, errs
	}

	i, err = h.Invites.Create(namespace.InviteParams{
		NamespaceID: n.ID,
		InviterID:   n.UserID,
		InviteeID:   invitee.ID,
	})

	if err != nil {
		return nil, f, errors.Err(err)
	}

	inviter, _, err := h.Users.Get(user.WhereID(n.UserID))

	if err != nil {
		return nil, f, errors.Err(err)
	}

	ctx := r.Context()

	if h.SMTP.Client != nil {
		h.Queues.Produce(ctx, "email", &mail.Mail{
			From:    h.SMTP.From,
			To:      []string{invitee.Email},
			Subject: fmt.Sprintf("Djinn CI - %s invited you to %s", inviter.Username, n.Path),
			Body:    fmt.Sprintf(inviteMail, inviter.Username, n.Path, webutil.BaseAddress(r)),
		})
	}

	i.Namespace = n
	i.Inviter = inviter
	i.Invitee = invitee

	h.Queues.Produce(ctx, "events", &namespace.InviteEvent{
		Action:    "sent",
		Namespace: n,
		Inviter:   inviter,
		Invitee:   invitee,
	})
	return i, f, nil
}

func (h *InviteHandler) Accept(ctx context.Context, u *user.User, i *namespace.Invite) (*namespace.Namespace, *user.User, *user.User, error) {
	if i.InviteeID != u.ID {
		return nil, nil, nil, database.ErrNotFound
	}

	n, inviter, invitee, err := h.Invites.Accept(i.ID)

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

func (h *InviteHandler) DeleteModel(ctx context.Context, u *user.User, i *namespace.Invite) error {
	if u.ID != i.InviteeID && u.ID != i.InviterID {
		return database.ErrNotFound
	}

	if err := h.Invites.Delete(i.ID); err != nil {
		return errors.Err(err)
	}

	n, _, err := h.Namespaces.Get(query.Where("id", "=", query.Arg(i.NamespaceID)))

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
	Users    user.Store
}

func NewWebhookHandler(srv *server.Server) *WebhookHandler {
	return &WebhookHandler{
		Server: srv,
		Webhooks: &namespace.WebhookStore{
			Pool:   srv.DB,
			AESGCM: srv.AESGCM,
		},
		Users: user.Store{Pool: srv.DB},
	}
}

func (h *WebhookHandler) webhookFromRequest(u *user.User, n *namespace.Namespace, r *http.Request) (*namespace.Webhook, bool, error) {
	var (
		wh  *namespace.Webhook
		ok  bool
		err error
	)

	switch r.Method {
	case "GET":
		_, ok = u.Permissions["webhook:read"]
	case "PATCH":
		_, ok = u.Permissions["webhook:write"]
	case "DELETE":
		_, ok = u.Permissions["webhook:delete"]
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["webhook"], 10, 64)

	wh, ok, err = h.Webhooks.Get(
		query.Where("id", "=", query.Arg(id)),
		query.Where("namespace_id", "=", query.Arg(n.ID)),
	)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	wh.Namespace = n
	return wh, ok, nil
}

func (h *WebhookHandler) StoreModel(u *user.User, n *namespace.Namespace, r *http.Request) (*namespace.Webhook, WebhookForm, error) {
	var (
		f     WebhookForm
		verrs webutil.ValidationErrors
	)

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		if verrs0, ok := err.(webutil.ValidationErrors); ok {
			verrs = verrs0
			goto validate
		}
		return nil, f, errors.Err(err)
	}

validate:
	v := WebhookValidator{
		Webhooks: h.Webhooks,
		Form:     f,
	}

	if err := webutil.Validate(v); err != nil {
		err.(webutil.ValidationErrors).Merge(verrs)

		return nil, f, errors.Err(err)
	}

	url, _ := url.Parse(f.PayloadURL)
	events, _ := event.UnmarshalType(f.Events...)

	wh, err := h.Webhooks.Create(namespace.WebhookParams{
		UserID:      u.ID,
		NamespaceID: n.ID,
		PayloadURL:  url,
		Secret:      f.Secret,
		SSL:         f.SSL,
		Events:      events,
		Active:      f.Active,
	})

	if err != nil {
		return nil, f, errors.Err(err)
	}

	wh.Namespace = n
	return wh, f, nil
}

func (h *WebhookHandler) UpdateModel(wh *namespace.Webhook, r *http.Request) (*namespace.Webhook, WebhookForm, error) {
	var (
		f     WebhookForm
		verrs webutil.ValidationErrors
	)

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		if verrs0, ok := err.(webutil.ValidationErrors); ok {
			verrs = verrs0
			goto validate
		}
		return nil, f, errors.Err(err)
	}

validate:
	v := WebhookValidator{
		Webhooks: h.Webhooks,
		Webhook:  wh,
		Form:     f,
	}

	if err := webutil.Validate(v); err != nil {
		err.(webutil.ValidationErrors).Merge(verrs)

		return nil, f, errors.Err(err)
	}

	url, _ := url.Parse(f.PayloadURL)
	events, _ := event.UnmarshalType(f.Events...)

	if f.Secret == "" {
		if !f.RemoveSecret {
			secret, err := h.AESGCM.Decrypt(wh.Secret)

			if err != nil {
				return nil, f, errors.Err(err)
			}
			f.Secret = string(secret)
		}
	}

	params := namespace.WebhookParams{
		PayloadURL: url,
		Secret:     f.Secret,
		SSL:        f.SSL,
		Events:     events,
		Active:     f.Active,
	}

	if err := h.Webhooks.Update(wh.ID, params); err != nil {
		return nil, f, errors.Err(err)
	}

	wh.PayloadURL = namespace.WebhookURL(url)
	wh.SSL = f.SSL
	wh.Events = events
	wh.Active = f.Active

	return wh, f, nil
}
