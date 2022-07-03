package http

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"djinn-ci.com/alert"
	"djinn-ci.com/build"
	buildtemplate "djinn-ci.com/build/template"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	imagetemplate "djinn-ci.com/image/template"
	keytemplate "djinn-ci.com/key/template"
	"djinn-ci.com/namespace"
	namespacetemplate "djinn-ci.com/namespace/template"
	objecttemplate "djinn-ci.com/object/template"
	"djinn-ci.com/runner"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"
	"djinn-ci.com/variable"
	variabletemplate "djinn-ci.com/variable/template"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type UI struct {
	*Handler
}

func (h UI) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	nn, paginator, err := h.IndexWithRelations(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
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

	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf.TemplateField(r))
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Create(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	path := r.URL.Query().Get("parent")

	parent, ok, err := h.Namespaces.Get(
		query.Where("path", "=", query.Arg(path)),
		namespace.SharedWith(u.ID),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if ok {
		if parent.Level+1 > namespace.MaxDepth {
			h.NotFound(w, r)
			return
		}

		if parent.UserID != u.ID && path != "" {
			h.NotFound(w, r)
			return
		}

		if err := h.Users.Load("user_id", "id", parent); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}
	}

	csrf := csrf.TemplateField(r)

	p := &namespacetemplate.Form{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
	}

	if ok {
		p.Parent = parent
	}

	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, f, err := h.StoreModel(u, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			if errs, ok := verrs["fatal"]; ok {
				h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
				alert.Flash(sess, alert.Danger, "Failed to create namespace")
				h.RedirectBack(w, r)
				return
			}

			webutil.FlashFormWithErrors(sess, f, verrs)
			h.RedirectBack(w, r)
			return
		}

		switch cause {
		case namespace.ErrDepth:
			alert.Flash(sess, alert.Danger, strings.Title(cause.Error()))
			h.RedirectBack(w, r)
		case database.ErrNotFound:
			alert.Flash(sess, alert.Danger, "Failed to create namespace: could not find parent")
			h.RedirectBack(w, r)
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to create namespace")
			h.RedirectBack(w, r)
		}
		return
	}

	alert.Flash(sess, alert.Success, "Namespace created: "+n.Name)
	h.Redirect(w, r, n.Endpoint())
}

func (h UI) Show(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	q := r.URL.Query()

	base := webutil.BasePath(r.URL.Path)
	csrf := csrf.TemplateField(r)

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}
	p := &namespacetemplate.Show{
		BasePage:       bp,
		Namespace:      n,
		IsCollaborator: n.IsCollaborator(h.DB, u.ID) == nil,
	}

	switch base {
	case "namespaces":
		nn, paginator, err := h.Namespaces.Index(r.URL.Query(), query.Where("parent_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if err := h.loadLastBuild(nn); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
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
		ii, paginator, err := h.Images.Index(q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		for _, i := range ii {
			i.Namespace = n
		}

		p.Section = &imagetemplate.Index{
			BasePage:  bp,
			CSRF:      csrf,
			Paginator: paginator,
			Images:    ii,
			Search:    q.Get("search"),
		}
	case "objects":
		oo, paginator, err := h.Objects.Index(q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		for _, o := range oo {
			o.Namespace = n
		}

		p.Section = &objecttemplate.Index{
			BasePage:  bp,
			CSRF:      csrf,
			Paginator: paginator,
			Objects:   oo,
			Search:    q.Get("search"),
		}
	case "variables":
		vv, paginator, err := h.Variables.Index(q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		unmasked := variable.GetUnmasked(sess.Values)

		for _, v := range vv {
			v.Namespace = n

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

		p.Section = &variabletemplate.Index{
			BasePage:  bp,
			CSRF:      csrf,
			Paginator: paginator,
			Variables: vv,
			Unmasked:  unmasked,
			Search:    q.Get("search"),
		}
	case "keys":
		kk, paginator, err := h.Keys.Index(q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		for _, k := range kk {
			k.Namespace = n
		}

		p.Section = &keytemplate.Index{
			BasePage:  bp,
			CSRF:      csrf,
			Paginator: paginator,
			Keys:      kk,
			Search:    q.Get("search"),
		}
	case "invites":
		ii, err := h.Invites.All(query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		mm := make([]database.Model, 0, len(ii))

		for _, i := range ii {
			i.Namespace = n

			mm = append(mm, i)
		}

		relations := []database.RelationFunc{
			database.Relation("invitee_id", "id", h.Users),
			database.Relation("inviter_id", "id", h.Users),
		}

		if err := database.LoadRelations(mm, relations...); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		p.Section = &namespacetemplate.InviteIndex{
			BasePage: bp,
			Form: template.Form{
				CSRF:   csrf,
				Errors: webutil.FormErrors(sess),
				Fields: webutil.FormFields(sess),
			},
			CSRF:      csrf,
			Namespace: n,
			Invites:   ii,
		}
	case "collaborators":
		cc, err := h.Collaborators.All(query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		mm := make([]database.Model, 0, len(cc))

		for _, c := range cc {
			c.Namespace = n
			mm = append(mm, c)
		}

		if err := h.Users.Load("user_id", "id", mm...); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		p.Section = &namespacetemplate.CollaboratorIndex{
			BasePage:      bp,
			CSRF:          csrf,
			Namespace:     n,
			Collaborators: cc,
		}
	case "webhooks":
		ww, err := h.Webhooks.All(query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		for _, w := range ww {
			w.Namespace = n
		}

		if err := h.Webhooks.LoadLastDeliveries(ww...); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		p.Section = &namespacetemplate.WebhookIndex{
			BasePage:  bp,
			CSRF:      csrf,
			Namespace: n,
			Webhooks:  ww,
		}
	default:
		bb, paginator, err := h.Builds.Index(q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		if err := build.LoadRelations(h.DB, bb...); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		for _, b := range bb {
			b.Namespace = n
			b.Namespace.User = n.User
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

	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) DestroyCollab(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.deleteCollab(u, n, r); err != nil {
		if errors.Is(err, namespace.ErrPermission) {
			alert.Flash(sess, alert.Danger, "Failed to remove collaborator")
			h.RedirectBack(w, r)
			return
		}

		if errors.Is(err, database.ErrNotFound) {
			alert.Flash(sess, alert.Danger, "No such collaborator")
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to remove collaborator")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Collaborator removed")
	h.RedirectBack(w, r)
}

func (h UI) Badge(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	owner, ok, err := h.Users.Get(query.Where("username", "=", query.Arg(vars["username"])))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	path := strings.TrimSuffix(vars["namespace"], "/")

	n, ok, err := h.Namespaces.Get(
		query.Where("user_id", "=", query.Arg(owner.ID)),
		query.Where("path", "=", query.Arg(path)),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, badgeUnknown)
		return
	}

	if n.Visibility == namespace.Internal || n.Visibility == namespace.Private {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, badgeUnknown)
		return
	}

	b, ok, err := h.Builds.Get(
		query.Where("namespace_id", "=", query.Arg(n.ID)),
		query.OrderDesc("created_at"),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, badgeUnknown)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.WriteHeader(http.StatusOK)

	switch b.Status {
	case runner.Queued:
		io.WriteString(w, badgeQueued)
	case runner.Running:
		io.WriteString(w, badgeRunning)
	case runner.Passed:
		io.WriteString(w, badgePassed)
	case runner.PassedWithFailures:
		io.WriteString(w, badgePassedWithFailures)
	case runner.Failed:
		io.WriteString(w, badgeFailed)
	case runner.Killed:
		io.WriteString(w, badgeKilled)
	case runner.TimedOut:
		io.WriteString(w, badgeTimedOut)
	default:
		io.WriteString(w, badgeUnknown)
	}
}

func (h UI) Edit(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrf := csrf.TemplateField(r)

	p := &namespacetemplate.Form{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		Namespace: n,
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Update(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, f, err := h.UpdateModel(n, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			if errs, ok := verrs["fatal"]; ok {
				h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
				alert.Flash(sess, alert.Danger, "Failed to update namespace")
				h.RedirectBack(w, r)
				return
			}

			webutil.FlashFormWithErrors(sess, f, verrs)
			h.RedirectBack(w, r)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	alert.Flash(sess, alert.Success, "Namespace changes saved")
	h.Redirect(w, r, n.Endpoint())
}

func (h UI) Destroy(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r.Context(), n); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to delete namespace")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Namespace has been deleted: "+n.Path)
	h.Redirect(w, r, "/namespaces")
}

type InviteUI struct {
	*InviteHandler
}

func (h InviteUI) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	ii, err := h.invitesWithRelations(u)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	csrf := csrf.TemplateField(r)

	p := &namespacetemplate.InviteIndex{
		CSRF:    csrf,
		Invites: ii,
	}

	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h InviteUI) Store(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if _, f, err := h.StoreModel(n, r); err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			webutil.FlashFormWithErrors(sess, f, verrs)
			h.RedirectBack(w, r)
			return
		}

		if errors.Is(cause, database.ErrNotFound) || errors.Is(cause, namespace.ErrPermission) {
			h.NotFound(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to send invite")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Invite sent")
	h.RedirectBack(w, r)
}

func (h InviteUI) Update(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	id, _ := strconv.ParseInt(mux.Vars(r)["invite"], 10, 64)

	i, ok, err := h.Invites.Get(query.Where("id", "=", query.Arg(id)))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to accept invite")
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	n, _, _, err := h.Accept(r.Context(), u, i)

	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.NotFound(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to accept invite")
		return
	}

	alert.Flash(sess, alert.Success, "You are now a collaborator in: "+n.Name)
	h.RedirectBack(w, r)
}

func (h InviteUI) Destroy(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	i, ok, err := h.inviteFromRequest(u, r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to accept invite")
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if err := h.DeleteModel(r.Context(), u, i); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.NotFound(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to reject invite")
		return
	}

	alert.Flash(sess, alert.Success, "Invite rejected")
	h.RedirectBack(w, r)
}

type WebhookUI struct {
	*WebhookHandler
}

func (h WebhookUI) Create(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrf := csrf.TemplateField(r)

	p := &namespacetemplate.WebhookForm{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		Namespace: n,
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h WebhookUI) Store(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	_, f, err := h.StoreModel(u, n, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			webutil.FlashFormWithErrors(sess, f, verrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to create webhook")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Webhook created")
	h.Redirect(w, r, n.Endpoint("webhooks"))
}

func (h WebhookUI) Show(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	wh, ok, err := h.webhookFromRequest(u, n, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	dd, err := h.Webhooks.Deliveries(wh.ID)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	for _, d := range dd {
		d.Webhook = wh
	}

	csrf := csrf.TemplateField(r)

	p := &namespacetemplate.WebhookForm{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		Namespace:  n,
		Webhook:    wh,
		Deliveries: dd,
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h WebhookUI) Delivery(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	wh, ok, err := h.webhookFromRequest(u, n, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["delivery"], 10, 64)

	del, ok, err := h.Webhooks.Delivery(wh.ID, id)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	del.Webhook = wh

	csrf := csrf.TemplateField(r)

	p := &namespacetemplate.ShowDelivery{
		BasePage: template.BasePage{
			User: u,
			URL:  r.URL,
		},
		CSRF:     csrf,
		Delivery: del,
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)

	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h WebhookUI) Redeliver(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	wh, ok, err := h.webhookFromRequest(u, n, r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to redeliver hook")
		h.RedirectBack(w, r)
		return
	}

	if !ok {
		alert.Flash(sess, alert.Danger, "No such hook")
		h.RedirectBack(w, r)
		return
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["delivery"], 10, 64)

	del, ok, err := h.Webhooks.Delivery(wh.ID, id)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to redeliver hook")
		h.RedirectBack(w, r)
		return
	}

	if !ok {
		alert.Flash(sess, alert.Danger, "No such hook")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Webhooks.Redeliver(wh.ID, del.EventID); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to redeliver hook")
		h.RedirectBack(w, r)
		return
	}

	latest, err := h.Webhooks.LastDelivery(wh.ID)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to redeliver hook")
		h.RedirectBack(w, r)
		return
	}

	latest.Webhook = wh

	alert.Flash(sess, alert.Success, "Webhook delivery sent")
	h.Redirect(w, r, latest.Endpoint())
}

func (h WebhookUI) Update(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	wh, ok, err := h.webhookFromRequest(u, n, r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to update hook")
		h.RedirectBack(w, r)
		return
	}

	if !ok {
		alert.Flash(sess, alert.Danger, "No such hook")
		h.RedirectBack(w, r)
		return
	}

	wh, f, err := h.UpdateModel(wh, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			webutil.FlashFormWithErrors(sess, f, verrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to update webhook")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Webhook has been updated: "+wh.PayloadURL.String())
	h.Redirect(w, r, wh.Endpoint())
}

func (h WebhookUI) Destroy(u *user.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	wh, ok, err := h.webhookFromRequest(u, n, r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to delete hook")
		h.RedirectBack(w, r)
		return
	}

	if !ok {
		alert.Flash(sess, alert.Danger, "No such hook")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Webhooks.Delete(wh.ID); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to delete hook")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Webhook has been deleted")
	h.Redirect(w, r, wh.Namespace.Endpoint("webhooks"))
}

func RegisterUI(srv *server.Server) {
	user := userhttp.NewHandler(srv)

	ui := UI{
		Handler: NewHandler(srv),
	}

	inviteui := InviteUI{
		InviteHandler: NewInviteHandler(srv),
	}

	hookui := WebhookUI{
		WebhookHandler: NewWebhookHandler(srv),
	}

	auth := srv.Router.PathPrefix("/").Subrouter()
	auth.HandleFunc("/namespaces", user.WithUser(ui.Index)).Methods("GET")
	auth.HandleFunc("/namespaces/create", user.WithUser(ui.Create)).Methods("GET")
	auth.HandleFunc("/namespaces", user.WithUser(ui.Store)).Methods("POST")
	auth.HandleFunc("/invites", user.WithUser(inviteui.Index)).Methods("GET")
	auth.HandleFunc("/invites/{invite:[0-9]+}", user.WithUser(inviteui.Update)).Methods("PATCH")
	auth.HandleFunc("/invites/{invite:[0-9]+}", user.WithUser(inviteui.Destroy)).Methods("DELETE")
	auth.Use(srv.CSRF)

	sr := srv.Router.PathPrefix("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}").Subrouter()
	sr.HandleFunc("", user.WithOptionalUser(ui.WithNamespace(ui.Show))).Methods("GET")
	sr.HandleFunc("/-/badge.svg", ui.Badge).Methods("GET")
	sr.HandleFunc("/-/edit", user.WithUser(ui.WithNamespace(ui.Edit))).Methods("GET")
	sr.HandleFunc("/-/namespaces", user.WithOptionalUser(ui.WithNamespace(ui.Show))).Methods("GET")
	sr.HandleFunc("/-/images", user.WithOptionalUser(ui.WithNamespace(ui.Show))).Methods("GET")
	sr.HandleFunc("/-/objects", user.WithOptionalUser(ui.WithNamespace(ui.Show))).Methods("GET")
	sr.HandleFunc("/-/variables", user.WithOptionalUser(ui.WithNamespace(ui.Show))).Methods("GET")
	sr.HandleFunc("/-/keys", user.WithOptionalUser(ui.WithNamespace(ui.Show))).Methods("GET")
	sr.HandleFunc("/-/invites", user.WithUser(ui.WithNamespace(ui.Show))).Methods("GET")
	sr.HandleFunc("/-/invites", user.WithUser(ui.WithNamespace(inviteui.Store))).Methods("POST")
	sr.HandleFunc("/-/collaborators", user.WithUser(ui.WithNamespace(ui.Show))).Methods("GET")
	sr.HandleFunc("/-/collaborators/{collaborator}", user.WithUser(ui.WithNamespace(ui.DestroyCollab))).Methods("DELETE")
	sr.HandleFunc("/-/webhooks", user.WithUser(ui.WithNamespace(ui.Show))).Methods("GET")
	sr.HandleFunc("/-/webhooks/create", user.WithUser(ui.WithNamespace(hookui.Create))).Methods("GET")
	sr.HandleFunc("/-/webhooks", user.WithUser(ui.WithNamespace(hookui.Store))).Methods("POST")
	sr.HandleFunc("/-/webhooks/{webhook:[0-9]+}", user.WithUser(ui.WithNamespace(hookui.Show))).Methods("GET")
	sr.HandleFunc("/-/webhooks/{webhook:[0-9]+}/deliveries/{delivery:[0-9]+}", user.WithUser(ui.WithNamespace(hookui.Delivery))).Methods("GET")
	sr.HandleFunc("/-/webhooks/{webhook:[0-9]+}/deliveries/{delivery:[0-9]+}", user.WithUser(ui.WithNamespace(hookui.Redeliver))).Methods("PATCH")
	sr.HandleFunc("/-/webhooks/{webhook:[0-9]+}", user.WithUser(ui.WithNamespace(hookui.Update))).Methods("PATCH")
	sr.HandleFunc("/-/webhooks/{webhook:[0-9]+}", user.WithUser(ui.WithNamespace(hookui.Destroy))).Methods("DELETE")
	sr.HandleFunc("", user.WithUser(ui.WithNamespace(ui.Update))).Methods("PATCH")
	sr.HandleFunc("", user.WithUser(ui.WithNamespace(ui.Destroy))).Methods("DELETE")
	sr.Use(srv.CSRF)
}
