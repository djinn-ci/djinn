package web

import (
	"database/sql"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/namespace"

	"github.com/gorilla/mux"
)

var namespaceMaxLevel int64 = 20

type Namespace struct {
	Handler
}

func NewNamespace(h Handler) Namespace {
	return Namespace{Handler: h}
}

func namespaceFromRequest(r *http.Request) (*model.Namespace, error) {
	vars := mux.Vars(r)

	u, err := model.FindUserByUsername(vars["username"])

	if err != nil {
		return nil, errors.Err(err)
	}

	n, err := u.FindNamespaceByFullName(vars["namespace"])

	if err != nil {
		return nil, errors.Err(err)
	}

	if !n.IsZero() {
		n.User = u
	}

	return n, nil
}

func (h Namespace) Index(w http.ResponseWriter, r *http.Request) {
	u, err := h.userFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	var namespaces []*model.Namespace

	search := r.URL.Query().Get("search")

	if search != "" {
		namespaces, err = u.NamespacesLike(search)
	} else {
		namespaces, err = u.Namespaces()
	}

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &namespace.IndexPage{
		Page: &template.Page{
			URI: r.URL.Path,
		},
		Namespaces: namespaces,
		Search:     search,
	}

	d := template.NewDashboard(p, r.URL.Path)

	HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Create(w http.ResponseWriter, r *http.Request) {
	u, err := h.userFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	n, err := u.FindNamespaceByFullName(r.URL.Query().Get("parent"))

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

	HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Store(w http.ResponseWriter, r *http.Request) {
	u, err := h.userFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	f := &form.Namespace{
		UserID:   u.ID,
	}

	if err := h.handleRequestData(f, w, r); err != nil {
		return
	}

	parent, err := u.FindNamespaceByFullName(f.Parent)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	n := model.Namespace{
		UserID:      u.ID,
		Name:        f.Name,
		FullName:    f.Name,
		Description: f.Description,
		Visibility:  f.Visibility,
	}

	if !parent.IsZero() {
		n.ParentID = sql.NullInt64{
			Int64: parent.ID,
			Valid: true,
		}
		n.FullName = strings.Join([]string{parent.FullName, f.Name}, "/")
		n.Level = parent.Level + 1
	}

	if n.Level >= namespaceMaxLevel {
		errs := form.NewErrors()
		errs.Put("namespace", form.ErrNamespaceTooDeep)

		h.flashErrors(w, r, errs)
		h.flashForm(w, r, f)

		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if err := n.Create(); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	n.User = u

	http.Redirect(w, r, n.URI(), http.StatusSeeOther)
}

func (h Namespace) Show(w http.ResponseWriter, r *http.Request) {
	n, err := namespaceFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if n.IsZero() || n.User.IsZero() {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	auth, err := h.userFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	switch n.Visibility {
		case model.Private:
			if n.UserID != auth.ID {
				HTMLError(w, "Not found", http.StatusNotFound)
				return
			}
		case model.Internal:
			if auth.IsZero() {
				HTMLError(w, "Not found", http.StatusNotFound)
				return
			}
		case model.Public:
			break
	}

	if err := n.LoadParents(); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if n.Parent != nil {
		n.Parent.User = n.User
	}

	if filepath.Base(r.URL.Path) == "namespaces" {
		var namespaces []*model.Namespace

		search := r.URL.Query().Get("search")

		if search != "" {
			namespaces, err = n.NamespacesLike(search)
		} else {
			namespaces, err = n.Namespaces()
		}

		if err := model.LoadNamespaceRelations(namespaces); err != nil {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p := &namespace.ShowNamespacesPage{
			ShowPage: &namespace.ShowPage{
				Page:      &template.Page{
					URI: r.URL.Path,
				},
				Namespace: n,
			},
			Namespaces: namespaces,
			Search:     search,
		}

		d := template.NewDashboard(p, r.URL.Path)

		HTML(w, template.Render(d), http.StatusOK)
		return
	}

	builds, err := n.Builds()

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := model.LoadBuildRelations(builds); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &namespace.ShowPage{
		Page:      &template.Page{
			URI: r.URL.Path,
		},
		Namespace: n,
		Builds:    builds,
	}

	d := template.NewDashboard(p, r.URL.Path)

	HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Edit(w http.ResponseWriter, r *http.Request) {
	auth, err := h.userFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	n, err := namespaceFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if n.IsZero() || n.User.IsZero() || n.UserID != auth.ID {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if err := n.LoadParents(); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &namespace.EditPage{
		Errors:    h.errors(w, r),
		Form:      h.form(w, r),
		Namespace: n,
	}

	d := template.NewDashboard(p, r.URL.Path)

	HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Update(w http.ResponseWriter, r *http.Request) {
	auth, err := h.userFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	n, err := namespaceFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if n.IsZero() || n.User.IsZero() || n.UserID != auth.ID {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	f := &form.Namespace{
		UserID:    n.UserID,
		Namespace: n,
	}

	if err := h.handleRequestData(f, w, r); err != nil {
		return
	}

	n.Name = f.Name
	n.Description = f.Description
	n.Visibility = f.Visibility

	if err := n.Update(); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, n.URI(), http.StatusSeeOther)
}

func (h Namespace) Destroy(w http.ResponseWriter, r *http.Request) {
	auth, err := h.userFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	n, err := namespaceFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if n.IsZero() || n.User.IsZero() || n.UserID != auth.ID {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if err := n.Destroy(); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/namespaces", http.StatusSeeOther)
}
