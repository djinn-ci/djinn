package ui

import (
	"database/sql"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/namespace"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
)

type Namespace struct {
	web.Handler

	Namespaces model.NamespaceStore
	Object     Object
	Variable   Variable
	Key        Key
}

func (h Namespace) Namespace(r *http.Request) *model.Namespace {
	val := r.Context().Value("namespace")

	n, _ := val.(*model.Namespace)

	return n
}

func (h Namespace) Index(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	search := r.URL.Query().Get("search")

	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts := []query.Option{
		model.Search("path", search),
		model.NamespaceSharedWith(u),
	}

	paginator, err := u.NamespaceStore().Paginate(page, opts...)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	opts = append(
		opts,
		query.OrderAsc("path"),
		query.Limit(model.PageLimit),
		query.Offset(paginator.Offset),
	)

	nn, err := u.NamespaceStore().Index(opts...)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &namespace.IndexPage{
		BasePage:   template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Paginator:  paginator,
		Namespaces: nn,
		Search:     search,
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Create(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	parent, err := u.NamespaceStore().FindByPath(r.URL.Query().Get("parent"))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if parent.Level + 1 > model.NamespaceMaxDepth {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	p := &namespace.Form{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Errors(w, r),
			Fields: h.Form(w, r),
		},
	}

	if !parent.IsZero() {
		p.Parent = parent
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Store(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	namespaces := u.NamespaceStore()

	f := &form.Namespace{
		Namespaces: namespaces,
		UserID:     u.ID,
	}

	if err := h.ValidateForm(f, w, r); err != nil {
		if _, ok := err.(form.Errors); ok {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create namespace: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	parent, err := namespaces.FindByPath(f.Parent)

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create namespace: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
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

	if n.Level >= model.NamespaceMaxDepth {
		h.FlashAlert(w, r, template.Warn("Namespaces can only be nested to 20 levels"))
		h.FlashForm(w, r, f)

		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if err := namespaces.Create(n); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create namespace: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if parent.IsZero() {
		n.RootID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}

		if err := namespaces.Update(n); err != nil {
			log.Error.Println(errors.Err(err))
			h.FlashAlert(w, r, template.Danger("Failed to create namespace: " + errors.Cause(err).Error()))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}
	}

	http.Redirect(w, r, n.UIEndpoint(), http.StatusSeeOther)
}

func (h Namespace) Show(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)
	n := h.Namespace(r)

	if err := n.LoadParents(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	base := filepath.Base(r.URL.Path)

	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	var p template.Dashboard

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}

	sp := namespace.ShowPage{
		BasePage:  bp,
		Namespace: n,
		Status:    status,
		Search:    search,
	}

	switch base {
	case "namespaces":
		nn, err := n.NamespaceStore().Index(model.Search("path", search))

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p = &namespace.ShowNamespaces{
			ShowPage:   sp,
			Index:      namespace.IndexPage{
				BasePage:   bp,
				Namespaces: nn,
				Search:     search,
			},
		}

		break
	case "objects":
		op, err := h.Object.indexPage(n.ObjectStore(), r)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p = &namespace.ShowObjects{
			ShowPage: sp,
			Index:    op,
		}

		break
	case "variables":
		vp, err := h.Variable.indexPage(n.VariableStore(), r)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p = &namespace.ShowVariables{
			ShowPage: sp,
			Index:    vp,
		}

		break
	case "keys":
		kp, err := h.Key.indexPage(n.KeyStore(), r)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p = &namespace.ShowKeys{
			ShowPage: sp,
			Index:    kp,
		}

		break
	case "collaborators":
		cc, err := n.CollaboratorStore().Index()

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p = &namespace.ShowCollaborators{
			ShowPage:      sp,
			CSRF:          string(csrf.TemplateField(r)),
			Fields:        h.Form(w, r),
			Errors:        h.Errors(w, r),
			Collaborators: cc,
		}

		break
	default:
		var err error

		sp.Builds, err = n.BuildStore().Index(
			model.BuildStatus(status),
			model.BuildSearch(search),
			query.OrderDesc("created_at"),
		)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p = &sp

		break
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Edit(w http.ResponseWriter, r *http.Request) {
	n := h.Namespace(r)

	if err := n.LoadParents(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &namespace.Form{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Errors(w, r),
			Fields: h.Form(w, r),
		},
		Namespace: n,
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Update(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)
	n := h.Namespace(r)

	f := &form.Namespace{
		Namespaces: u.NamespaceStore(),
		Namespace:  n,
		UserID:     n.UserID,
	}

	if err := h.ValidateForm(f, w, r); err != nil {
		if _, ok := err.(form.Errors); ok {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to update namespace: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if err := n.LoadParent(); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to update namespace: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
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
			h.FlashAlert(w, r, template.Danger("Failed to update namespace: " + errors.Cause(err).Error()))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}
	}

	if err := h.Namespaces.Update(n); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to update namespace: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Namespace changes saved"))

	http.Redirect(w, r, n.UIEndpoint(), http.StatusSeeOther)
}

func (h Namespace) Destroy(w http.ResponseWriter, r *http.Request) {
	n := h.Namespace(r)

	if err := h.Namespaces.Delete(n); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to delete namespace: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Namespace has been deleted: " + n.Path))

	http.Redirect(w, r, "/namespaces", http.StatusSeeOther)
}
