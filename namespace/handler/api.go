package handler

import (
	"net/http"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/image"
	"github.com/andrewpillar/thrall/key"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/object"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/variable"
	"github.com/andrewpillar/thrall/web"
)

type API struct {
	Namespace

	Prefix string
}

type InviteAPI struct {
	Invite

	Prefix string
}

type CollaboratorAPI struct {
	Collaborator

	Prefix string
}

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
	addr := web.BaseAddress(r) + h.Prefix

	for _, n := range nn {
		data = append(data, n.JSON(addr))
	}

	w.Header().Set("Link", web.EncodeToLink(paginator, r))
	web.JSON(w, data, http.StatusOK)
}

func (h API) Store(w http.ResponseWriter, r *http.Request) {
	n, _, err := h.StoreModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.JSON(w, ferrs, http.StatusBadRequest)
			return
		}

		switch cause {
		case namespace.ErrDepth:
			web.JSONError(w, cause.Error(), http.StatusUnprocessableEntity)
			return
		case database.ErrNotFound:
			web.JSONError(w, "Could not find parent" , http.StatusUnprocessableEntity)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}
	web.JSON(w, n.JSON(web.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

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

	base := web.BasePath(r.URL.Path)
	addr := web.BaseAddress(r) + h.Prefix

	switch base {
	case "builds":
		bb, paginator, err := build.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		// Namespace already bound to build models, so no need to reload.
		loaders := h.Loaders.Copy()
		loaders.Delete("namespace")

		if err := build.LoadRelations(h.Loaders, bb...); err != nil {
			h.Log.Error.Println(errors.Err(err))
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
			h.Log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(bb))

		for _, b := range bb {
			data = append(data, b.JSON(addr))
		}

		w.Header().Set("Link", web.EncodeToLink(paginator, r))
		web.JSON(w, data, http.StatusOK)
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
		web.JSON(w, data, http.StatusOK)
		return
	case "images":
		ii, paginator, err := image.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(ii))

		for _, i := range ii {
			data = append(data, i.JSON(addr))
		}

		w.Header().Set("Link", web.EncodeToLink(paginator, r))
		web.JSON(w, data, http.StatusOK)
		return
	case "objects":
		oo, paginator, err := object.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(oo))

		for _, o := range oo {
			data = append(data, o.JSON(addr))
		}

		w.Header().Set("Link", web.EncodeToLink(paginator, r))
		web.JSON(w, data, http.StatusOK)
		return
	case "variables":
		vv, paginator, err := variable.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(vv))

		for _, v := range vv {
			data = append(data, v.JSON(addr))
		}

		w.Header().Set("Link", web.EncodeToLink(paginator, r))
		web.JSON(w, data, http.StatusOK)
		return
	case "keys":
		kk, paginator, err := key.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(kk))

		for _, k := range kk {
			data = append(data, k.JSON(addr))
		}

		w.Header().Set("Link", web.EncodeToLink(paginator, r))
		web.JSON(w, data, http.StatusOK)
		return
	default:
		web.JSON(w, n.JSON(addr), http.StatusOK)
		return
	}
}

func (h API) Update(w http.ResponseWriter, r *http.Request) {
	n, _, err := h.UpdateModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.JSON(w, ferrs, http.StatusBadRequest)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	web.JSON(w, n.JSON(web.BaseAddress(r)+h.Prefix), http.StatusOK)
}

func (h API) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h InviteAPI) Index(w http.ResponseWriter, r *http.Request) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	ii, err := h.IndexWithRelations(namespace.NewInviteStore(h.DB, u))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(ii))
	addr := web.BaseAddress(r) + h.Prefix

	for _, i := range ii {
		data = append(data, i.JSON(addr))
	}
	web.JSON(w, data, http.StatusOK)
}

func (h InviteAPI) Store(w http.ResponseWriter, r *http.Request) {
	i, _, err := h.StoreModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.JSON(w, ferrs, http.StatusBadRequest)
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
	web.JSON(w, i.JSON(web.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

func (h InviteAPI) Update(w http.ResponseWriter, r *http.Request) {
	n, inviter, invitee, err := h.Accept(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	addr := web.BaseAddress(r) + h.Prefix

	data := map[string]interface{}{
		"namespace": n.JSON(addr),
		"inviter":   inviter.JSON(addr),
		"invitee":   invitee.JSON(addr),
	}
	web.JSON(w, data, http.StatusOK)
}

func (h InviteAPI) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

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
	addr := web.BaseAddress(r) + h.Prefix

	for _, c := range cc {
		data = append(data, c.JSON(addr))
	}
	web.JSON(w, data, http.StatusOK)
}

func (h CollaboratorAPI) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
