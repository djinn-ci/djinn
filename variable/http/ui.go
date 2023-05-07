package http

import (
	"net/http"

	"djinn-ci.com/alert"
	"djinn-ci.com/auth"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/template/form"
	"djinn-ci.com/variable"
)

type UI struct {
	*Handler
}

func (h UI) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	p, err := h.Handler.Index(u, r)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get variables"))
		return
	}

	sess, _ := h.Session(r)

	unmasked := variable.GetUnmasked(sess.Values)

	for _, v := range p.Items {
		if _, ok := unmasked[v.ID]; ok && v.Masked {
			if err := variable.Unmask(h.AESGCM, v); err != nil {
				alert.Flash(sess, alert.Danger, "Could not unmask variable")
				h.Log.Error.Println(r.Method, r.URL, "could not unmask variable", errors.Err(err))
			}
			continue
		}

		if v.Masked {
			v.Value = variable.MaskString
		}
	}

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.VariableIndex{
		Paginator: template.NewPaginator[*variable.Variable](tmpl.Page, p),
		Unmasked:  unmasked,
		Variables: p.Items,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Create(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.VariableCreate{
		Form: form.New(sess, r),
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	v, f, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, f, errors.Wrap(err, "Failed to create variable"))
		return
	}

	alert.Flash(sess, alert.Success, "Variable has been added: "+v.Key)
	h.Redirect(w, r, "/variables")
}

func (h UI) Mask(u *auth.User, v *variable.Variable, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if u.ID != v.AuthorID {
		h.RedirectBack(w, r)
		return
	}

	unmasked := variable.GetUnmasked(sess.Values)

	if _, ok := unmasked[v.ID]; ok {
		delete(unmasked, v.ID)
		variable.PutUnmasked(sess.Values, unmasked)
	}
	h.RedirectBack(w, r)
}

func (h UI) Unmask(u *auth.User, v *variable.Variable, w http.ResponseWriter, r *http.Request) {
	h.Log.Debug.Println(r.Method, r.URL, "unmasking variable", v.ID)
	sess, _ := h.Session(r)

	if u.ID != v.AuthorID {
		h.RedirectBack(w, r)
		return
	}

	unmasked := variable.GetUnmasked(sess.Values)
	unmasked[v.ID] = struct{}{}

	variable.PutUnmasked(sess.Values, unmasked)

	h.RedirectBack(w, r)
}

func (h UI) Destroy(u *auth.User, v *variable.Variable, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.Handler.Destroy(r.Context(), v); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to delete variable"))
		return
	}

	alert.Flash(sess, alert.Success, "Variable has been deleted")
	h.RedirectBack(w, r)
}

func RegisterUI(a auth.Authenticator, srv *server.Server) {
	ui := UI{
		Handler: NewHandler(srv),
	}

	index := ui.Restrict(a, []string{"variable:read"}, ui.Index)
	create := ui.Restrict(a, []string{"variable:write"}, ui.Create)
	store := ui.Restrict(a, []string{"variable:write"}, ui.Store)

	a = namespace.NewAuth(a, "variable", ui.Variables.Store)

	mask := ui.Restrict(a, []string{"variable:read"}, ui.Variable(ui.Mask))
	unmask := ui.Sudo(ui.Variable(ui.Unmask))
	destroy := ui.Restrict(a, []string{"variable:delete"}, ui.Variable(ui.Destroy))

	sr := srv.Router.PathPrefix("/variables").Subrouter()
	sr.HandleFunc("", index).Methods("GET")
	sr.HandleFunc("/create", create).Methods("GET")
	sr.HandleFunc("", store).Methods("POST")
	sr.HandleFunc("/{variable:[0-9]+}/mask", mask).Methods("PATCH")
	sr.HandleFunc("/{variable:[0-9]+}/unmask", unmask).Methods("GET")
	sr.HandleFunc("/{variable:[0-9]+}", destroy).Methods("DELETE")
	sr.Use(srv.CSRF)
}
