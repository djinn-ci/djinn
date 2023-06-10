package http

import (
	"net/http"
	"strconv"

	"djinn-ci.com/alert"
	"djinn-ci.com/auth"
	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/key"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/template/form"
	"djinn-ci.com/user"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/gorilla/mux"
)

type UI struct {
	*Handler
}

func (h UI) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	p, err := h.Handler.Index(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get namespaces"))
		return
	}

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.NamespaceIndex{
		Paginator:  template.NewPaginator[*namespace.Namespace](tmpl.Page, p),
		Namespaces: p.Items,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Create(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	ctx := r.Context()
	path := r.URL.Query().Get("parent")

	parent, ok, err := h.Namespaces.Get(
		ctx,
		query.Where("path", "=", query.Arg(path)),
		namespace.SharedWith(u.ID),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get namespace parent"))
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

		if err := user.Loader(h.DB).Load(ctx, "user_id", "id", parent); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to load namespace user"))
			return
		}
	}

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.NamespaceForm{
		Form:   form.New(sess, r),
		Parent: parent,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, f, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, f, errors.Wrap(err, "Failed to create namespace"))
		return
	}

	alert.Flash(sess, alert.Success, "Namespace created: "+n.Name)
	h.Redirect(w, r, n.Endpoint())
}

func (h UI) Show(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	ctx := r.Context()
	q := r.URL.Query()

	tmpl := template.NewDashboard(u, sess, r)
	show := template.NamespaceShow{
		Page:      tmpl.Page,
		Namespace: n,
	}

	switch webutil.BasePath(r.URL.Path) {
	case "namespaces":
		p, err := h.Namespaces.Index(ctx, q, query.Where("parent_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get namespaces"))
			return
		}

		if err := h.loadLastBuild(ctx, p.Items); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		show.Partial = &template.NamespaceIndex{
			Paginator:  template.NewPaginator[*namespace.Namespace](tmpl.Page, p),
			Namespaces: p.Items,
		}
	case "images":
		p, err := h.Images.Index(ctx, q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get images"))
			return
		}

		for _, i := range p.Items {
			i.Namespace = n
		}

		show.Partial = &template.ImageIndex{
			Paginator: template.NewPaginator[*image.Image](tmpl.Page, p),
			Images:    p.Items,
		}
	case "objects":
		p, err := h.Objects.Index(ctx, q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get objects"))
			return
		}

		for _, o := range p.Items {
			o.Namespace = n
		}

		show.Partial = &template.ObjectIndex{
			Paginator: template.NewPaginator[*object.Object](tmpl.Page, p),
			Objects:   p.Items,
		}
	case "variables":
		p, err := h.Variables.Index(ctx, q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get variables"))
			return
		}

		unmasked := variable.GetUnmasked(sess.Values)

		for _, v := range p.Items {
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

		show.Partial = &template.VariableIndex{
			Paginator: template.NewPaginator[*variable.Variable](tmpl.Page, p),
			Variables: p.Items,
		}
	case "keys":
		p, err := h.Keys.Index(ctx, q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get keys"))
			return
		}

		for _, k := range p.Items {
			k.Namespace = n
		}

		show.Partial = &template.KeyIndex{
			Paginator: template.NewPaginator[*key.Key](tmpl.Page, p),
			Keys:      p.Items,
		}
	case "invites":
		ii, err := h.Invites.All(ctx, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get invites"))
			return
		}

		mm := make([]database.Model, 0, len(ii))

		for _, i := range ii {
			i.Namespace = n
			mm = append(mm, i)
		}

		ld := user.Loader(h.DB)

		if err := ld.Load(ctx, "invitee_id", "id", mm...); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to loader users"))
			return
		}

		if err := ld.Load(ctx, "inviter_id", "id", mm...); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to loader users"))
			return
		}

		show.Partial = &template.InviteIndex{
			Form:      form.New(sess, r),
			Namespace: n,
			Invites:   ii,
		}
	case "collaborators":
		cc, err := h.Collaborators.All(ctx, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get collaborators"))
			return
		}

		mm := make([]database.Model, 0, len(cc))

		for _, c := range cc {
			c.Namespace = n
			mm = append(mm, c)
		}

		if err := user.Loader(h.DB).Load(ctx, "user_id", "id", mm...); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to load users"))
			return
		}

		show.Partial = &template.CollaboratorIndex{
			Page:          tmpl.Page,
			Namespace:     n,
			Collaborators: cc,
		}
	case "webhooks":
		ww, err := h.Webhooks.All(ctx, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get webhooks"))
			return
		}

		mm := make([]database.Model, 0, len(ww))

		for _, w := range ww {
			w.Namespace = n
			mm = append(mm, w)
		}

		if err := user.Loader(h.DB).Load(ctx, "author_id", "id", mm...); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to loader users"))
			return
		}

		if err := h.Webhooks.LoadLastDeliveries(ctx, ww...); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get deliveries"))
			return
		}

		show.Partial = &template.WebhookIndex{
			Page:      tmpl.Page,
			Namespace: n,
			Webhooks:  ww,
		}
	default:
		p, err := h.Builds.Index(ctx, q, query.Where("namespace_id", "=", query.Arg(n.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to get builds"))
			return
		}

		if err := build.LoadRelations(ctx, h.DB, p.Items...); err != nil {
			h.InternalServerError(w, r, errors.Wrap(err, "Failed to load build relations"))
			return
		}

		for _, b := range p.Items {
			b.Namespace = n
			b.Namespace.User = n.User
		}

		show.Partial = &template.BuildIndex{
			Paginator: template.NewPaginator[*build.Build](tmpl.Page, p),
			Builds:    p.Items,
		}
	}

	tmpl.Partial = &show
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) DestroyCollaborator(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.Handler.DestroyCollaborator(u, n, r); err != nil {
		if errors.Is(err, database.ErrPermission) {
			h.InternalServerError(w, r, errors.Benign("Failed to remove collaborator"))
			return
		}

		if errors.Is(err, database.ErrNoRows) {
			h.InternalServerError(w, r, errors.Benign("No such collaborator"))
			return
		}

		h.InternalServerError(w, r, errors.Wrap(err, "Failed to remove collaborator"))
		return
	}

	alert.Flash(sess, alert.Success, "Collaborator removed")
	h.RedirectBack(w, r)
}

func (h UI) Edit(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.NamespaceForm{
		Form:      form.New(sess, r),
		Namespace: n,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Update(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	n, f, err := h.Handler.Update(u, n, r)

	if err != nil {
		h.FormError(w, r, f, errors.Wrap(err, "Failed to update namespace"))
		return
	}

	alert.Flash(sess, alert.Success, "Namespace changes saved")
	h.Redirect(w, r, n.Endpoint())
}

func (h UI) Destroy(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.Handler.Destroy(r.Context(), n); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to delete namespace"))
		return
	}

	alert.Flash(sess, alert.Success, "Namespace has been deleted: "+n.Path)
	h.Redirect(w, r, "/namespaces")
}

type InviteUI struct {
	*InviteHandler
}

func (h InviteUI) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	ii, err := h.Invites.All(r.Context(), query.Where("invitee_id", "=", query.Arg(u.ID)))

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get invites"))
		return
	}

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.InviteIndex{
		Invites: ii,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h InviteUI) Store(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if _, f, err := h.InviteHandler.Store(u, n, r); err != nil {
		errtab := map[error]string{
			database.ErrNoRows:        "No such user",
			namespace.ErrSelfInvite:   "Cannot invite self",
			namespace.ErrInviteSent:   "Invite already sent",
			namespace.ErrCollaborator: "Already a collaborator",
		}

		if err, ok := errtab[err]; ok {
			alert.Flash(sess, alert.Danger, err)
			h.RedirectBack(w, r)
			return
		}

		h.FormError(w, r, f, errors.Wrap(err, "Failed to send invite"))
		return
	}

	alert.Flash(sess, alert.Success, "Invite sent")
	h.RedirectBack(w, r)
}

func (h InviteUI) Update(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	ctx := r.Context()
	id := mux.Vars(r)["invite"]

	i, ok, err := h.Invites.Get(ctx, query.Where("id", "=", query.Arg(id)))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to accept invite")
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	n, _, _, err := h.Accept(ctx, u, i)

	if err != nil {
		if errors.Is(err, database.ErrNoRows) {
			h.NotFound(w, r)
			return
		}

		h.InternalServerError(w, r, errors.Wrap(err, "Failed to accept invite"))
		return
	}

	alert.Flash(sess, alert.Success, "You are now a collaborator in: "+n.Name)
	h.RedirectBack(w, r)
}

func (h InviteUI) Destroy(u *auth.User, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	i, ok, err := h.Invites.Get(ctx, query.Where("id", "=", query.Arg(mux.Vars(r)["invite"])))

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to accept invite"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if err := h.InviteHandler.Destroy(ctx, u, i); err != nil {
		if errors.Is(err, database.ErrNoRows) {
			h.NotFound(w, r)
			return
		}

		h.InternalServerError(w, r, errors.Wrap(err, "Failed to reject invite"))
		return
	}

	alert.Flash(sess, alert.Success, "Invite rejected")
	h.RedirectBack(w, r)
}

type WebhookUI struct {
	*WebhookHandler
}

func (h WebhookUI) Create(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.WebhookForm{
		Form:      form.New(sess, r),
		Namespace: n,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h WebhookUI) Store(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	_, f, err := h.WebhookHandler.Store(u, n, r)

	if err != nil {
		h.FormError(w, r, f, errors.Wrap(err, "Failed to create webhook"))
		return
	}

	alert.Flash(sess, alert.Success, "Webhook created")
	h.Redirect(w, r, n.Endpoint("webhooks"))
}

func (h WebhookUI) Show(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	wh, ok, err := h.Webhooks.Get(
		ctx,
		query.Where("id", "=", query.Arg(mux.Vars(r)["webhook"])),
		query.Where("namespace_id", "=", query.Arg(n.ID)),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get webhook"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	wh.Namespace = n

	dd, err := h.Webhooks.Deliveries(ctx, wh)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get webhook deliveries"))
		return
	}

	for _, d := range dd {
		d.Webhook = wh
	}

	f := form.New(sess, r)
	f.Fields["payload_url"] = wh.PayloadURL.String()

	if wh.SSL {
		f.Fields["ssl"] = "checked"
	}
	if wh.Active {
		f.Fields["active"] = "checked"
	}

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.WebhookForm{
		Form:       f,
		Namespace:  n,
		Webhook:    wh,
		Deliveries: dd,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h WebhookUI) Delivery(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	wh, ok, err := h.Webhooks.Get(
		ctx,
		query.Where("id", "=", query.Arg(mux.Vars(r)["webhook"])),
		query.Where("namespace_id", "=", query.Arg(n.ID)),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get webhook"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["delivery"], 10, 64)

	del, ok, err := h.Webhooks.Delivery(ctx, wh, id)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to get webhook delivery"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	del.Webhook = wh

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.WebhookDelivery{
		Page:     tmpl.Page,
		Delivery: del,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h WebhookUI) Redeliver(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	wh, ok, err := h.Webhooks.Get(
		ctx,
		query.Where("id", "=", query.Arg(mux.Vars(r)["webhook"])),
		query.Where("namespace_id", "=", query.Arg(n.ID)),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to redeliver webhook"))
		return
	}

	if !ok {
		h.InternalServerError(w, r, errors.Benign("No such webhook"))
		return
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["delivery"], 10, 64)

	del, ok, err := h.Webhooks.Delivery(ctx, wh, id)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to redeliver webhook"))
		return
	}

	if !ok {
		h.InternalServerError(w, r, errors.Benign("No such webhook"))
		return
	}

	if err := h.Webhooks.Redeliver(ctx, wh, del.EventID); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to redeliver webhook"))
		return
	}

	latest, err := h.Webhooks.LastDelivery(ctx, wh)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to redeliver webhook"))
		return
	}

	latest.Webhook = wh

	alert.Flash(sess, alert.Success, "Webhook delivery sent")
	h.Redirect(w, r, latest.Endpoint())
}

func (h WebhookUI) Update(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	wh, ok, err := h.Webhooks.Get(
		ctx,
		query.Where("id", "=", query.Arg(mux.Vars(r)["webhook"])),
		query.Where("namespace_id", "=", query.Arg(n.ID)),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to update webhook"))
		return
	}

	if !ok {
		h.InternalServerError(w, r, errors.Benign("No such webhook"))
		return
	}

	wh.Namespace = n

	wh, f, err := h.WebhookHandler.Update(wh, r)

	if err != nil {
		h.FormError(w, r, f, errors.Wrap(err, "Failed to update webhook"))
		return
	}

	alert.Flash(sess, alert.Success, "Webhook has been updated: "+wh.PayloadURL.String())
	h.Redirect(w, r, wh.Endpoint())
}

func (h WebhookUI) Destroy(u *auth.User, n *namespace.Namespace, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	wh, ok, err := h.Webhooks.Get(
		ctx,
		query.Where("id", "=", query.Arg(mux.Vars(r)["webhook"])),
		query.Where("namespace_id", "=", query.Arg(n.ID)),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to delete webhook"))
		return
	}

	if !ok {
		h.InternalServerError(w, r, errors.Benign("No such webhook"))
		return
	}

	wh.Namespace = n

	if err := h.Webhooks.Delete(ctx, wh); err != nil {
		h.InternalServerError(w, r, errors.Wrap(err, "Failed to delete webhook"))
		return
	}

	alert.Flash(sess, alert.Success, "Webhook has been deleted")
	h.Redirect(w, r, wh.Namespace.Endpoint("webhooks"))
}

func registerNamespaceUI(a auth.Authenticator, srv *server.Server) {
	ui := UI{
		Handler: NewHandler(srv),
	}

	index := srv.Restrict(a, []string{"namespace:read"}, ui.Index)
	create := srv.Restrict(a, []string{"namespace:write"}, ui.Create)
	store := srv.Restrict(a, []string{"namespace:write"}, ui.Store)

	root := srv.Router.PathPrefix("/").Subrouter()
	root.HandleFunc("/namespaces", index).Methods("GET")
	root.HandleFunc("/namespaces/create", create).Methods("GET")
	root.HandleFunc("/namespaces", store).Methods("POST")
	root.Use(srv.CSRF)

	a = namespace.NewAuth[*namespace.Namespace](a, "namespace", ui.Namespaces.Store)

	show := srv.Optional(a, ui.Namespace(ui.Show))
	edit := srv.Restrict(a, []string{"namespace:write", "owner"}, ui.Namespace(ui.Edit))
	update := srv.Restrict(a, []string{"namespace:write", "owner"}, ui.Namespace(ui.Update))
	destroy := srv.Restrict(a, []string{"namespace:delete"}, ui.Namespace(ui.Destroy))
	destroyCollab := srv.Restrict(a, []string{"namespace:delete", "owner"}, ui.Namespace(ui.DestroyCollaborator))

	sr := srv.Router.PathPrefix("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}").Subrouter()
	sr.HandleFunc("", srv.Optional(a, ui.Namespace(ui.Show))).Methods("GET")
	sr.HandleFunc("/-/badge.svg", ui.Badge).Methods("GET")
	sr.HandleFunc("/-/edit", edit).Methods("GET")
	sr.HandleFunc("/-/namespaces", show).Methods("GET")
	sr.HandleFunc("/-/images", show).Methods("GET")
	sr.HandleFunc("/-/objects", show).Methods("GET")
	sr.HandleFunc("/-/variables", show).Methods("GET")
	sr.HandleFunc("/-/keys", show).Methods("GET")
	sr.HandleFunc("/-/collaborators", show).Methods("GET")
	sr.HandleFunc("/-/collaborators/{collaborator}", destroyCollab).Methods("DELETE")
	sr.HandleFunc("/-/invites", srv.Restrict(a, []string{"invite:read", "owner"}, ui.Namespace(ui.Show))).Methods("GET")
	sr.HandleFunc("/-/webhooks", srv.Restrict(a, []string{"webhook:read"}, ui.Namespace(ui.Show))).Methods("GET")
	sr.HandleFunc("", update).Methods("PATCH")
	sr.HandleFunc("", destroy).Methods("DELETE")
	sr.Use(srv.CSRF)
}

func registerWebhookUI(a auth.Authenticator, srv *server.Server) {
	ui := WebhookUI{
		WebhookHandler: NewWebhookHandler(srv),
	}

	h := NewHandler(srv)

	create := srv.Restrict(a, []string{"webhook:write"}, h.Namespace(ui.Create))
	store := srv.Restrict(a, []string{"webhook:write"}, h.Namespace(ui.Store))
	show := srv.Restrict(a, []string{"webhook:read"}, h.Namespace(ui.Show))
	delivery := srv.Restrict(a, []string{"webhook:read"}, h.Namespace(ui.Delivery))
	redeliver := srv.Restrict(a, []string{"webhook:write"}, h.Namespace(ui.Redeliver))
	update := srv.Restrict(a, []string{"webhook:write"}, h.Namespace(ui.Update))
	destroy := srv.Restrict(a, []string{"webhook:delete"}, h.Namespace(ui.Destroy))

	sr := srv.Router.PathPrefix("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}/-/webhooks").Subrouter()
	sr.HandleFunc("/create", create).Methods("GET")
	sr.HandleFunc("", store).Methods("POST")
	sr.HandleFunc("/{webhook:[0-9]+}", show).Methods("GET")
	sr.HandleFunc("/{webhook:[0-9]+}/deliveries/{delivery:[0-9]+}", delivery).Methods("GET")
	sr.HandleFunc("/{webhook:[0-9]+}/deliveries/{delivery:[0-9]+}", redeliver).Methods("PATCH")
	sr.HandleFunc("/{webhook:[0-9]+}", update).Methods("PATCH")
	sr.HandleFunc("/{webhook:[0-9]+}", destroy).Methods("DELETE")
	sr.Use(srv.CSRF)
}

func registerInviteUI(a auth.Authenticator, srv *server.Server) {
	ui := InviteUI{
		InviteHandler: NewInviteHandler(srv),
	}

	h := NewHandler(srv)

	inviteAuth := namespace.NewAuth[*namespace.Invite](a, "invite", ui.Invites.Store)

	index := srv.Restrict(a, []string{"invite:read"}, ui.Index)
	update := srv.Restrict(inviteAuth, []string{"invite:write", "owner"}, ui.Update)
	destroy := srv.Restrict(inviteAuth, []string{"invite:write"}, ui.Destroy)

	a = namespace.NewAuth[*namespace.Namespace](a, "namespace", ui.Namespaces)

	store := srv.Restrict(a, []string{"invite:write", "owner"}, h.Namespace(ui.Store))

	root := srv.Router.PathPrefix("/").Subrouter()
	root.HandleFunc("/invites", index).Methods("GET")
	root.HandleFunc("/invites/{invite:[0-9]+}", update).Methods("PATCH")
	root.HandleFunc("/invites/{invite:[0-9]+}", destroy).Methods("DELETE")
	root.HandleFunc("/n/{username}/{namespace:[a-zA-Z0-9\\/?]+}/-/invites", store).Methods("POST")
	root.Use(srv.CSRF)
}

func RegisterUI(a auth.Authenticator, srv *server.Server) {
	registerNamespaceUI(a, srv)
	registerWebhookUI(a, srv)
	registerInviteUI(a, srv)
}
