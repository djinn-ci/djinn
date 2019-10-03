package ui

import (
	"net/http"
	"path/filepath"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/namespace"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/core"

	"github.com/gorilla/csrf"
)

type Namespace struct {
	Core core.Namespace

	Build      Build
	Object     Object
	Variable   Variable
	Key        Key
}

func (h Namespace) Index(w http.ResponseWriter, r *http.Request) {
	u := h.Core.User(r)

	nn, paginator, err := h.Core.Index(u.NamespaceStore(), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	search := r.URL.Query().Get("search")

	p := &namespace.IndexPage{
		BasePage:   template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Paginator:  paginator,
		Namespaces: nn,
		Search:     search,
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Create(w http.ResponseWriter, r *http.Request) {
	u := h.Core.User(r)

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
			Errors: h.Core.Errors(w, r),
			Fields: h.Core.Form(w, r),
		},
	}

	if !parent.IsZero() {
		p.Parent = parent
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Store(w http.ResponseWriter, r *http.Request) {
	n, err := h.Core.Store(w, r)

	if err != nil {
		cause := errors.Cause(err)

		if cause == core.ErrValidationFailed {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		if cause == core.ErrNamespaceTooDeep {
			errs := form.NewErrors()
			errs.Put("namespace", cause)

			h.Core.FlashErrors(w, r, errs)

			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		h.Core.FlashAlert(w, r, template.Danger("Failed to create namespace: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, n.UIEndpoint(), http.StatusSeeOther)
}

func (h Namespace) Show(w http.ResponseWriter, r *http.Request) {
	u := h.Core.User(r)
	n := h.Core.Namespace(r)

	if err := n.LoadParents(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	base := filepath.Base(r.URL.Path)

	search := r.URL.Query().Get("search")

	var p template.Dashboard

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}

	sp := namespace.ShowPage{
		BasePage:  bp,
		Namespace: n,
	}

	switch base {
	case "namespaces":
		nn, paginator, err := h.Core.Index(n.NamespaceStore(), r)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p = &namespace.ShowNamespaces{
			ShowPage:   sp,
			Index:      namespace.IndexPage{
				BasePage:   bp,
				Paginator:  paginator,
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
			Fields:        h.Core.Form(w, r),
			Errors:        h.Core.Errors(w, r),
			Collaborators: cc,
		}

		break
	default:
		var err error

		sp.Index, err = h.Build.indexPage(n.BuildStore(), r)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p = &sp

		break
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Edit(w http.ResponseWriter, r *http.Request) {
	n := h.Core.Namespace(r)

	if err := n.LoadParents(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &namespace.Form{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Core.Errors(w, r),
			Fields: h.Core.Form(w, r),
		},
		Namespace: n,
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Update(w http.ResponseWriter, r *http.Request) {
	n, err := h.Core.Update(w, r)

	if err != nil {
		cause := errors.Cause(err)

		if _, ok := cause.(form.Errors); ok {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		h.Core.FlashAlert(w, r, template.Danger("Failed to update namespace: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("Namespace changes saved"))

	http.Redirect(w, r, n.UIEndpoint(), http.StatusSeeOther)
}

func (h Namespace) Destroy(w http.ResponseWriter, r *http.Request) {
	n := h.Core.Namespace(r)

	if err := h.Core.Destroy(r); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.Core.FlashAlert(w, r, template.Danger("Failed to delete namespace: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("Namespace has been deleted: " + n.Path))

	http.Redirect(w, r, "/namespaces", http.StatusSeeOther)
}
