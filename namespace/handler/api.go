package handler

import (
	"net/http"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/image"
	"github.com/andrewpillar/djinn/key"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/object"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/variable"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/webutil"
)

// API is the handler for handling API requests made for namespace creation,
// and management.
type API struct {
	Namespace

	// Prefix is the part of the URL under which the API is being served, for
	// example "/api".
	Prefix string
}

// InviteAPI is the handler for handling API requests made for sending and
// receiving namespace invites.
type InviteAPI struct {
	Invite

	// Prefix is the part of the URL under which the API is being served, for
	// example "/api".
	Prefix string
}

// CollaboratorAPI is the handler for handling API requests made for managing a
// namespace's collaborators.
type CollaboratorAPI struct {
	Collaborator

	// Prefix is the part of the URL under which the API is being served, for
	// example "/api".
	Prefix string
}

// Index serves the JSON encoded list of namespaces for the given request. If
// multiple pages of namespaces are returned then the database.Paginator is
// encoded in the Link response header.
func (h API) Index(w http.ResponseWriter, r *http.Request) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	nn, paginator, err := h.IndexWithRelations(namespace.NewStore(h.DB, u), r.URL.Query())

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(nn))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, n := range nn {
		data = append(data, n.JSON(addr))
	}

	w.Header().Set("Link", web.EncodeToLink(paginator, r))
	webutil.JSON(w, data, http.StatusOK)
}

// Store stores the namespace from the given request body. If any validation
// errors occur then these will be sent back in the JSON response. On success
// the build is sent in the JSON response.
func (h API) Store(w http.ResponseWriter, r *http.Request) {
	n, _, err := h.StoreModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.JSON(w, ferrs, http.StatusBadRequest)
			return
		}

		switch cause {
		case namespace.ErrDepth:
			web.JSONError(w, cause.Error(), http.StatusUnprocessableEntity)
			return
		case database.ErrNotFound:
			webutil.JSON(w, map[string][]string{"parent": {"Could not find parent"}}, http.StatusBadRequest)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}
	webutil.JSON(w, n.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

// Show serves up the JSON response for the namespace in the given request. This
// serves different responses based on the base path of the request URL.
func (h API) Show(w http.ResponseWriter, r *http.Request) {
	n, ok := namespace.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get namespace from request context")
	}

	err := h.Users.Load("id", []interface{}{n.UserID}, database.Bind("user_id", "id", n))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := h.loadParent(n); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()

	base := webutil.BasePath(r.URL.Path)
	addr := webutil.BaseAddress(r) + h.Prefix

	switch base {
	case "builds":
		bb, paginator, err := build.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if err := build.LoadRelations(h.loaders, bb...); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		mm := make([]database.Model, 0, len(bb))

		for _, b := range bb {
			if b.NamespaceID.Valid {
				mm = append(mm, b.Namespace)
			}
		}

		err = h.Users.Load("id", database.MapKey("user_id", mm), database.Bind("user_id", "id", mm...))

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(bb))

		for _, b := range bb {
			data = append(data, b.JSON(addr))
		}

		w.Header().Set("Link", web.EncodeToLink(paginator, r))
		webutil.JSON(w, data, http.StatusOK)
		return
	case "namespaces":
		nn, paginator, err := h.IndexWithRelations(namespace.NewStore(h.DB, n), q)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(nn))

		for _, n := range nn {
			data = append(data, n.JSON(addr))
		}

		w.Header().Set("Link", web.EncodeToLink(paginator, r))
		webutil.JSON(w, data, http.StatusOK)
		return
	case "images":
		ii, paginator, err := image.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if err := image.LoadRelations(h.loaders, ii...); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(ii))

		for _, i := range ii {
			data = append(data, i.JSON(addr))
		}

		w.Header().Set("Link", web.EncodeToLink(paginator, r))
		webutil.JSON(w, data, http.StatusOK)
		return
	case "objects":
		oo, paginator, err := object.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if err := object.LoadRelations(h.loaders, oo...); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(oo))

		for _, o := range oo {
			data = append(data, o.JSON(addr))
		}

		w.Header().Set("Link", web.EncodeToLink(paginator, r))
		webutil.JSON(w, data, http.StatusOK)
		return
	case "variables":
		vv, paginator, err := variable.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		loaders := h.loaders.Copy()
		loaders.Delete("namespace")

		if err := variable.LoadRelations(loaders, vv...); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(vv))

		for _, v := range vv {
			data = append(data, v.JSON(addr))
		}

		w.Header().Set("Link", web.EncodeToLink(paginator, r))
		webutil.JSON(w, data, http.StatusOK)
		return
	case "keys":
		kk, paginator, err := key.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		loaders := h.loaders.Copy()
		loaders.Delete("namespace")

		if err := key.LoadRelations(loaders, kk...); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(kk))

		for _, k := range kk {
			data = append(data, k.JSON(addr))
		}

		w.Header().Set("Link", web.EncodeToLink(paginator, r))
		webutil.JSON(w, data, http.StatusOK)
		return
	default:
		webutil.JSON(w, n.JSON(addr), http.StatusOK)
		return
	}
}

// Update applies the changes in the given request body to the existing
// namespace in the database. If any validation errors occur then these will
// be sent back in the JSON response. On success the updated namespace is sent
// in the JSON response.
func (h API) Update(w http.ResponseWriter, r *http.Request) {
	n, _, err := h.UpdateModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.JSON(w, ferrs, http.StatusBadRequest)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	webutil.JSON(w, n.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

// Destroy removes the namespace in the given request context from the database.
// This serves no content in the response body.
func (h API) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Index serves the JSON encoded list of invites for the given request.
func (h InviteAPI) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	u, ok := user.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	invites := namespace.NewInviteStore(h.DB)

	n, ok := namespace.FromContext(ctx)

	if !ok {
		invites.Bind(u)
	} else {
		invites.Bind(n)
	}

	ii, err := h.IndexWithRelations(invites)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(ii))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, i := range ii {
		data = append(data, i.JSON(addr))
	}
	webutil.JSON(w, data, http.StatusOK)
}

// Store stores and submits the invite from the given request body. If any
// validation errors occur then these will be sent back in the JSON response.
// On success the invite is sent in the JSON response.
func (h InviteAPI) Store(w http.ResponseWriter, r *http.Request) {
	i, _, err := h.StoreModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.JSON(w, ferrs, http.StatusBadRequest)
			return
		}

		if cause == database.ErrNotFound {
			web.JSONError(w, "Not found", http.StatusNotFound)
			return
		}

		if cause == namespace.ErrPermission {
			web.JSONError(w, "Unprocessable entity", http.StatusUnprocessableEntity)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	webutil.JSON(w, i.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

// Update will accept the invite in the given request context. This will send
// back the JSON response containing the namespace the invite was for, along
// with the inviter and invitee.
func (h InviteAPI) Update(w http.ResponseWriter, r *http.Request) {
	n, inviter, invitee, err := h.Accept(r)

	if err != nil {
		if errors.Cause(err) == database.ErrNotFound {
			web.JSONError(w, "Not found", http.StatusNotFound)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	addr := webutil.BaseAddress(r) + h.Prefix

	data := map[string]interface{}{
		"namespace": n.JSON(addr),
		"inviter":   inviter.JSON(addr),
		"invitee":   invitee.JSON(addr),
	}
	webutil.JSON(w, data, http.StatusOK)
}

// Destroy removes the invite in the given request context from the database.
// This serves no content in the response body.
func (h InviteAPI) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Index serves the JSON encoded list of collaborators for the given request.
func (h CollaboratorAPI) Index(w http.ResponseWriter, r *http.Request) {
	n, ok := namespace.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get namespace from request context")
	}

	cc, err := namespace.NewCollaboratorStore(h.DB, n).All()

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	mm := database.ModelSlice(len(cc), namespace.CollaboratorModel(cc))

	err = h.Users.Load("id", database.MapKey("user_id", mm), database.Bind("user_id", "id", mm...))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(cc))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, c := range cc {
		data = append(data, c.JSON(addr))
	}
	webutil.JSON(w, data, http.StatusOK)
}

// Destroy removes the collaborator in the given request context from the
// database. This serves no content in the response body.
func (h CollaboratorAPI) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		if err == namespace.ErrDeleteSelf {
			body := map[string][]string{
				"collaborator": {"You canno remove yourself from the namespace"},
			}
			webutil.JSON(w, body, http.StatusBadRequest)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
