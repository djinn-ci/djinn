package handler

import (
	"net/http"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/cron"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

// API is the handler for handling API requests made for cron job creation,
// and management.
type API struct {
	Cron

	// Prefix is the part of the URL under which the API is being served, for
	// example "/api".
	Prefix string
}

// Index serves the JSON encoded list of cron jobs for the given request. If
// multiple pages of cron jobs are returned then the database.Paginator is
// encoded in the Link response header.
func (h API) Index(w http.ResponseWriter, r *http.Request) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	q := r.URL.Query()

	cc, paginator, err := h.IndexWithRelations(cron.NewStore(h.DB, u), q)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	data := make([]interface{}, 0, len(cc))
	addr := webutil.BaseAddress(r) + h.Prefix

	for _, c := range cc {
		data = append(data, c.JSON(addr))
	}

	w.Header().Set("Link", web.EncodeToLink(paginator, r))
	webutil.JSON(w, data, http.StatusOK)
}

// Store stores and submits the cron job from the given request body. If any
// validation errors occur then these will be sent back in the JSON response.
// On success the cron job is sent in the JSON response.
func (h API) Store(w http.ResponseWriter, r *http.Request) {
	c, _, err := h.StoreModel(r)

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
			webutil.JSON(w, map[string][]string{"namespace": []string{"Could not find namespace"}}, http.StatusBadRequest)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}
	webutil.JSON(w, c.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusCreated)
}

// Show serves up the JSON response for the cron job in the given request. If
// the base path of the URL is "/builds" then the builds submitted via the
// cron will be served up in the JSON response.
func (h API) Show(w http.ResponseWriter, r *http.Request) {
	c, err := h.ShowWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Cause(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if webutil.BasePath(r.URL.Path) == "builds" {
		bb, paginator, err := build.NewStore(h.DB).Index(
			r.URL.Query(),
			query.Where(
				"id", "IN", cron.SelectBuild("build_id", query.Where("cron_id", "=", query.Arg(c.ID))),
			),
		)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Cause(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		data := make([]interface{}, 0, len(bb))
		addr := webutil.BaseAddress(r) + h.Prefix

		for _, b := range bb {
			data = append(data, b.JSON(addr))
		}

		w.Header().Set("Link", web.EncodeToLink(paginator, r))
		webutil.JSON(w, data, http.StatusOK)
		return
	}
	webutil.JSON(w, c.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

// Update applies the changes in the given request body to the existing cron
// job in the database. If any validation errors occur then these will be sent
// back in the JSON response. On success the updated cron job is sent in the
// JSON response.
func (h API) Update(w http.ResponseWriter, r *http.Request) {
	c, _, err := h.UpdateModel(r)

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
			web.JSONError(w, "Unprocessable entity", http.StatusUnprocessableEntity)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}
	webutil.JSON(w, c.JSON(webutil.BaseAddress(r)+h.Prefix), http.StatusOK)
}

// Destroy removes the cron job in the given request context from the database.
// This serves no content in the response body.
func (h API) Destroy(w http.ResponseWriter, r *http.Request) {
	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Cause(err))
		web.JSONError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
