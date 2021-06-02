package handler

import (
	"net/http"
	"strings"

	"djinn-ci.com/build"
	buildtemplate "djinn-ci.com/build/template"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	imagetemplate "djinn-ci.com/image/template"
	"djinn-ci.com/key"
	keytemplate "djinn-ci.com/key/template"
	"djinn-ci.com/namespace"
	namespacetemplate "djinn-ci.com/namespace/template"
	"djinn-ci.com/object"
	objecttemplate "djinn-ci.com/object/template"
	"djinn-ci.com/template"
	"djinn-ci.com/user"
	"djinn-ci.com/variable"
	variabletemplate "djinn-ci.com/variable/template"
	"djinn-ci.com/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

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

type WebhookUI struct {
	Webhook
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

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf.TemplateField(r))
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Create serves the HTML response for creating namespaces via the web frontend.
func (h Namespace) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	path := r.URL.Query().Get("parent")

	parent, err := namespace.NewStore(h.DB, u).Get(query.Where("path", "=", query.Arg(path)))

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

	csrf := csrf.TemplateField(r)

	p := &namespacetemplate.Form{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
	}

	if !parent.IsZero() {
		p.Parent = parent
	}

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Store validates the form submitted in the given request for creating a
// namespace. If validation fails then the user is redirected back to the
// request referer, otherwise they are redirect back to the namespace index.
func (h Namespace) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, f, err := h.StoreModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		switch cause {
		case namespace.ErrDepth:
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: strings.Title(cause.Error()),
			}, "alert")
			h.RedirectBack(w, r)
			return
		case database.ErrNotFound:
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to create namespace: could not find parent",
			}, "alert")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to create namespace",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Namespace created: " + n.Name,
	}, "alert")
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

	base := webutil.BasePath(r.URL.Path)
	csrf := csrf.TemplateField(r)

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
			CSRF:      csrf,
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
			CSRF:      csrf,
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
			CSRF:      csrf,
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
			CSRF:      csrf,
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
			CSRF:      csrf,
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
			CSRF:          csrf,
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

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
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

	csrf := csrf.TemplateField(r)

	p := &namespacetemplate.Form{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		Namespace: n,
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
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

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to update namespace",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Namespace changes saved",
	}, "alert")
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

	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Success,
			Close:   true,
			Message: "Failed to delete namespace",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Namespace has been deleted: " + n.Path,
	}, "alert")
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

	csrf := csrf.TemplateField(r)

	inviteIndex := &namespacetemplate.InviteIndex{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		CSRF:      csrf,
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

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Store validates the form submitted in the given request for sending an
// invite.
func (h InviteUI) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if _, f, err := h.StoreModel(r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, &f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		if cause == database.ErrNotFound || cause == namespace.ErrPermission {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to send invite",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Invite sent",
	}, "alert")
	h.RedirectBack(w, r)
}

// Update accepts the invite in the given request context. Upon success this
// will redirect back to the request referer.
func (h InviteUI) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, _, _, err := h.Accept(r)

	if err != nil {
		if errors.Cause(err) == database.ErrNotFound {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))

		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to accept invite",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "You are now a collaborator in: " + n.Name,
	}, "alert")
	h.RedirectBack(w, r)
}

// Destroy will delete the invite in the request's context from the database.
func (h InviteUI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r); err != nil {
		if err == database.ErrNotFound {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))

		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to reject invite",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Invite rejected",
	}, "alert")
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

	csrf := csrf.TemplateField(r)

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}
	p := &namespacetemplate.Show{
		BasePage:  bp,
		Namespace: n,
		Section: &namespacetemplate.CollaboratorIndex{
			BasePage:      bp,
			CSRF:          csrf,
			Namespace:     n,
			Collaborators: cc,
		},
	}

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
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
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "You cannot remove yourself",
			}, "alert")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.URL, r.Method, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to remove collaborator",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Collaborator remove: " + mux.Vars(r)["collaborator"],
	}, "alert")
	h.RedirectBack(w, r)
}

func (h WebhookUI) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	ctx := r.Context()

	n, ok := namespace.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "no namespace in request")
	}

	u, ok := user.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "no user in request")
	}

	webhooks := namespace.NewWebhookStore(h.DB, n, u)

	ww, err := webhooks.All()

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	for _, w := range ww {
		w.Namespace = n
	}

	if err := webhooks.LoadLastDeliveries(ww...); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrf := csrf.TemplateField(r)

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}

	p := &namespacetemplate.Show{
		BasePage:  bp,
		Namespace: n,
		Section: &namespacetemplate.WebhookIndex{
			BasePage:  bp,
			CSRF:      csrf,
			Namespace: n,
			Webhooks:  ww,
		},
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h WebhookUI) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	ctx := r.Context()

	n, ok := namespace.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "no namespace in request")
	}

	u, ok := user.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "no user in request")
	}

	csrf := csrf.TemplateField(r)

	p := &namespacetemplate.WebhookForm{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		Namespace: n,
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h WebhookUI) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, ok := namespace.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "no namespace in request context")
	}

	_, f, err := h.StoreModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to create webhook",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Webhook created",
	}, "alert")
	h.Redirect(w, r, n.Endpoint("webhooks"))
}

func (h WebhookUI) Show(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	ctx := r.Context()

	n, ok := namespace.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "no namespace in request context")
	}

	wh, ok := namespace.WebhookFromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "no webhook in request context")
	}

	u, ok := user.FromContext(ctx)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "no user in request context")
	}

	csrf := csrf.TemplateField(r)

	p := &namespacetemplate.WebhookForm{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		Namespace: n,
		Webhook:   wh,
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h WebhookUI) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	wh, f, err := h.UpdateModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to update webhook",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Webhook has been updated: " + wh.PayloadURL.String(),
	}, "alert")
	h.Redirect(w, r, wh.Namespace.Endpoint("webhooks"))
}

func (h WebhookUI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	wh, ok := namespace.WebhookFromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "no webhook in request context")
	}

	if err := namespace.NewWebhookStore(h.DB).Delete(wh.ID); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Webhook has been deleted",
	}, "alert")
	h.Redirect(w, r, wh.Namespace.Endpoint("webhooks"))
}
