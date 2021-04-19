package handler

import (
	"net/http"

	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/web"

	"github.com/andrewpillar/webutil"
)

// API is the handler for handling API requests made for image creation, and
// management.
type API struct {
	Variable

	// Prefix is the part of the URL under which the API is being served, for
	// example "/api".
	Prefix string
}

// Index serves the JSON encoded list of variables for the given request. If
// multiple pages of variables are returned then the database.Paginator is
// encoded in the Link response header.
func (h API) Index(w http.ResponseWriter, r *http.Request) {
	vv, paginator, err := h.IndexWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Cause(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(vv))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, v := range vv {
		data = append(data, v.JSON(addr))
	}

	w.Header().Set("Link", web.EncodeToLink(paginator, r))
	webutil.JSON(w, data, http.StatusOK)
}

// Store stores and submits the variable from the given request body. If any
// validation errors occur then these will be sent back in the JSON response.
// On success the variable is sent in the JSON response.
func (h API) Store(w http.ResponseWriter, r *http.Request) {
	v, _, err := h.StoreModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.JSON(w, ferrs, http.StatusBadRequest)
			return
		}

		switch cause {
		case namespace.ErrName:
			errs := webutil.NewErrors()
			errs.Put("namespace", cause)

			webutil.JSON(w, errs, http.StatusBadRequest)
			return
		case namespace.ErrPermission:
			webutil.JSON(w, map[string][]string{"namespace": {"Could not find namespace"}}, http.StatusBadRequest)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}
	webutil.JSON(w, v.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

// Destroy removes the variable in the given request context from the database.
// This serves no content in the response body.
func (h API) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Cause(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
