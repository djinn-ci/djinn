package ui

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
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

var namespaceMaxLevel int64 = 20

type Namespace struct {
	web.Handler

	Namespaces *model.NamespaceStore
}

func NewNamespace(h web.Handler, namespaces *model.NamespaceStore) Namespace {
	return Namespace{
		Handler:    h,
		Namespaces: namespaces,
	}
}

func (h Namespace) namespace(r *http.Request) (*model.Namespace, error) {
	vars := mux.Vars(r)

	u, err := h.Users.FindByUsername(vars["username"])

	if err != nil {
		return nil, errors.Err(err)
	}

	n, err := u.NamespaceStore().FindByPath(vars["namespace"])

	if err != nil {
		return nil, errors.Err(err)
	}

	return n, nil
}

func (h Namespace) Index(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	var nn []*model.Namespace

	search := r.URL.Query().Get("search")

	if search != "" {
		nn, err = u.NamespaceStore().Like(search)
	} else {
		nn, err = u.NamespaceStore().All()
	}

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &namespace.IndexPage{
		Page: template.Page{
			URI: r.URL.Path,
		},
		Namespaces: nn,
		Search:     search,
	}

	d := template.NewDashboard(p, r.URL.Path)

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Create(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	parent, err := u.NamespaceStore().FindByPath(r.URL.Query().Get("parent"))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &namespace.CreatePage{
		Form: template.Form{
			CSRF:   csrf.TemplateField(r),
			Errors: h.Errors(w, r),
			Form:   h.Form(w, r),
		},
	}

	if !parent.IsZero() {
		p.Parent = parent
	}

	d := template.NewDashboard(p, r.URL.RequestURI())

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Store(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	namespaces := u.NamespaceStore()

	f := &form.Namespace{
		UserID: u.ID,
	}

	f.Bind(namespaces, nil)

	if err := h.ValidateForm(f, w, r); err != nil {
		if _, ok := err.(form.Errors); ok {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	parent, err := namespaces.FindByPath(f.Parent)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	n := namespaces.New()
	n.Name = f.Name
	n.Path = f.Name
	n.Description = f.Description
	n.Visibility = f.Visibility

	if !parent.IsZero() {
		n.RootID = parent.RootID
		n.ParentID = sql.NullInt64{
			Int64: parent.ID,
			Valid: true,
		}
		n.Path = strings.Join([]string{parent.Path, f.Name}, "/")
		n.Level = parent.Level + 1
		n.Visibility = parent.Visibility
	}

	if n.Level >= namespaceMaxLevel {
		errs := form.NewErrors()
		errs.Put("namespace", form.ErrNamespaceTooDeep)

		h.FlashErrors(w, r, errs)
		h.FlashForm(w, r, f)

		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if err := n.Create(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if parent.IsZero() {
		n.RootID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}

		if err := n.Update(); err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, n.UIEndpoint(), http.StatusSeeOther)
}

func (h Namespace) Show(w http.ResponseWriter, r *http.Request) {
	n, err := h.namespace(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if n.IsZero() || n.User.IsZero() {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	switch n.Visibility {
		case model.Private:
			if n.UserID != u.ID {
				web.HTMLError(w, "Not found", http.StatusNotFound)
				return
			}
		case model.Internal:
			if u.IsZero() {
				web.HTMLError(w, "Not found", http.StatusNotFound)
				return
			}
		case model.Public:
			break
	}

	if err := n.LoadParents(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := namespace.ShowPage{
		Page: template.Page{
			URI: r.URL.Path,
		},
		Namespace: n,
	}

	namespaces := n.NamespaceStore()

	if filepath.Base(r.URL.Path) == "namespaces" {
		var nn []*model.Namespace

		search := r.URL.Query().Get("search")

		if search != "" {
			nn, err = namespaces.Like(search)
		} else {
			nn, err = namespaces.All()
		}

		if err := namespaces.LoadUsers(nn); err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		np := &namespace.ShowNamespacesPage{
			ShowPage:   p,
			Namespaces: nn,
			Search:     search,
		}

		d := template.NewDashboard(np, r.URL.Path)

		web.HTML(w, template.Render(d), http.StatusOK)
		return
	}

	builds := n.BuildStore()

	bb, err := builds.All()

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := builds.LoadNamespaces(bb); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := builds.LoadTags(bb); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := builds.LoadUsers(bb); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	nn := make([]*model.Namespace, 0)

	for _, b := range bb {
		nn = append(nn, b.Namespace)
	}

	if err := namespaces.LoadUsers(nn); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p.Builds = bb

	d := template.NewDashboard(&p, r.URL.Path)

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Edit(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	n, err := h.namespace(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if n.IsZero() || n.User.IsZero() || n.UserID != u.ID {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if err := n.LoadParents(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &namespace.EditPage{
		Form: template.Form{
			CSRF:   csrf.TemplateField(r),
			Errors: h.Errors(w, r),
			Form:   h.Form(w, r),
		},
		Namespace: n,
	}

	d := template.NewDashboard(p, r.URL.Path)

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Update(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	n, err := h.namespace(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if n.IsZero() || n.User.IsZero() || n.UserID != u.ID {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	f := &form.Namespace{
		UserID: n.UserID,
	}

	f.Bind(*h.Namespaces, n)

	if err := h.ValidateForm(f, w, r); err != nil {
		if _, ok := err.(form.Errors); ok {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := n.LoadParent(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	n.Name = f.Name
	n.Description = f.Description
	n.Visibility = f.Visibility

	if !n.Parent.IsZero() {
		n.Visibility = n.Parent.Visibility
	} else {
		if err := n.CascadeVisibility(); err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	if err := n.Update(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, n.UIEndpoint(), http.StatusSeeOther)
}

func (h Namespace) Destroy(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	n, err := h.namespace(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if n.IsZero() || n.User.IsZero() || n.UserID != u.ID {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if err := n.Destroy(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/namespaces", http.StatusSeeOther)
}
