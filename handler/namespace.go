package handler

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/namespace"

	"github.com/gorilla/mux"
)

type Namespace struct {
	Handler
}

func NewNamespace(h Handler) Namespace {
	return Namespace{Handler: h}
}

func (h Namespace) Index(w http.ResponseWriter, r *http.Request) {
	u, err := h.UserFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	namespaces, err := u.Namespaces()

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &namespace.IndexPage{
		Namespaces: namespaces,
	}

	d := template.NewDashboard(p, r.URL.RequestURI())

	html(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Create(w http.ResponseWriter, r *http.Request) {
	u, err := h.UserFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	n, err := u.FindNamespaceByName(r.URL.Query().Get("parent"))

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &namespace.CreatePage{
		Errors:    h.errors(w, r),
		Form:      h.form(w, r),
		Parent:    n,
	}

	d := template.NewDashboard(p, r.URL.RequestURI())

	html(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Store(w http.ResponseWriter, r *http.Request) {
	u, err := h.UserFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	f := &form.CreateNamespace{
		UserID: u.ID,
	}

	if err := unmarshalForm(f, r); err != nil {
		log.Error.Println(errors.Err(err))
		return
	}

	if err := f.Validate(); err != nil {
		h.flashErrors(w, r, err.(form.Errors))
		h.flashForm(w, r, f)

		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	parent, err := u.FindNamespaceByName(f.Parent)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	n := model.Namespace{
		UserID:      u.ID,
		Name:        f.Name,
		Description: f.Description,
		Visibility:  model.ParseVisibility(f.Visibility),
	}

	if !parent.IsZero() {
		n.ParentID = sql.NullInt64{
			Int64: parent.ID,
			Valid: true,
		}
		n.Name = strings.Join([]string{parent.Name, f.Name}, "/")
	}

	if err := n.Create(); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/namespaces", http.StatusSeeOther)
}

func (h Namespace) Show(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	u, err := model.FindUserByUsername(vars["username"])

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if u.IsZero() {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	n, err := u.FindNamespaceByName(vars["namespace"])

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if n.IsZero() {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	p := &namespace.ShowPage{
		User:      u,
		Namespace: n,
	}

	d := template.NewDashboard(p, r.URL.RequestURI())

	html(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Edit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	html(w, vars["namespace"], http.StatusOK)
}
