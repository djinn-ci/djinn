package http

import (
	"net/http"

	"djinn-ci.com/alert"
	"djinn-ci.com/errors"
	"djinn-ci.com/key"
	keytemplate "djinn-ci.com/key/template"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"

	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
)

type UI struct {
	*Handler
}

func (h UI) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	kk, paginator, err := h.IndexWithRelations(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := key.LoadNamespaces(h.DB, kk...); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	search := r.URL.Query().Get("search")
	csrf := csrf.TemplateField(r)

	p := &keytemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrf,
		Search:    search,
		Paginator: paginator,
		Keys:      kk,
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Create(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrf := csrf.TemplateField(r)

	p := &keytemplate.Form{
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

	k, f, err := h.StoreModel(u, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			if errs, ok := verrs["fatal"]; ok {
				h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
				alert.Flash(sess, alert.Danger, "Failed to create key")
				h.RedirectBack(w, r)
				return
			}

			webutil.FlashFormWithErrors(sess, f, verrs)
			h.RedirectBack(w, r)
			return
		}

		errs := webutil.NewValidationErrors()

		switch cause {
		case namespace.ErrName:
			errs.Add("namespace", cause)

			sess.AddFlash(errs, "form_errors")
			h.RedirectBack(w, r)
		case namespace.ErrPermission, namespace.ErrOwner:
			alert.Flash(sess, alert.Danger, "Failed to create key: could not add to namespace")
			h.RedirectBack(w, r)
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to create key")
			h.RedirectBack(w, r)
		}
		return
	}

	alert.Flash(sess, alert.Success, "Key has been added: "+k.Name)
	h.Redirect(w, r, "/keys")
}

func (h UI) Edit(u *user.User, k *key.Key, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrf := csrf.TemplateField(r)

	p := &keytemplate.Form{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		Key: k,
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Update(u *user.User, k *key.Key, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	k, f, err := h.UpdateModel(k, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			if errs, ok := verrs["fatal"]; ok {
				h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
				alert.Flash(sess, alert.Danger, "Failed to update key")
				h.RedirectBack(w, r)
				return
			}

			webutil.FlashFormWithErrors(sess, f, verrs)
			h.RedirectBack(w, r)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	alert.Flash(sess, alert.Success, "Key has been updated: "+k.Name)
	h.Redirect(w, r, "/keys")
}

func (h UI) Destroy(u *user.User, k *key.Key, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r.Context(), k); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to delete key")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Key has been deleted")
	h.RedirectBack(w, r)
}

func RegisterUI(srv *server.Server) {
	user := userhttp.NewHandler(srv)

	ui := UI{
		Handler: NewHandler(srv),
	}

	sr := srv.Router.PathPrefix("/keys").Subrouter()
	sr.HandleFunc("", user.WithUser(ui.Index)).Methods("GET")
	sr.HandleFunc("/create", user.WithUser(ui.Create)).Methods("GET")
	sr.HandleFunc("", user.WithUser(ui.Store)).Methods("POST")
	sr.HandleFunc("/{key:[0-9]+}/edit", user.WithUser(ui.WithKey(ui.Edit))).Methods("GET")
	sr.HandleFunc("/{key:[0-9]+}", user.WithUser(ui.WithKey(ui.Update))).Methods("PATCH")
	sr.HandleFunc("/{key:[0-9]+}", user.WithUser(ui.WithKey(ui.Destroy))).Methods("DELETE")
	sr.Use(srv.CSRF)
}
