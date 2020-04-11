package handler

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/andrewpillar/thrall/build"
	buildtemplate "github.com/andrewpillar/thrall/build/template"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/image"
	imagetemplate "github.com/andrewpillar/thrall/image/template"
	"github.com/andrewpillar/thrall/key"
	keytemplate "github.com/andrewpillar/thrall/key/template"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/object"
	objecttemplate "github.com/andrewpillar/thrall/object/template"
	namespacetemplate "github.com/andrewpillar/thrall/namespace/template"
	"github.com/andrewpillar/thrall/template"
	usertemplate "github.com/andrewpillar/thrall/user/template"
	"github.com/andrewpillar/thrall/variable"
	variabletemplate "github.com/andrewpillar/thrall/variable/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type UI struct {
	Namespace

	Invite       InviteUI
	Collaborator CollaboratorUI
}

type InviteUI struct {
	Invite
}

type CollaboratorUI struct {
	Collaborator
}

func (h Namespace) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	nn, paginator, err := h.IndexWithRelations(namespace.NewStore(h.DB, u), r.URL.Query())

	if err  != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &namespacetemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Paginator:  paginator,
		Namespaces: nn,
		Search:     r.URL.Query().Get("search"),
	}

	d := template.NewDashboard(p, r.URL, h.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	parent, err := namespace.NewStore(h.DB, u).Get(
		query.Where("path", "=", r.URL.Query().Get("parent")),
	)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if parent.Level + 1 > namespace.MaxDepth {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	csrfField := string(csrf.TemplateField(r))

	p := &namespacetemplate.Form{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: h.FormErrors(sess),
			Fields: h.FormFields(sess),
		},
	}

	if !parent.IsZero() {
		p.Parent = parent
	}

	d := template.NewDashboard(p, r.URL, h.Alert(sess), string(csrfField))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, err := h.StoreModel(r, sess)

	if err != nil {
		cause := errors.Cause(err)

		if _, ok := cause.(form.Errors); ok {
			h.RedirectBack(w, r)
			return
		}

		switch cause {
		case namespace.ErrDepth:
			sess.AddFlash(template.Danger(strings.Title(cause.Error())), "alert")
			h.RedirectBack(w, r)
			return
		default:
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create namespace"), "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	sess.AddFlash(template.Success("Namespace '"+n.Name+"' created"), "alert")
	h.Redirect(w, r, n.Endpoint())
}

func (h Namespace) loadParents(n *namespace.Namespace) error {
	if !n.ParentID.Valid {
		return nil
	}

	p, err := h.Namespaces.Get(query.Where("id", "=", n.ParentID))

	if err != nil {
		return errors.Err(err)
	}

	n.Parent = p

	if n.ID == n.RootID.Int64 {
		return nil
	}
	return errors.Err(h.loadParents(n.Parent))
}

func (h Namespace) Show(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	n, err := h.Get(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := h.loadParents(n); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()

	base := filepath.Base(r.URL.Path)
	csrfField := string(csrf.TemplateField(r))

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}
	p := &namespacetemplate.Show{
		BasePage:  bp,
		Namespace: n,
	}

	switch base {
	case "namespaces":
		nn, paginator, err := h.IndexWithRelations(namespace.NewStore(h.DB, n), r.URL.Query())

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &namespacetemplate.Index{
			BasePage:   bp,
			Paginator:  paginator,
			Namespace:  n,
			Namespaces: nn,
			Search:     q.Get("search"),
		}
	case "images":
		ii, paginator, err := image.NewStore(h.DB, n).Index(r.URL.Query())

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &imagetemplate.Index{
			BasePage:  bp,
			CSRF:      csrfField,
			Paginator: paginator,
			Images:    ii,
			Search:    q.Get("search"),
		}
	case "objects":
		oo, paginator, err := object.NewStore(h.DB, n).Index(r.URL.Query())

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &objecttemplate.Index{
			BasePage:  bp,
			CSRF:      csrfField,
			Paginator: paginator,
			Objects:   oo,
			Search:    q.Get("search"),
		}
	case "variables":
		vv, paginator, err := variable.NewStore(h.DB, n).Index(r)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &variabletemplate.Index{
			BasePage:  bp,
			CSRF:      csrfField,
			Paginator: paginator,
			Variables: vv,
			Search:    q.Get("search"),
		}
	case "keys":
		kk, paginator, err := key.NewStore(h.DB, n).Index(r.URL.Query())

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &keytemplate.Index{
			BasePage:  bp,
			CSRF:      csrfField,
			Paginator: paginator,
			Keys:      kk,
			Search:    q.Get("search"),
		}
	case "collaborators":
		_, err := namespace.NewCollaboratorStore(h.DB, n).All()

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
		return
	default:
		bb, paginator, err := build.NewStore(h.DB, n).Index(r.URL.Query())

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if err := build.LoadRelations(h.Loaders, bb...); err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		mm := make([]model.Model, 0, len(bb))

		for _, b := range bb {
			if b.NamespaceID.Valid {
				mm = append(mm, b.Namespace)
			}
		}

		err = h.Users.Load("id", model.MapKey("user_id", mm), model.Bind("user_id", "id", mm...))

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Tag = q.Get("tag")
		p.Section = &buildtemplate.Index{
			BasePage:  bp,
			Paginator: paginator,
			Builds:    bb,
			Search:    q.Get("search"),
			Status:    q.Get("status"),
			Tag:       q.Get("tag"),
		}
	}

	d := template.NewDashboard(p, r.URL, h.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Edit(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	n := h.Model(r)

	if err := h.loadParents(n); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrfField := string(csrf.TemplateField(r))

	p := &namespacetemplate.Form{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: h.FormErrors(sess),
			Fields: h.FormFields(sess),
		},
		Namespace: n,
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, err := h.UpdateModel(r, sess)

	if err != nil {
		cause := errors.Cause(err)

		if _, ok := cause.(form.Errors); ok {
			h.RedirectBack(w, r)
			return
		}

		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update namespace"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Namespace changes saved"), "alert")
	h.Redirect(w, r, n.Endpoint())
}

func (h UI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n := h.Model(r)

	alert := template.Success("Namespace has been deleted: "+n.Path)

	if err := h.Delete(r); err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert = template.Danger("Failed to delete namespace")
	}
	sess.AddFlash(alert, "alert")
	h.RedirectBack(w, r)
}

func (h InviteUI) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	ii, err := h.IndexWithRelations(r)

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrfField := string(csrf.TemplateField(r))

	p := &usertemplate.Settings{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Section: &namespacetemplate.InviteIndex{
			CSRF:    csrfField,
			Invites: ii,
		},
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTMLError(w, template.Render(d), http.StatusOK)
}

func (h InviteUI) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if _, err := h.StoreModel(r, sess); err != nil {
		cause := errors.Cause(err)

		if _, ok := cause.(form.Errors); ok {
			h.RedirectBack(w, r)
			return
		}

		switch cause {
		case model.ErrNotFound:
			fallthrough
		case namespace.ErrPermission:
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		log.Error.Println(r.Method, r.URL, errors.Err(err))
		h.RedirectBack(w, r)
	}

	sess.AddFlash(template.Success("Invite sent"), "alert")
	h.RedirectBack(w, r)
}

func (h CollaboratorUI) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, err := h.StoreModel(r)

	alert := template.Success("Your are now a collaborator in:" + n.Name)

	if err != nil {
		log.Error.Println(r.URL, r.Method, errors.Err(err))
		alert = template.Danger("Failed to accept invite")
	}

	sess.AddFlash(alert, "alert")
	h.RedirectBack(w, r)
}

func (h CollaboratorUI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	alert := template.Danger("Collaborator removed: "+mux.Vars(r)["collaborator"])

	if err := h.Delete(r); err != nil {
		log.Error.Println(r.URL, r.Method, errors.Err(err))
		alert = template.Danger("Failed to remove collaborator")
	}

	sess.AddFlash(alert, "alert")
	h.RedirectBack(w, r)
}
