package handler

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/image"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/web"
)

type API struct {
	Image

	Prefix string
}

func (h API) Index(w http.ResponseWriter, r *http.Request) {
	ii, paginator, err := h.IndexWithRelations(r)

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

	w.Header().Set("Link", web.EncodeToLink(paginator, r))
	web.JSON(w, data, http.StatusOK)
}

func (h API) Store(w http.ResponseWriter, r *http.Request) {
	i, _, err := h.StoreModel(w, r)

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
			web.JSONError(w, "Unprocessable entity", http.StatusUnprocessableEntity)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}
	web.JSON(w, i.JSON(web.BaseAddress(r) + h.Prefix), http.StatusCreated)
}

func (h API) Show(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Accept") == "application/octet-stream" {
		i, ok := image.FromContext(r.Context())

		if !ok {
			h.Log.Error.Println(r.Method, r.URL, "failed to get image from request context")
		}

		rec, err := h.BlockStore.Open(filepath.Join(i.Driver.String(), i.Hash))

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
	web.JSON(w, i.JSON(web.BaseAddress(r) + h.Prefix), http.StatusOK)
}

func (h API) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
