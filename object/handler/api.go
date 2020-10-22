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

type API struct {
	Object

	Prefix string
}

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
		bb, paginator, err := h.Builds.Index(
			r.URL.Query(),
			query.WhereQuery(
				"id", "IN", build.SelectObject("build_id", query.Where("object_id", "=", o.ID)),
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
		rec, err := h.BlockStore.Open(o.Hash)

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

func (h API) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
