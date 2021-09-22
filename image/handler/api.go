package handler

import (
	"net/http"
	"os"
	"path/filepath"

	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/namespace"
	"djinn-ci.com/web"

	"github.com/andrewpillar/webutil"
)

// API is the handler for handling API requests made for image creation,
// and management.
type API struct {
	Image

	// Prefix is the part of the URL under which the API is being served, for
	// example "/api".
	Prefix string
}

// Index serves the JSON encoded list of images for the given request. If
// multiple pages of images are returned then the database.Paginator is encoded
// in the Link response header.
func (h API) Index(w http.ResponseWriter, r *http.Request) {
	ii, paginator, err := h.IndexWithRelations(r)

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

	w.Header().Set("Link", web.EncodeToLink(paginator, r))
	webutil.JSON(w, data, http.StatusOK)
}

// Store stores the image from the given request body. If any validation
// errors occur then these will be sent back in the JSON response. On success
// the image is sent in the JSON response.
func (h API) Store(w http.ResponseWriter, r *http.Request) {
	i, _, err := h.StoreModel(w, r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.JSON(w, ferrs, http.StatusBadRequest)
			return
		}

		if patherr, ok := cause.(*os.PathError); ok {
			web.JSONError(w, patherr.Error(), http.StatusInternalServerError)
			return
		}

		switch cause {
		case image.ErrInvalidScheme:
			errs := webutil.NewErrors()
			errs.Put("download_url", cause)

			webutil.JSON(w, errs, http.StatusBadRequest)
			return
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
	webutil.JSON(w, i.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

// Show serves up the JSON response for the image in the given request. If the
// Accept header in the request is "application/x-qemu-disk" then the content
// of the image itself is sent in the response.
func (h API) Show(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Accept") == image.MimeTypeQEMU {
		i, ok := image.FromContext(r.Context())

		if !ok {
			h.Log.Error.Println(r.Method, r.URL, "failed to get image from request context")
		}

		store, err := h.store.Partition(i.UserID)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		rec, err := store.Open(filepath.Join(i.Driver.String(), i.Hash))

		if err != nil {
			if os.IsNotExist(errors.Cause(err)) {
				web.JSONError(w, "Not found", http.StatusNotFound)
				return
			}
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		defer rec.Close()
		http.ServeContent(w, r, i.Name, i.CreatedAt, rec)
		return
	}

	i, err := h.ShowWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
	}
	webutil.JSON(w, i.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

// Destroy removes the image in the given request context from the database and
// underlying block store. This serves no content in the response body.
func (h API) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
