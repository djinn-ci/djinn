package http

import (
	"net/http"

	"djinn-ci.com/alert"
	"djinn-ci.com/auth"
	"djinn-ci.com/errors"
	"djinn-ci.com/key"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/template/form"
)

type UI struct {
	*Handler
}

func (h UI) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	p, err := h.Handler.Index(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get keys"))
		return
	}

	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.KeyIndex{
		Paginator: template.NewPaginator[*key.Key](tmpl.Page, p),
		Keys:      p.Items,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Create(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.KeyForm{
		Form: form.New(sess, r),
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	k, f, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, f, err)
		return
	}

	alert.Flash(sess, alert.Success, "Key has been added: "+k.Name)
	h.Redirect(w, r, "/keys")
}

func (h UI) Edit(u *auth.User, k *key.Key, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.KeyForm{
		Form: form.NewWithModel(sess, r, k),
		Key:  k,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Update(u *auth.User, k *key.Key, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	k, f, err := h.Handler.Update(u, k, r)

	if err != nil {
		h.FormError(w, r, f, err)
		return
	}

	alert.Flash(sess, alert.Success, "Key has been updated: "+k.Name)
	h.Redirect(w, r, "/keys")
}

func (h UI) Destroy(u *auth.User, k *key.Key, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.Handler.Destroy(r.Context(), k); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to delete key"))
		return
	}

	alert.Flash(sess, alert.Success, "Key has been deleted")
	h.RedirectBack(w, r)
}

func RegisterUI(a auth.Authenticator, srv *server.Server) {
	ui := UI{
		Handler: NewHandler(srv),
	}

	index := ui.Restrict(a, []string{"key:read"}, ui.Index)
	create := ui.Restrict(a, []string{"key:write"}, ui.Create)
	store := ui.Restrict(a, []string{"key:write"}, ui.Store)

	a = namespace.NewAuth(a, "key", ui.Keys.Store)

	edit := ui.Restrict(a, []string{"key:write"}, ui.Key(ui.Edit))
	update := ui.Restrict(a, []string{"key:write"}, ui.Key(ui.Update))
	destroy := ui.Restrict(a, []string{"key:delete"}, ui.Key(ui.Destroy))

	sr := srv.Router.PathPrefix("/keys").Subrouter()
	sr.HandleFunc("", index).Methods("GET")
	sr.HandleFunc("/create", create).Methods("GET")
	sr.HandleFunc("", store).Methods("POST")
	sr.HandleFunc("/{key:[0-9]+}/edit", edit).Methods("GET")
	sr.HandleFunc("/{key:[0-9]+}", update).Methods("PATCH")
	sr.HandleFunc("/{key:[0-9]+}", destroy).Methods("DELETE")
	sr.Use(srv.CSRF)
}
