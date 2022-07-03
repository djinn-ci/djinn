package http

import (
	"net/http"
	"strconv"

	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/mux"
)

type API struct {
	*Handler

	Prefix string
}

func (h API) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	nn, paginator, err := h.IndexWithRelations(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	data := make([]map[string]interface{}, 0, len(nn))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, n := range nn {
		data = append(data, n.JSON(addr))
	}

	w.Header().Set("Link", paginator.EncodeToLink(r.URL))
	webutil.JSON(w, data, http.StatusOK)
}

func (h API) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	n, _, err := h.StoreModel(u, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			if errs, ok := verrs["fatal"]; ok {
				h.InternalServerError(w, r, errors.Slice(errs))
				return
			}

			webutil.JSON(w, verrs, http.StatusBadRequest)
			return
		}

		switch cause {
		case namespace.ErrDepth:
			h.Error(w, r, cause.Error(), http.StatusUnprocessableEntity)
		case database.ErrNotFound:
			webutil.JSON(w, map[string][]string{"parent": {"Could not find parent"}}, http.StatusBadRequest)
		default:
			h.InternalServerError(w, r, errors.Err(err))
		}
		return
	}
	webutil.JSON(w, n.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

func (h API) Show(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	base := webutil.BasePath(r.URL.Path)
	addr := webutil.BaseAddress(r) + h.Prefix

	switch base {
	case "builds":
		bb, paginator, err := h.Builds.Index(q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if err := build.LoadRelations(h.DB, bb...); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(bb))

		for _, b := range bb {
			data = append(data, b.JSON(addr))
		}

		w.Header().Set("Link", paginator.EncodeToLink(r.URL))
		webutil.JSON(w, data, http.StatusOK)
	case "namespaces":
		nn, paginator, err := h.Namespaces.Index(r.URL.Query(), query.Where("parent_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if err := h.loadLastBuild(nn); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(nn))

		for _, n := range nn {
			data = append(data, n.JSON(addr))
		}

		w.Header().Set("Link", paginator.EncodeToLink(r.URL))
		webutil.JSON(w, data, http.StatusOK)
	case "images":
		ii, paginator, err := h.Images.Index(q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(ii))

		for _, i := range ii {
			data = append(data, i.JSON(addr))
		}

		w.Header().Set("Link", paginator.EncodeToLink(r.URL))
		webutil.JSON(w, data, http.StatusOK)
	case "objects":
		oo, paginator, err := h.Objects.Index(q, query.Where("namespace_id", "-", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(oo))

		for _, o := range oo {
			data = append(data, o.JSON(addr))
		}

		w.Header().Set("Link", paginator.EncodeToLink(r.URL))
		webutil.JSON(w, data, http.StatusOK)
	case "variables":
		vv, paginator, err := h.Variables.Index(q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(vv))

		for _, v := range vv {
			data = append(data, v.JSON(addr))
		}

		w.Header().Set("Link", paginator.EncodeToLink(r.URL))
		webutil.JSON(w, data, http.StatusOK)
	case "keys":
		kk, paginator, err := h.Keys.Index(q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(kk))

		for _, k := range kk {
			data = append(data, k.JSON(addr))
		}

		w.Header().Set("Link", paginator.EncodeToLink(r.URL))
		webutil.JSON(w, data, http.StatusOK)
	case "invites":
		ii, err := h.Invites.All(query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		mm := make([]database.Model, 0, len(ii))

		for _, i := range ii {
			mm = append(mm, i)
		}

		if err := h.Users.Load("inviter_id", "id", mm...); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if err := h.Users.Load("invitee_id", "id", mm...); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(ii))

		for _, i := range ii {
			data = append(data, i.JSON(addr))
		}
		webutil.JSON(w, data, http.StatusOK)
	case "collaborators":
		cc, err := h.Collaborators.All(query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		mm := make([]database.Model, 0, len(cc))

		for _, c := range cc {
			mm = append(mm, c)
		}

		if err := h.Users.Load("user_id", "id", mm...); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(cc))

		for _, c := range cc {
			data = append(data, c.JSON(addr))
		}
		webutil.JSON(w, data, http.StatusOK)
	case "webhooks":
		ww, err := h.Webhooks.All(query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		data := make([]map[string]interface{}, 0, len(ww))
		mm := make([]database.Model, 0, len(ww))

		for _, w := range ww {
			data = append(data, w.JSON(addr))
			mm = append(mm, w)
		}

		if err := h.Users.Load("author_id", "id", mm...); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}
		webutil.JSON(w, data, http.StatusOK)
	default:
		webutil.JSON(w, n.JSON(addr), http.StatusOK)
	}
}

func (h API) DestroyCollab(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	if err := h.deleteCollab(u, n, r); err != nil {
		if errors.Is(err, namespace.ErrPermission) {
			h.Error(w, r, "Failed to remove collaborator", http.StatusBadRequest)
			return
		}

		if errors.Is(err, database.ErrNotFound) {
			h.Error(w, r, "No such collaborator", http.StatusBadRequest)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h API) Update(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	n, _, err := h.UpdateModel(n, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			if errs, ok := verrs["fatal"]; ok {
				h.InternalServerError(w, r, errors.Slice(errs))
				return
			}

			webutil.JSON(w, verrs, http.StatusBadRequest)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	webutil.JSON(w, n.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

func (h API) Destroy(_ *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r.Context(), n); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type InviteAPI struct {
	*InviteHandler

	Prefix string
}

func (h InviteAPI) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	ii, err := h.invitesWithRelations(u)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	data := make([]map[string]interface{}, 0, len(ii))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, i := range ii {
		data = append(data, i.JSON(addr))
	}
	webutil.JSON(w, data, http.StatusOK)
}

func (h InviteAPI) Store(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	i, _, err := h.StoreModel(n, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			webutil.JSON(w, verrs, http.StatusBadRequest)
			return
		}

		if errors.Is(cause, database.ErrNotFound) || errors.Is(cause, namespace.ErrPermission) {
			h.NotFound(w, r)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	webutil.JSON(w, i.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

func (h InviteAPI) Update(u *user.User, w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(mux.Vars(r)["invite"], 10, 64)

	i, ok, err := h.Invites.Get(query.Where("id", "=", query.Arg(id)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	n, inviter, invitee, err := h.Accept(r.Context(), u, i)

	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.NotFound(w, r)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	addr := webutil.BaseAddress(r) + h.Prefix

	data := map[string]interface{}{
		"namespace": n.JSON(addr),
		"invitee":   invitee.JSON(addr),
		"inviter":   inviter.JSON(addr),
	}
	webutil.JSON(w, data, http.StatusOK)
}

func (h InviteAPI) Destroy(u *user.User, w http.ResponseWriter, r *http.Request) {
	i, ok, err := h.inviteFromRequest(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if err := h.DeleteModel(r.Context(), u, i); err != nil {
		if errors.Is(err, database.ErrNotFound) {
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

	Prefix string
}

func (h WebhookAPI) Store(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	wh, _, err := h.StoreModel(u, n, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			webutil.JSON(w, verrs, http.StatusBadRequest)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := h.Users.Load("user_id", "id", wh.Namespace); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	webutil.JSON(w, wh.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

func (h WebhookAPI) Show(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	wh, ok, err := h.webhookFromRequest(u, n, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	wh.LastDelivery, err = h.Webhooks.LastDelivery(wh.ID)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	webutil.JSON(w, wh.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

func (h WebhookAPI) Update(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	wh, ok, err := h.webhookFromRequest(u, n, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	wh, _, err = h.UpdateModel(wh, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			webutil.JSON(w, verrs, http.StatusBadRequest)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	webutil.JSON(w, wh.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

func (h WebhookAPI) Destroy(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	wh, ok, err := h.webhookFromRequest(u, n, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if err := h.Webhooks.Delete(wh.ID); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func RegisterAPI(prefix string, srv *server.Server) {
	user := userhttp.NewHandler(srv)

	api := API{
		Handler: NewHandler(srv),
		Prefix:  prefix,
	}

	inviteapi := InviteAPI{
		InviteHandler: NewInviteHandler(srv),
		Prefix:        prefix,
	}

	hookapi := WebhookAPI{
		WebhookHandler: NewWebhookHandler(srv),
		Prefix:         prefix,
	}

	auth := srv.Router.PathPrefix("/").Subrouter()
	auth.HandleFunc("/namespaces", user.WithUser(api.Index)).Methods("GET")
	auth.HandleFunc("/namespaces", user.WithUser(api.Store)).Methods("POST")
	auth.HandleFunc("/invites", user.WithUser(inviteapi.Index)).Methods("GET")
	auth.HandleFunc("/invites/{invite:[0-9]+}", user.WithUser(inviteapi.Update)).Methods("PATCH")
	auth.HandleFunc("/invites/{invite:[0-9]+}", user.WithUser(inviteapi.Destroy)).Methods("DELETE")

	sr := srv.Router.PathPrefix("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}").Subrouter()
	sr.HandleFunc("", user.WithUser(api.WithNamespace(api.Show))).Methods("GET")
	sr.HandleFunc("/-/namespaces", user.WithOptionalUser(api.WithNamespace(api.Show))).Methods("GET")
	sr.HandleFunc("/-/images", user.WithOptionalUser(api.WithNamespace(api.Show))).Methods("GET")
	sr.HandleFunc("/-/objects", user.WithOptionalUser(api.WithNamespace(api.Show))).Methods("GET")
	sr.HandleFunc("/-/variables", user.WithOptionalUser(api.WithNamespace(api.Show))).Methods("GET")
	sr.HandleFunc("/-/keys", user.WithOptionalUser(api.WithNamespace(api.Show))).Methods("GET")
	sr.HandleFunc("/-/invites", user.WithUser(api.WithNamespace(api.Show))).Methods("GET")
	sr.HandleFunc("/-/invites", user.WithUser(api.WithNamespace(inviteapi.Store))).Methods("POST")
	sr.HandleFunc("/-/collaborators", user.WithUser(api.WithNamespace(api.Show))).Methods("GET")
	sr.HandleFunc("/-/collaborators/{collaborator}", user.WithUser(api.WithNamespace(api.DestroyCollab))).Methods("DELETE")
	sr.HandleFunc("/-/webhooks", user.WithUser(api.WithNamespace(api.Show))).Methods("GET")
	sr.HandleFunc("/-/webhooks", user.WithUser(api.WithNamespace(hookapi.Store))).Methods("POST")
	sr.HandleFunc("/-/webhooks/{webhook:[0-9]+}", user.WithUser(api.WithNamespace(hookapi.Show))).Methods("GET")
	sr.HandleFunc("/-/webhooks/{webhook:[0-9]+}", user.WithUser(api.WithNamespace(hookapi.Update))).Methods("PATCH")
	sr.HandleFunc("/-/webhooks/{webhook:[0-9]+}", user.WithUser(api.WithNamespace(hookapi.Destroy))).Methods("DELETE")
	sr.HandleFunc("", user.WithUser(api.WithNamespace(api.Update))).Methods("PATCH")
	sr.HandleFunc("", user.WithUser(api.WithNamespace(api.Destroy))).Methods("DELETE")
}
