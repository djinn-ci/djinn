package http

import (
	"io"
	"net/http"
	"regexp"
	"strings"

	"djinn-ci.com/alert"
	"djinn-ci.com/auth"
	"djinn-ci.com/build"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/template/form"

	"github.com/andrewpillar/fs"
	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type UI struct {
	*Handler
}

func (h UI) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	p, err := h.Handler.Index(u, r)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get objects"))
		return
	}

	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.ObjectIndex{
		Paginator: template.NewPaginator[*object.Object](tmpl.Page, p),
		Objects:   p.Items,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Create(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.ObjectCreate{
		Form: form.New(sess, r),
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	o, f, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, f, errors.Wrap(err, "Failed to create object"))
		return
	}

	alert.Flash(sess, alert.Success, "Object has been added: "+o.Name)
	h.Redirect(w, r, "/objects")
}

func (h UI) Show(u *auth.User, o *object.Object, w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")

	if parts[len(parts)-2] == "download" {
		if o.Name != mux.Vars(r)["name"] {
			h.NotFound(w, r)
			return
		}

		f, err := o.Open(h.Objects.FS)

		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				h.NotFound(w, r)
				return
			}

			h.Error(w, r, errors.Wrap(err, "Failed to download object"))
			return
		}

		defer f.Close()

		http.ServeContent(w, r, o.Name, o.CreatedAt, f.(io.ReadSeeker))
		return
	}

	ctx := r.Context()

	p, err := h.Builds.Index(
		ctx,
		r.URL.Query(),
		query.Where("id", "IN", build.SelectObject(
			query.Columns("build_id"),
			query.Where("object_id", "=", query.Arg(o.ID)),
		)),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get builds"))
		return
	}

	if err := build.LoadRelations(ctx, h.DB, p.Items...); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to load build relations"))
		return
	}

	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.ObjectShow{
		Page:   tmpl.Page,
		Object: o,
		Builds: &template.BuildIndex{
			Paginator: template.NewPaginator[*build.Build](tmpl.Page, p),
			Builds:    p.Items,
		},
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

var reObjectURI = regexp.MustCompile("/objects/[0-9]+")

func (h UI) Destroy(u *auth.User, o *object.Object, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.Handler.Destroy(r.Context(), o); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to delete object"))
		return
	}

	alert.Flash(sess, alert.Success, "Object has been deleted")

	if reObjectURI.Match([]byte(r.Header.Get("Referer"))) {
		h.Redirect(w, r, "/objects")
		return
	}
	h.RedirectBack(w, r)
}

func RegisterUI(a auth.Authenticator, srv *server.Server) {
	ui := UI{
		Handler: NewHandler(srv),
	}

	index := ui.Restrict(a, []string{"object:read"}, ui.Index)
	create := ui.Restrict(a, []string{"object:write"}, ui.Create)
	store := ui.Restrict(a, []string{"object:write"}, ui.Store)

	a = namespace.NewAuth(a, "object", object.NewStore(srv.DB))

	show := ui.Restrict(a, []string{"object:read", "build:read"}, ui.Object(ui.Show))
	destroy := ui.Restrict(a, []string{"object:delete"}, ui.Object(ui.Destroy))

	sr := srv.Router.PathPrefix("/objects").Subrouter()
	sr.HandleFunc("", index).Methods("GET")
	sr.HandleFunc("/create", create).Methods("GET")
	sr.HandleFunc("", store).Methods("POST")
	sr.HandleFunc("/{object:[0-9]+}", show).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}/download/{name}", show).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}", destroy).Methods("DELETE")
	sr.Use(srv.CSRF)
}
