package http

import (
	"io"
	"net/http"
	"regexp"

	"djinn-ci.com/alert"
	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/template/form"

	"github.com/andrewpillar/fs"
	"github.com/gorilla/mux"
)

type UI struct {
	*Handler
}

func (h UI) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	p, err := h.Handler.Index(u, r)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get images"))
		return
	}

	ld := namespace.Loader(h.DB)

	mm := database.Map[*image.Image, database.Model](p.Items, func(i *image.Image) database.Model {
		return i
	})

	if err := ld.Load(r.Context(), "namespace_id", "id", mm...); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to load namespaces"))
		return
	}

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.ImageIndex{
		Paginator: template.NewPaginator[*image.Image](tmpl.Page, p),
		Images:    p.Items,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Create(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.ImageCreate{
		Form: form.New(sess, r),
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	i, f, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, f, errors.Wrap(err, "Failed to create image"))
		return
	}

	alert.Flash(sess, alert.Success, "Image has been added: "+i.Name)
	h.Redirect(w, r, "/images")
}

func (h UI) Show(u *auth.User, i *image.Image, w http.ResponseWriter, r *http.Request) {
	if i.Name != mux.Vars(r)["name"] {
		h.NotFound(w, r)
		return
	}

	f, err := i.Open(h.Images)

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			h.NotFound(w, r)
			return
		}
		h.Error(w, r, errors.Wrap(err, "Failed to get image"))
		return
	}

	defer f.Close()

	w.Header().Set("Content-Type", image.MimeTypeQEMU)
	http.ServeContent(w, r, i.Name, i.CreatedAt, f.(io.ReadSeeker))
}

var reImageURI = regexp.MustCompile("/images/[0-9]+")

func (h UI) Destroy(u *auth.User, i *image.Image, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.Handler.Destroy(r.Context(), i); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to delete image"))
		return
	}

	alert.Flash(sess, alert.Success, "Image has been deleted")

	if reImageURI.Match([]byte(r.Header.Get("Referer"))) {
		h.Redirect(w, r, "/images")
		return
	}
	h.RedirectBack(w, r)
}

func RegisterUI(a auth.Authenticator, srv *server.Server) {
	ui := UI{
		Handler: NewHandler(srv),
	}

	index := ui.Restrict(a, []string{"image:read"}, ui.Index)
	create := ui.Restrict(a, []string{"image:write"}, ui.Create)
	store := ui.Restrict(a, []string{"image:write"}, ui.Store)

	a = namespace.NewAuth[*image.Image](a, "image", ui.Images.Store)

	show := ui.Optional(a, ui.Image(ui.Show))
	destroy := ui.Restrict(a, []string{"image:write"}, ui.Image(ui.Destroy))

	sr := srv.Router.PathPrefix("/images").Subrouter()
	sr.HandleFunc("", index).Methods("GET")
	sr.HandleFunc("/create", create).Methods("GET")
	sr.HandleFunc("", store).Methods("POST")
	sr.HandleFunc("/{image:[0-9]+}/download/{name}", show).Methods("GET")
	sr.HandleFunc("/{image:[0-9]+}", destroy).Methods("DELETE")
	sr.Use(srv.CSRF)
}
