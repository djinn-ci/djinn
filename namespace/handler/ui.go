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
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/object"
	objecttemplate "github.com/andrewpillar/thrall/object/template"
	namespacetemplate "github.com/andrewpillar/thrall/namespace/template"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/user"
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
}

type InviteUI struct {
	Invite
}

type CollaboratorUI struct {
	Collaborator
}

func (h Namespace) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	nn, paginator, err := h.IndexWithRelations(namespace.NewStore(h.DB, u), r.URL.Query())

	if err  != nil {
		println(err.Error())
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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

	d := template.NewDashboard(p, r.URL, web.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	parent, err := namespace.NewStore(h.DB, u).Get(query.Where("path", "=", r.URL.Query().Get("parent")))

	if err != nil {
		h.Log.Error.Println(errors.Err(err))
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
			Errors: web.FormErrors(sess),
			Fields: web.FormFields(sess),
		},
	}

	if !parent.IsZero() {
		p.Parent = parent
	}

	d := template.NewDashboard(p, r.URL, web.Alert(sess), string(csrfField))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Namespace) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, f, err := h.StoreModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		if cause == namespace.ErrDepth {
			sess.AddFlash(template.Danger(strings.Title(cause.Error())), "alert")
			h.RedirectBack(w, r)
			return
		}
		h.Log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create namespace"), "alert")
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Success("Namespace '"+n.Name+"' created"), "alert")
	h.Redirect(w, r, n.Endpoint())
}

func (h Namespace) Show(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, save := h.Session(r)

	u, ok := user.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	n, ok := namespace.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get namespace from request context")
	}

	err := h.Users.Load("id", []interface{}{n.UserID}, database.Bind("user_id", "id", n))

	if err != nil {
		h.Log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := h.loadParent(n); err != nil {
		h.Log.Error.Println(errors.Err(err))
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
		nn, paginator, err := h.IndexWithRelations(namespace.NewStore(h.DB, n), q)

		if err != nil {
			h.Log.Error.Println(errors.Err(err))
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
		ii, paginator, err := image.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(errors.Err(err))
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
		oo, paginator, err := object.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(errors.Err(err))
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
		vv, paginator, err := variable.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(errors.Err(err))
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
		kk, paginator, err := key.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(errors.Err(err))
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
	case "invites":
		ii, err := namespace.NewInviteStore(h.DB, n).All()

		if err != nil {
			h.Log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if err := namespace.LoadInviteRelations(h.Loaders, ii...); err != nil {
			h.Log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &namespacetemplate.Invites{
			BasePage:  bp,
			CSRF:      csrf.TemplateField(r),
			Namespace: n,
			Invites:   ii,
		}
	case "collaborators":
		cc, err := namespace.NewCollaboratorStore(h.DB, n).All()

		if err != nil {
			h.Log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		mm := make([]database.Model, 0, len(cc))

		for _, c := range cc {
			mm = append(mm, c)
		}

		err = h.Users.Load("id", database.MapKey("user_id", mm), database.Bind("user_id", "id", mm...))

		if err != nil {
			h.Log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &namespacetemplate.CollaboratorIndex{
			BasePage:      bp,
			CSRF:          csrf.TemplateField(r),
			Namespace:     n,
			Collaborators: cc,
		}
	default:
		bb, paginator, err := build.NewStore(h.DB, n).Index(q)

		if err != nil {
			h.Log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

//		// Namespace already bound to build models, so no need to reload.
//		loaders := h.Loaders.Copy()
//		loaders.Delete("namespace")

		if err := build.LoadRelations(h.Loaders, bb...); err != nil {
			h.Log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		mm := make([]database.Model, 0, len(bb))

		for _, b := range bb {
			if b.NamespaceID.Valid {
				mm = append(mm, b.Namespace)
			}
		}

		err = h.Users.Load("id", database.MapKey("user_id", mm), database.Bind("user_id", "id", mm...))

		if err != nil {
			h.Log.Error.Println(errors.Err(err))
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

	d := template.NewDashboard(p, r.URL, web.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Edit(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	n, ok := namespace.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get namespace from request context")
	}

	if err := h.loadParent(n); err != nil {
		h.Log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrfField := string(csrf.TemplateField(r))

	p := &namespacetemplate.Form{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: web.FormErrors(sess),
			Fields: web.FormFields(sess),
		},
		Namespace: n,
	}
	d := template.NewDashboard(p, r.URL, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, f, err := h.UpdateModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update namespace"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Namespace changes saved"), "alert")
	h.Redirect(w, r, n.Endpoint())
}

func (h UI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, ok := namespace.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "no namespace in request context")
	}

	alert := template.Success("Namespace has been deleted: "+n.Path)

	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert = template.Danger("Failed to delete namespace")
	}
	sess.AddFlash(alert, "alert")
	h.Redirect(w, r, "/namespaces")
}

func (h InviteUI) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, save := h.Session(r)

	u, ok := user.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	invites := namespace.NewInviteStore(h.DB)

	n, ok := namespace.FromContext(ctx)

	if !ok {
		invites.Bind(u)
	} else {
		invites.Bind(n)
	}

	ii, err := h.IndexWithRelations(invites)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	var p template.Dashboard

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}

	section := &namespacetemplate.Invites{
		Form:      template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: web.FormErrors(sess),
			Fields: web.FormFields(sess),
		},
		CSRF:      csrf.TemplateField(r),
		Namespace: n,
		Invites:   ii,
	}

	if n != nil {
		p = &namespacetemplate.Show{
			BasePage:  bp,
			Namespace: n,
			Section:   section,
		}
	} else {
		p = &usertemplate.Settings{
			BasePage: bp,
			Section:  section,
		}
	}

	csrfField := string(csrf.TemplateField(r))

	d := template.NewDashboard(p, r.URL, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h InviteUI) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if _, f, err := h.StoreModel(r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, &f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		if cause == database.ErrNotFound || cause == namespace.ErrPermission {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		h.RedirectBack(w, r)
	}

	sess.AddFlash(template.Success("Invite sent"), "alert")
	h.RedirectBack(w, r)
}

func (h InviteUI) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, _, _, err := h.Accept(r)

	alert := template.Success("Your are now a collaborator in: " + n.Name)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert = template.Danger("Failed to accept invite")
	}

	sess.AddFlash(alert, "alert")
	h.RedirectBack(w, r)
}

func (h InviteUI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	alert := template.Success("Invite rejected")

	if err := h.DeleteModel(r); err != nil {
		if err == database.ErrNotFound {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert = template.Danger("Failed to reject invite")
	}

	sess.AddFlash(alert, "alert")
	h.RedirectBack(w, r)
}

func (h CollaboratorUI) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, save := h.Session(r)

	u, ok := user.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	n, ok := namespace.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get namespace from request context")
	}

	cc, err := namespace.NewCollaboratorStore(h.DB, n).All()

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	mm := database.ModelSlice(len(cc), namespace.CollaboratorModel(cc))

	err = h.Users.Load("id", database.MapKey("user_id", mm), database.Bind("user_id", "id", mm...))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}
	p := &namespacetemplate.Show{
		BasePage:  bp,
		Namespace: n,
		Section:   &namespacetemplate.CollaboratorIndex{
			BasePage:      bp,
			CSRF:          csrf.TemplateField(r),
			Namespace:     n,
			Collaborators: cc,
		},
	}

	d := template.NewDashboard(p, r.URL, web.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h CollaboratorUI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	alert := template.Success("Collaborator removed: "+mux.Vars(r)["collaborator"])

	if err := h.DeleteModel(r); err != nil {
		if err == database.ErrNotFound {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		h.Log.Error.Println(r.URL, r.Method, errors.Err(err))
		alert = template.Danger("Failed to remove collaborator")
	}

	sess.AddFlash(alert, "alert")
	h.RedirectBack(w, r)
}
