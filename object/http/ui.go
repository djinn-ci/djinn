package http

import (
	"net/http"
	"regexp"
	"strings"

	"djinn-ci.com/alert"
	buildtemplate "djinn-ci.com/build/template"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"
	objecttemplate "djinn-ci.com/object/template"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"

	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type UI struct {
	*Handler
}

func (h UI) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	oo, paginator, err := h.IndexWithRelations(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := object.LoadNamespaces(h.DB, oo...); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	csrf := csrf.TemplateField(r)

	p := &objecttemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrf,
		Paginator: paginator,
		Objects:   oo,
		Search:    r.URL.Query().Get("search"),
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Create(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrf := csrf.TemplateField(r)

	p := &objecttemplate.Create{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	o, f, err := h.StoreModel(u, r)

	if err != nil {
		cause := errors.Cause(err)

		errs := webutil.NewValidationErrors()

		switch err := cause.(type) {
		case webutil.ValidationErrors:
			if errs, ok := err["fatal"]; ok {
				h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
				alert.Flash(sess, alert.Danger, "Failed to create object")
				h.RedirectBack(w, r)
				return
			}
			errs = err
		case *namespace.PathError:
			errs.Add("namespace", err)
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to create object")
			h.RedirectBack(w, r)
			return
		}

		webutil.FlashFormWithErrors(sess, f, errs)
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Object has been added: "+o.Name)
	h.Redirect(w, r, "/objects")
}

func (h UI) Show(u *user.User, o *object.Object, w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")

	if parts[len(parts)-2] == "download" {
		if o.Name != mux.Vars(r)["name"] {
			h.NotFound(w, r)
			return
		}

		rec, err := o.Data(h.Objects.Store)

		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				h.NotFound(w, r)
				return
			}

			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		defer rec.Close()

		http.ServeContent(w, r, o.Name, o.CreatedAt, rec)
		return
	}

	sess, save := h.Session(r)

	if err := object.LoadRelations(h.DB, o); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := namespace.Load(h.DB, o); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	bb, paginator, err := h.getBuilds(o, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	mm := make([]database.Model, 0, len(bb))

	for _, b := range bb {
		mm = append(mm, b)
	}

	if err := namespace.Load(h.DB, mm...); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	csrf := csrf.TemplateField(r)

	bp := template.BasePage{
		URL:   r.URL,
		Query: r.URL.Query(),
		User:  u,
	}
	p := &objecttemplate.Show{
		BasePage: bp,
		CSRF:     csrf,
		Object:   o,
		Section: &buildtemplate.Index{
			BasePage:  bp,
			Paginator: paginator,
			Builds:    bb,
		},
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

var reobjecturi = regexp.MustCompile("/objects/[0-9]+")

func (h UI) Destroy(u *user.User, o *object.Object, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r.Context(), o); err != nil {
		h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to delete object")
		return
	}

	alert.Flash(sess, alert.Success, "Object has been deleted")

	if reobjecturi.Match([]byte(r.Header.Get("Referer"))) {
		h.Redirect(w, r, "/objects")
		return
	}
	h.RedirectBack(w, r)
}

func RegisterUI(srv *server.Server) {
	user := userhttp.NewHandler(srv)

	ui := UI{
		Handler: NewHandler(srv),
	}

	sr := srv.Router.PathPrefix("/objects").Subrouter()
	sr.HandleFunc("", user.WithUser(ui.Index)).Methods("GET")
	sr.HandleFunc("/create", user.WithUser(ui.Create)).Methods("GET")
	sr.HandleFunc("", user.WithUser(ui.Store)).Methods("POST")
	sr.HandleFunc("/{object:[0-9]+}", user.WithUser(ui.WithObject(ui.Show))).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}/download/{name}", user.WithUser(ui.WithObject(ui.Show))).Methods("GET")
	sr.HandleFunc("/{object:[0-9]+}", user.WithUser(ui.WithObject(ui.Destroy))).Methods("DELETE")
	sr.Use(srv.CSRF)
}
