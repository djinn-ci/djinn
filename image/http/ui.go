package http

import (
	"net/http"
	"regexp"

	"djinn-ci.com/alert"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/image"
	imagetemplate "djinn-ci.com/image/template"
	"djinn-ci.com/namespace"
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

	ii, paginator, err := h.IndexWithRelations(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := image.LoadNamespaces(h.DB, ii...); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	csrf := csrf.TemplateField(r)

	p := &imagetemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrf,
		Paginator: paginator,
		Images:    ii,
		Search:    r.URL.Query().Get("search"),
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Create(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrf := csrf.TemplateField(r)

	p := &imagetemplate.Create{
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

	i, f, err := h.StoreModel(u, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			if errs, ok := verrs["fatal"]; ok {
				h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
				alert.Flash(sess, alert.Danger, "Failed to create image")
				h.RedirectBack(w, r)
				return
			}

			webutil.FlashFormWithErrors(sess, f, verrs)
			h.RedirectBack(w, r)
			return
		}

		errs := webutil.NewValidationErrors()

		switch cause {
		case image.ErrInvalidScheme:
			errs.Add("download_url", cause)

			sess.AddFlash(errs, "form_errors")
			h.RedirectBack(w, r)
			return
		case image.ErrMissingPassword:
			errs.Add("download_url", cause)

			sess.AddFlash(errs, "form_errors")
			h.RedirectBack(w, r)
			return
		case namespace.ErrName:
			errs.Add("namespace", cause)

			sess.AddFlash(errs, "form_errors")
			h.RedirectBack(w, r)
			return
		case namespace.ErrPermission, namespace.ErrOwner:
			alert.Flash(sess, alert.Danger, "Failed to create image: could not add to namespace")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to create image")
			h.RedirectBack(w, r)
			return
		}
	}

	alert.Flash(sess, alert.Success, "Image has been added: "+i.Name)
	h.Redirect(w, r, "/images")
}

func (h UI) Show(u *user.User, i *image.Image, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	if i.Name != vars["name"] {
		h.NotFound(w, r)
		return
	}

	rec, err := i.Data(h.Images.Store)

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			h.NotFound(w, r)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	defer rec.Close()

	w.Header().Set("Content-Type", image.MimeTypeQEMU)
	http.ServeContent(w, r, i.Name, i.CreatedAt, rec)
}

var reimageuri = regexp.MustCompile("/images/[0-9]+")

func (h UI) Destroy(u *user.User, i *image.Image, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r.Context(), i); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to delete image")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Image has been deleted")

	if matched, _ := regexp.Match("/images/[0-9]+", []byte(r.Header.Get("Referer"))); matched {
		h.Redirect(w, r, "/images")
		return
	}

	if reimageuri.Match([]byte(r.Header.Get("Referer"))) {
		h.Redirect(w, r, "/images")
		return
	}
	h.RedirectBack(w, r)
}

func RegisterUI(srv *server.Server) {
	user := userhttp.NewHandler(srv)

	ui := UI{
		Handler: NewHandler(srv),
	}

	sr := srv.Router.PathPrefix("/images").Subrouter()
	sr.HandleFunc("", user.WithUser(ui.Index)).Methods("GET")
	sr.HandleFunc("/create", user.WithUser(ui.Create)).Methods("GET")
	sr.HandleFunc("", user.WithUser(ui.Store)).Methods("POST")
	sr.HandleFunc("/{image:[0-9]+}/download/{name}", user.WithUser(ui.WithImage(ui.Show))).Methods("GET")
	sr.HandleFunc("/{image:[0-9]+}", user.WithUser(ui.WithImage(ui.Destroy))).Methods("DELETE")
	sr.Use(srv.CSRF)
}
