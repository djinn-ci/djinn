package handler

import (
	"net/http"
	"os"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"
)

// API is the handler for handling API requests made for object creation,
// and management.
type API struct {
	Object

	// Prefix is the part of the URL under which the API is being served, for
	// example "/api".
	Prefix string
}

// Index serves the JSON encoded list of objects for the given request. If
// multiple pages of objects are returned then the database.Paginator is encoded
// in the Link response header.
func (h API) Index(w http.ResponseWriter, r *http.Request) {
	oo, paginator, err := h.IndexWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(oo))
	addr := web.BaseAddress(r) + h.Prefix

	for _, o := range oo {
		data = append(data, o.JSON(addr))
	}

	w.Header().Set("Link", web.EncodeToLink(paginator, r))
	web.JSON(w, data, http.StatusOK)
}

// Store stores the object from the given request body. If any validation
// errors occur then these will be sent back in the JSON response. On success
// the object is sent in the JSON response.
func (h API) Store(w http.ResponseWriter, r *http.Request) {
	o, _, err := h.StoreModel(w, r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.JSON(w, ferrs, http.StatusBadRequest)
			return
		}

		switch cause {
		case namespace.ErrName:
			errs := form.NewErrors()
			errs.Put("namespace", cause)

			web.JSON(w, errs, http.StatusBadRequest)
			return
		case namespace.ErrPermission:
			web.JSON(w, map[string][]string{"namespace": []string{"Could not find namespace"}}, http.StatusBadRequest)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}
	web.JSON(w, o.JSON(web.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

// Show serves up the JSON response for the object in the given request. If the
// Accept header in the request is "application/octet-stream" then the content
// of the image itself is sent in the response. If the base request URL path
// is "/builds" then a list of the builds that the object was placed on will be
// returned. If there are multiple pages of builds then the paginator is
// encoded into the Link response header.
func (h API) Show(w http.ResponseWriter, r *http.Request) {
	base := web.BasePath(r.URL.Path)

	o, err := h.ShowWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	addr := web.BaseAddress(r) + h.Prefix

	if base == "builds" {
		bb, paginator, err := build.NewStore(h.DB).Index(
			r.URL.Query(),
			query.Where(
				"id", "IN", build.SelectObject("build_id", query.Where("object_id", "=", query.Arg(o.ID))),
			),
		)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(bb))

		for _, b := range bb {
			data = append(data, b.JSON(addr))
		}

		w.Header().Set("Link", web.EncodeToLink(paginator, r))
		web.JSON(w, data, http.StatusOK)
		return
	}

	if r.Header.Get("Accept") == "application/octet-stream" {
		rec, err := h.store.Open(o.Hash)

		if err != nil {
			if os.IsNotExist(errors.Cause(err)) {
				web.JSONError(w, "Not found", http.StatusNotFound)
				return
			}
			h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		defer rec.Close()

		http.ServeContent(w, r, o.Name, o.CreatedAt, rec)
		return
	}
	web.JSON(w, o.JSON(addr), http.StatusOK)
}

// Destroy removes the object in the given request context from the database and
// underlying block store. This serves no content in the response body.
func (h API) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
