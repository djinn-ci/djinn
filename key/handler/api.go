package handler

import (
	"net/http"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/web"
)

type API struct {
	Key

	Prefix string
}

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
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}
	web.JSON(w, k.JSON(web.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

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

func (h API) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Cause(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
