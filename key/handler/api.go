package handler

import (
	"net/http"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/web"
)

// API is the handler for handling API requests made for key creation, and
// management.
type API struct {
	Key

	// Prefix is the part of the URL under which the API is being served, for
	// example "/api".
	Prefix string
}

// Index serves the JSON encoded list of keys for the given request. If
// multiple pages of keys are returned then the database.Paginator is encoded
// in the Link response header.
func (h API) Index(w http.ResponseWriter, r *http.Request) {
	kk, paginator, err := h.IndexWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(kk))
	addr := web.BaseAddress(r) + h.Prefix

	for _, k := range kk {
		data = append(data, k.JSON(addr))
	}

	w.Header().Set("Link", web.EncodeToLink(paginator, r))
	web.JSON(w, data, http.StatusOK)
}

// Store stores and submits the key from the given request body. If any
// validation errors occur then these will be sent back in the JSON response.
// On success the key is sent in the JSON response.
func (h API) Store(w http.ResponseWriter, r *http.Request) {
	k, _, err := h.StoreModel(r)

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
	web.JSON(w, k.JSON(web.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

// Update applies the changes in the given request body to the existing key
// in the database. If any validation errors occur then these will be sent
// back in the JSON response. On success the updated key is sent in the JSON
// response.
func (h API) Update(w http.ResponseWriter, r *http.Request) {
	k, _, err := h.UpdateModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.JSON(w, ferrs, http.StatusBadRequest)
			return
		}

		if cause == namespace.ErrPermission {
			web.JSONError(w, "Unprocessable entity", http.StatusUnprocessableEntity)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Cause(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	web.JSON(w, k.JSON(web.BaseAddress(r)+h.Prefix), http.StatusOK)
}

// Destroy removes the key in the given request context from the database.
// This serves no content in the response body.
func (h API) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Cause(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
