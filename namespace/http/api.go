package http

import (
	"net/http"

	"djinn-ci.com/auth"
	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/gorilla/mux"
)

type API struct {
	*Handler
}

func (h API) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	p, err := h.Handler.Index(u, r)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get namespaces"))
		return
	}

	w.Header().Set("Link", p.EncodeToLink(r.URL))
	webutil.JSON(w, p.Items, http.StatusOK)
}

func (h API) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	n, _, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, nil, errors.Wrap(err, "Failed to create namespace"))
		return
	}
	webutil.JSON(w, n, http.StatusCreated)
}

func (h API) Show(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	switch webutil.BasePath(r.URL.Path) {
	case "builds":
		p, err := h.Builds.Index(ctx, q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get builds"))
			return
		}

		if err := build.LoadRelations(ctx, h.DB, p.Items...); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		w.Header().Set("Link", p.EncodeToLink(r.URL))
		webutil.JSON(w, p.Items, http.StatusOK)
	case "namespaces":
		p, err := h.Namespaces.Index(ctx, q, query.Where("parent_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get namespaces"))
			return
		}

		if err := h.loadLastBuild(ctx, p.Items); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		w.Header().Set("Link", p.EncodeToLink(r.URL))
		webutil.JSON(w, p.Items, http.StatusOK)
	case "images":
		p, err := h.Images.Index(ctx, q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get images"))
			return
		}

		w.Header().Set("Link", p.EncodeToLink(r.URL))
		webutil.JSON(w, p.Items, http.StatusOK)
	case "objects":
		p, err := h.Objects.Index(ctx, q, query.Where("namespace_id", "-", query.Arg(n.ID)))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get objects"))
			return
		}

		w.Header().Set("Link", p.EncodeToLink(r.URL))
		webutil.JSON(w, p.Items, http.StatusOK)
	case "variables":
		p, err := h.Variables.Index(ctx, q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get variables"))
			return
		}

		w.Header().Set("Link", p.EncodeToLink(r.URL))
		webutil.JSON(w, p.Items, http.StatusOK)
	case "keys":
		p, err := h.Keys.Index(ctx, q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get keys"))
			return
		}

		w.Header().Set("Link", p.EncodeToLink(r.URL))
		webutil.JSON(w, p.Items, http.StatusOK)
	case "invites":
		ii, err := h.Invites.All(ctx, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get invites"))
			return
		}

		mm := make([]database.Model, 0, len(ii))

		for _, i := range ii {
			mm = append(mm, i)
		}

		ld := user.Loader(h.DB)

		if err := ld.Load(ctx, "invitee_id", "id", mm...); err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to loader users"))
			return
		}

		if err := ld.Load(ctx, "inviter_id", "id", mm...); err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to loader users"))
			return
		}
		webutil.JSON(w, ii, http.StatusOK)
	case "collaborators":
		cc, err := h.Collaborators.All(ctx, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get collaborators"))
			return
		}

		mm := make([]database.Model, 0, len(cc))

		for _, c := range cc {
			mm = append(mm, c)
		}

		if err := user.Loader(h.DB).Load(ctx, "user_id", "id", mm...); err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to load users"))
			return
		}
		webutil.JSON(w, cc, http.StatusOK)
	case "webhooks":
		ww, err := h.Webhooks.All(ctx, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get webhooks"))
			return
		}

		mm := make([]database.Model, 0, len(ww))

		for _, w := range ww {
			mm = append(mm, w)
		}

		if err := user.Loader(h.DB).Load(ctx, "author_id", "id", mm...); err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to load users"))
			return
		}
		webutil.JSON(w, ww, http.StatusOK)
	default:
		b, ok, err := h.Builds.Get(
			ctx, query.Where("namespace_id", "=", query.Arg(n.ID)), query.OrderDesc("created_at"),
		)

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get last build"))
			return
		}

		if ok {
			if err := build.LoadRelations(ctx, h.DB, b); err != nil {
				h.Error(w, r, errors.Wrap(err, "Failed to get last build"))
				return
			}

			b.Namespace = nil
			n.Build = b
		}
		webutil.JSON(w, n, http.StatusOK)
	}
}

func (h API) DestroyCollaborator(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	if err := h.Handler.DestroyCollaborator(u, n, r); err != nil {
		if errors.Is(err, database.ErrPermission) {
			webutil.JSON(w, map[string]string{"message": "Failed to remove collaborator"}, http.StatusBadRequest)
			return
		}

		if errors.Is(err, database.ErrNoRows) {
			webutil.JSON(w, map[string]string{"message": "No such collaborator"}, http.StatusBadRequest)
			h.Error(w, r, errors.Benign("No such collaborator"))
			return
		}

		h.Error(w, r, errors.Wrap(err, "Failed to remove collaborator"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h API) Update(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	n, _, err := h.Handler.Update(u, n, r)

	if err != nil {
		h.FormError(w, r, nil, errors.Wrap(err, "Failed to update namespace"))
		return
	}
	webutil.JSON(w, n, http.StatusOK)
}

func (h API) Destroy(_ *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	if err := h.Handler.Destroy(r.Context(), n); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type InviteAPI struct {
	*InviteHandler
}

func (h InviteAPI) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ii, err := h.Invites.All(ctx, query.Where("invitee_id", "=", query.Arg(u.ID)))

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get invites"))
		return
	}

	mm := make([]database.Model, 0, len(ii))

	for _, i := range ii {
		mm = append(mm, i)
	}

	ld := user.Loader(h.DB)

	if err := ld.Load(ctx, "invitee_id", "id", mm...); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to load users"))
		return
	}

	if err := ld.Load(ctx, "inviter_id", "id", mm...); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to loader users"))
		return
	}
	webutil.JSON(w, ii, http.StatusOK)
}

func (h InviteAPI) Store(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	i, _, err := h.InviteHandler.Store(u, n, r)

	if err != nil {
		errtab := map[error]string{
			database.ErrNoRows:        "No such user",
			namespace.ErrSelfInvite:   "Cannot invite self",
			namespace.ErrInviteSent:   "Invite already sent",
			namespace.ErrCollaborator: "Already a collaborator",
		}

		if err, ok := errtab[err]; ok {
			webutil.JSON(w, map[string]string{"message": err}, http.StatusBadRequest)
			return
		}

		h.FormError(w, r, nil, errors.Wrap(err, "Failed to send invite"))
		return
	}
	webutil.JSON(w, i, http.StatusCreated)
}

func (h InviteAPI) Update(u *auth.User, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id := mux.Vars(r)["invite"]

	i, ok, err := h.Invites.Get(ctx, query.Where("id", "=", query.Arg(id)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	n, inviter, invitee, err := h.Accept(ctx, u, i)

	if err != nil {
		if errors.Is(err, database.ErrNoRows) {
			h.NotFound(w, r)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	data := map[string]any{
		"namespace": n,
		"invitee":   invitee,
		"inviter":   inviter,
	}
	webutil.JSON(w, data, http.StatusOK)
}

func (h InviteAPI) Destroy(u *auth.User, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	i, ok, err := h.Invites.Get(ctx, query.Where("id", "=", query.Arg(mux.Vars(r)["invite"])))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if err := h.InviteHandler.Destroy(ctx, u, i); err != nil {
		if errors.Is(err, database.ErrNoRows) {
			h.NotFound(w, r)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type WebhookAPI struct {
	*WebhookHandler
}

func (h WebhookAPI) Store(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	wh, _, err := h.WebhookHandler.Store(u, n, r)

	if err != nil {
		h.FormError(w, r, nil, errors.Wrap(err, "Failed to create webhook"))
		return
	}
	webutil.JSON(w, wh, http.StatusCreated)
}

func (h WebhookAPI) Show(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	wh, ok, err := h.Webhooks.Get(
		ctx,
		query.Where("id", "=", query.Arg(mux.Vars(r)["webhook"])),
		query.Where("namespace_id", "=", query.Arg(n.ID)),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get webhook"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	wh.LastDelivery, err = h.Webhooks.LastDelivery(ctx, wh)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get last webhook delivery"))
		return
	}
	webutil.JSON(w, wh, http.StatusOK)
}

func (h WebhookAPI) Update(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	wh, ok, err := h.Webhooks.Get(
		ctx,
		query.Where("id", "=", query.Arg(mux.Vars(r)["webhook"])),
		query.Where("namespace_id", "=", query.Arg(n.ID)),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get webhook"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	wh, _, err = h.WebhookHandler.Update(wh, r)

	if err != nil {
		h.FormError(w, r, nil, errors.Wrap(err, "Failed to update webhook"))
		return
	}
	webutil.JSON(w, wh, http.StatusOK)
}

func (h WebhookAPI) Destroy(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	wh, ok, err := h.Webhooks.Get(
		ctx,
		query.Where("id", "=", query.Arg(mux.Vars(r)["webhook"])),
		query.Where("namespace_id", "=", query.Arg(n.ID)),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get webhook"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if err := h.Webhooks.Delete(ctx, wh); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to delete webhook"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func registerNamespaceAPI(a auth.Authenticator, srv *server.Server) {
	api := API{
		Handler: NewHandler(srv),
	}

	index := srv.Restrict(a, []string{"namespace:read"}, api.Index)
	store := srv.Restrict(a, []string{"namespace:write"}, api.Store)

	root := srv.Router.PathPrefix("/").Subrouter()
	root.HandleFunc("/namespaces", index).Methods("GET")
	root.HandleFunc("/namespaces", store).Methods("POST")

	a = namespace.NewAuth(a, "namespace", api.Namespaces.Store)

	show := srv.Restrict(a, []string{"namespace:read"}, api.Namespace(api.Show))
	update := srv.Restrict(a, []string{"namespace:write", "owner"}, api.Namespace(api.Update))
	destroy := srv.Restrict(a, []string{"namespace:delete", "owner"}, api.Namespace(api.Destroy))
	destroyCollab := srv.Restrict(a, []string{"namespace:delete", "owner"}, api.Namespace(api.DestroyCollaborator))

	sr := srv.Router.PathPrefix("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}").Subrouter()
	sr.HandleFunc("", show).Methods("GET")
	sr.HandleFunc("/-/badge.svg", api.Badge).Methods("GET")
	sr.HandleFunc("/-/namespaces", show).Methods("GET")
	sr.HandleFunc("/-/images", show).Methods("GET")
	sr.HandleFunc("/-/objects", show).Methods("GET")
	sr.HandleFunc("/-/variables", show).Methods("GET")
	sr.HandleFunc("/-/keys", show).Methods("GET")
	sr.HandleFunc("/-/collaborators", show).Methods("GET")
	sr.HandleFunc("/-/collaborators/{collaborator}", destroyCollab).Methods("DELETE")
	sr.HandleFunc("/-/invites", srv.Restrict(a, []string{"invite:read"}, api.Namespace(api.Show))).Methods("GET")
	sr.HandleFunc("/-/webhooks", srv.Restrict(a, []string{"webhook:read"}, api.Namespace(api.Show))).Methods("GET")
	sr.HandleFunc("", update).Methods("PATCH")
	sr.HandleFunc("", destroy).Methods("DELETE")
}

func registerWebhookAPI(a auth.Authenticator, srv *server.Server) {
	api := WebhookAPI{
		WebhookHandler: NewWebhookHandler(srv),
	}

	h := NewHandler(srv)

	store := srv.Restrict(a, []string{"webhook:write"}, h.Namespace(api.Store))

	a = namespace.NewAuth(a, "webhook", api.Webhooks.Store)

	show := srv.Restrict(a, []string{"webhook:read"}, h.Namespace(api.Show))
	update := srv.Restrict(a, []string{"webhook:write"}, h.Namespace(api.Update))
	destroy := srv.Restrict(a, []string{"webhook:delete"}, h.Namespace(api.Destroy))

	sr := srv.Router.PathPrefix("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}/-/webhooks").Subrouter()
	sr.HandleFunc("", store).Methods("POST")
	sr.HandleFunc("/{webhook:[0-9]+}", show).Methods("GET")
	sr.HandleFunc("/{webhook:[0-9]+}", update).Methods("PATCH")
	sr.HandleFunc("/{webhook:[0-9]+}", destroy).Methods("DELETE")
}

func registerInviteAPI(a auth.Authenticator, srv *server.Server) {
	api := InviteAPI{
		InviteHandler: NewInviteHandler(srv),
	}

	h := NewHandler(srv)

	index := srv.Restrict(a, []string{"invite:read"}, api.Index)
	store := srv.Restrict(a, []string{"invite:write"}, h.Namespace(api.Store))

	a = namespace.NewAuth[*namespace.Invite](a, "invite", api.Invites.Store)

	update := srv.Restrict(a, []string{"invite:write"}, api.Update)
	destroy := srv.Restrict(a, []string{"invite:write"}, api.Destroy)

	root := srv.Router.PathPrefix("/").Subrouter()
	root.HandleFunc("/invites", index).Methods("GET")
	root.HandleFunc("/invites/{invite:[0-9]+}", update).Methods("PATCH")
	root.HandleFunc("/invites/{invite:[0-9]+}", destroy).Methods("DELETE")
	root.HandleFunc("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}/-/invites", store).Methods("POST")
}

func RegisterAPI(a auth.Authenticator, srv *server.Server) {
	registerNamespaceAPI(a, srv)
	registerWebhookAPI(a, srv)
	registerInviteAPI(a, srv)
}
