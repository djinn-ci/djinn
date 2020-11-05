package handler

import (
	"net/http"
	"strings"

	"github.com/andrewpillar/djinn/build"
	buildtemplate "github.com/andrewpillar/djinn/build/template"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/image"
	imagetemplate "github.com/andrewpillar/djinn/image/template"
	"github.com/andrewpillar/djinn/key"
	keytemplate "github.com/andrewpillar/djinn/key/template"
	"github.com/andrewpillar/djinn/namespace"
	namespacetemplate "github.com/andrewpillar/djinn/namespace/template"
	"github.com/andrewpillar/djinn/object"
	objecttemplate "github.com/andrewpillar/djinn/object/template"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/variable"
	variabletemplate "github.com/andrewpillar/djinn/variable/template"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

// UI is the handler for handling UI requests made for namespace creation,
// and management.
type UI struct {
	Namespace
}

// InviteUI is the handler for handling UI requests made for sending and
// receiving namespace invites.
type InviteUI struct {
	Invite
}

// CollaboratorUI is the handler for handling UI requests made for managing
// namespace collaborators.
type CollaboratorUI struct {
	Collaborator
}

// Index serves the HTML response detailing the list of namespaces.
func (h Namespace) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	nn, paginator, err := h.IndexWithRelations(namespace.NewStore(h.DB, u), r.URL.Query())

	if err != nil {
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

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Create serves the HTML response for creating namespaces via the web frontend.
func (h Namespace) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	path := r.URL.Query().Get("parent")

	parent, err := namespace.NewStore(h.DB, u).Get(query.Where("path", "=", path))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if parent.Level+1 > namespace.MaxDepth {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if parent.UserID != u.ID && path != "" {
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

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrfField))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Store validates the form submitted in the given request for creating a
// namespace. If validation fails then the user is redirected back to the
// request referer, otherwise they are redirect back to the namespace index.
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

		switch cause {
		case namespace.ErrDepth:
			sess.AddFlash(template.Danger(strings.Title(cause.Error())), "alert")
			h.RedirectBack(w, r)
			return
		case database.ErrNotFound:
			sess.AddFlash(template.Danger("Failed to create namespace: could not find parent"), "alert")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create namespace"), "alert")
			h.RedirectBack(w, r)
			return
		}
	}
	sess.AddFlash(template.Success("Namespace '"+n.Name+"' created"), "alert")
	h.Redirect(w, r, n.Endpoint())
}

// Show serves the HTML response for viewing an individual namespace in the
// given request. This serves different responses based on the base path of the
// request URL.
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
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := h.loadParent(n); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()

	base := web.BasePath(r.URL.Path)
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
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if err := namespace.LoadInviteRelations(h.loaders, ii...); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &namespacetemplate.InviteIndex{
			BasePage:  bp,
			CSRF:      csrf.TemplateField(r),
			Namespace: n,
			Invites:   ii,
		}
	case "collaborators":
		cc, err := namespace.NewCollaboratorStore(h.DB, n).All()

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		mm := make([]database.Model, 0, len(cc))

		for _, c := range cc {
			mm = append(mm, c)
		}

		err = h.Users.Load("id", database.MapKey("user_id", mm), database.Bind("user_id", "id", mm...))

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if err := build.LoadRelations(h.loaders, bb...); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Edit serves the HTML response for editing the namespace in the given request
// context.
func (h UI) Edit(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	n, ok := namespace.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get namespace from request context")
	}

	if err := h.loadParent(n); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Update validates the form submitted in the given request for updating a
// namespace. If validation fails then the user is redirect back to the
// request's referer, otherwise they are redirected back to the updated cron
// job.
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

// Destroy removes the namespace in the given request context from the database.
// This redirects back to the namespace index upon success.
func (h UI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, ok := namespace.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "no namespace in request context")
	}

	alert := template.Success("Namespace has been deleted: " + n.Path)

	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert = template.Danger("Failed to delete namespace")
	}
	sess.AddFlash(alert, "alert")
	h.Redirect(w, r, "/namespaces")
}

// Index serves the HTML response detailing the list of namespace invites.
func (h InviteUI) Index(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, save := h.Session(r)

	u, ok := user.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	invites := namespace.NewInviteStore(h.DB)

	n, ok := namespace.FromContext(ctx)

	// Check if we're serving a response for the invites sent from a namespace
	// or if we're serving a response for the invites sent to the current user.
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

	inviteIndex := &namespacetemplate.InviteIndex{
		Form: template.Form{
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
			Section:   inviteIndex,
		}
	} else {
		p = inviteIndex
	}

	csrfField := string(csrf.TemplateField(r))

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Store validates the form submitted in the given request for sending an
// invite.
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

// Update accepts the invite in the given request context. Upon success this
// will redirect back to the request referer.
func (h InviteUI) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, _, _, err := h.Accept(r)

	alert := template.Success("Your are now a collaborator in: " + n.Name)

	if err != nil {
		if errors.Cause(err) == database.ErrNotFound {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert = template.Danger("Failed to accept invite")
	}

	sess.AddFlash(alert, "alert")
	h.RedirectBack(w, r)
}

// Destroy will delete the invite in the request's context from the database.
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

// Index serves the HTML response detailing the list of namespace collaborators.
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
		Section: &namespacetemplate.CollaboratorIndex{
			BasePage:      bp,
			CSRF:          csrf.TemplateField(r),
			Namespace:     n,
			Collaborators: cc,
		},
	}

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Destroy removes the given namespace collaborator in the request from the
// namespace.
func (h CollaboratorUI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r); err != nil {
		switch err {
		case database.ErrNotFound:
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		case namespace.ErrDeleteSelf:
			sess.AddFlash(template.Danger("You cannot remove yourself"), "alert")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.URL, r.Method, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to remove collaborator"), "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	sess.AddFlash(template.Success("Collaborator remove: "+mux.Vars(r)["collaborator"]), "alert")
	h.RedirectBack(w, r)
}
