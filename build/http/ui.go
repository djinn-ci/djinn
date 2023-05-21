package http

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"djinn-ci.com/alert"
	"djinn-ci.com/auth"
	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/template/form"
	"djinn-ci.com/user"
	"djinn-ci.com/variable"

	"github.com/andrewpillar/fs"
	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/gorilla/mux"
)

type UI struct {
	*Handler
}

func (h UI) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	p, err := h.Handler.Index(u, r)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get builds"))
		return
	}

	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.BuildIndex{
		Paginator: template.NewPaginator[*build.Build](tmpl.Page, p),
		Builds:    p.Items,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Create(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.BuildCreate{
		Form: form.New(sess, r),
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	b, f, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, f, errors.Wrap(err, "Failed to submit build"))
		return
	}

	alert.Flash(sess, alert.Success, "Build submitted: #"+strconv.FormatInt(b.Number, 10))
	h.Redirect(w, r, b.Endpoint())
}

func (h UI) Show(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	ctx := r.Context()

	if err := build.LoadRelations(ctx, h.DB, b); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to load build relations"))
		return
	}

	if err := build.LoadStageRelations(ctx, h.DB, b.Stages...); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to load build relations"))
		return
	}

	for _, st := range b.Stages {
		for _, j := range st.Jobs {
			j.Build = b
		}
	}

	tmpl := template.NewDashboard(u, sess, r)
	show := template.BuildShow{
		Page:  tmpl.Page,
		Build: b,
	}

	switch webutil.BasePath(r.URL.Path) {
	case "manifest":
		show.Partial = &template.BuildManifest{
			Build: b,
		}
	case "raw":
		parts := strings.Split(r.URL.Path, "/")
		attr := parts[len(parts)-2]

		if attr == "manifest" {
			webutil.Text(w, b.Manifest.String(), http.StatusOK)
			return
		}
		if attr == "output" {
			webutil.Text(w, b.Output.Elem, http.StatusOK)
			return
		}
	case "objects":
		p, err := h.Objects.Index(ctx, r.URL.Query(), query.Where("id", "IN", build.SelectObject(
			query.Columns("object_id"),
			query.Where("build_id", "=", query.Arg(b.ID)),
		)))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get objects"))
			return
		}

		if err := namespace.LoadResourceRelations[*object.Object](ctx, h.DB, p.Items...); err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to load object relations"))
			return
		}

		show.Partial = &template.ObjectIndex{
			Paginator: template.NewPaginator[*object.Object](tmpl.Page, p),
			Objects:   p.Items,
		}
	case "artifacts":
		p, err := h.Artifacts.Index(ctx, r.URL.Query(), query.Where("build_id", "=", query.Arg(b.ID)))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get artifacts"))
			return
		}

		for _, a := range p.Items {
			a.Build = b
		}

		show.Partial = &template.BuildArtifacts{
			Paginator: template.NewPaginator[*build.Artifact](tmpl.Page, p),
			Artifacts: p.Items,
		}
	case "variables":
		vv, err := h.Variables.All(ctx, query.Where("build_id", "=", query.Arg(b.ID)), query.OrderAsc("key"))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get variables"))
			return
		}

		unmasked := variable.GetUnmasked(sess.Values)

		for _, v := range vv {
			if _, ok := unmasked[v.VariableID.Elem]; ok && v.Masked {
				if err := variable.Unmask(h.AESGCM, v.Variable); err != nil {
					alert.Flash(sess, alert.Danger, "Could not unmask variable")
					h.Log.Error.Println(r.Method, r.URL, "could not unmask variable", errors.Err(err))
				}

				v.Value = v.Variable.Value
				continue
			}

			if v.Masked {
				v.Value = variable.MaskString
			}
		}

		mm := database.Map[*build.Variable, database.Model](vv, func(v *build.Variable) database.Model {
			return v
		})

		if err := variable.Loader(h.DB).Load(ctx, "variable_id", "id", mm...); err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to load variables"))
			return
		}

		show.Partial = &template.BuildVariables{
			Page:      tmpl.Page,
			Unmasked:  unmasked,
			Variables: vv,
		}
	case "keys":
		kk, err := h.Keys.All(ctx, query.Where("build_id", "=", query.Arg(b.ID)), query.OrderAsc("name"))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get keys"))
			return
		}

		show.Partial = &template.BuildKeys{
			Keys: kk,
		}
	case "tags":
		isCollaborator := u.ID == b.UserID

		if b.NamespaceID.Valid {
			isCollaborator = true

			if err := b.Namespace.IsCollaborator(ctx, h.DB, u); err != nil {
				if !errors.Is(err, database.ErrPermission) {
					h.Error(w, r, errors.Wrap(err, "Failed to get tags"))
					return
				}
				isCollaborator = false
			}
		}

		if isCollaborator {
			tmpl.Page.User.Grant("tag:modify")
		}

		tt, err := h.Tags.All(ctx, query.Where("build_id", "=", query.Arg(b.ID)), query.OrderAsc("name"))

		if err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to get tags"))
			return
		}

		mm := make([]database.Model, 0, len(tt))

		for _, t := range tt {
			mm = append(mm, t)
			t.Build = b
		}

		if err := user.Loader(h.DB).Load(ctx, "user_id", "id", mm...); err != nil {
			h.Error(w, r, errors.Wrap(err, "Failed to load tag relations"))
			return
		}

		show.Partial = &template.BuildTags{
			Page:  tmpl.Page,
			Build: b,
			Tags:  tt,
		}
	}

	tmpl.Partial = &show
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Destroy(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := b.Kill(h.Redis); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to kill build"))
		return
	}

	alert.Flash(sess, alert.Success, "Build killed")
	h.RedirectBack(w, r)
}

func (h UI) TogglePin(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	var (
		err error
		msg string
	)

	ctx := r.Context()

	switch webutil.BasePath(r.URL.Path) {
	case "pin":
		err = b.Pin(ctx, h.DB)
		msg = "Build pinned"
	case "unpin":
		err = b.Unpin(ctx, h.DB)
		msg = "Build unpinned"
	}

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to update pin"))
		return
	}

	h.Queues.Produce(r.Context(), "events", &build.PinEvent{Build: b})

	alert.Flash(sess, alert.Success, msg)
	h.RedirectBack(w, r)
}

func (h UI) ShowJob(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	ctx := r.Context()

	j, ok, err := h.Jobs.Get(
		ctx,
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("name", "=", query.Arg(mux.Vars(r)["name"])),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get job"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if webutil.BasePath(r.URL.Path) == "raw" {
		webutil.Text(w, j.Output.Elem, http.StatusOK)
		return
	}

	j.Stage, _, err = h.Stages.Get(ctx, query.Where("id", "=", query.Arg(j.StageID)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	b.Trigger, _, err = h.Triggers.Get(ctx, query.Where("build_id", "=", query.Arg(b.ID)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	p, err := h.Artifacts.Index(ctx, r.URL.Query(), query.Where("job_id", "=", query.Arg(j.ID)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	for _, a := range p.Items {
		a.Build = b
	}

	j.Build = b

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.BuildJob{
		Page: tmpl.Page,
		Build: &template.BuildShow{
			Build: b,
		},
		Artifacts: &template.BuildArtifacts{
			Paginator: template.NewPaginator[*build.Artifact](tmpl.Page, p),
			Artifacts: p.Items,
		},
		Job: j,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Download(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	a, ok, err := h.Artifacts.Get(
		r.Context(),
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("name", "=", query.Arg(name)),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get artifact"))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	f, err := a.Open(h.Artifacts.FS)

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			h.NotFound(w, r)
			return
		}

		h.Error(w, r, errors.Wrap(err, "Failed to get artifact"))
		return
	}

	defer f.Close()

	http.ServeContent(w, r, a.Name, a.CreatedAt, f.(io.ReadSeeker))
}

func (h UI) StoreTag(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	if _, err := h.Handler.StoreTag(u, b, r); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to tag build"))
		return
	}
	h.RedirectBack(w, r)
}

func (h UI) DestroyTag(u *auth.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.Handler.DestroyTag(r.Context(), b, mux.Vars(r)["name"]); err != nil {
		if errors.Is(err, database.ErrNoRows) {
			h.NotFound(w, r)
			return
		}

		h.Error(w, r, errors.Wrap(err, "Failed to delete tag"))
		return
	}

	alert.Flash(sess, alert.Success, "Tag has been deleted")
	h.RedirectBack(w, r)
}

func RegisterUI(a auth.Authenticator, srv *server.Server) {
	ui := UI{
		Handler: NewHandler(srv),
	}

	index := srv.Restrict(a, []string{"build:read"}, ui.Index)
	create := srv.Restrict(a, []string{"build:write"}, ui.Create)
	store := srv.Restrict(a, []string{"build:write"}, ui.Store)

	root := srv.Router.PathPrefix("/builds").Subrouter()
	root.HandleFunc("", index).Methods("GET")
	root.HandleFunc("/create", create).Methods("GET")
	root.HandleFunc("", store).Methods("POST")
	root.Use(srv.CSRF)

	show := srv.Optional(a, ui.Build(ui.Show))
	destroy := srv.Restrict(a, []string{"build:delete"}, ui.Build(ui.Destroy))
	pin := srv.Restrict(a, []string{"build:write"}, ui.Build(ui.TogglePin))
	showJob := srv.Restrict(a, []string{"build:read"}, ui.Build(ui.ShowJob))
	download := srv.Restrict(a, []string{"build:read"}, ui.Build(ui.Download))
	storeTag := srv.Restrict(a, []string{"build:write"}, ui.Build(ui.StoreTag))
	destroyTag := srv.Restrict(a, []string{"build:delete"}, ui.Build(ui.DestroyTag))

	sr := srv.Router.PathPrefix("/b/{username}/{build:[0-9]+}").Subrouter()
	sr.HandleFunc("", show).Methods("GET")
	sr.HandleFunc("", destroy).Methods("DELETE")
	sr.HandleFunc("/pin", pin).Methods("PATCH")
	sr.HandleFunc("/unpin", pin).Methods("PATCH")
	sr.HandleFunc("/manifest", show).Methods("GET")
	sr.HandleFunc("/manifest/raw", show).Methods("GET")
	sr.HandleFunc("/output/raw", show).Methods("GET")
	sr.HandleFunc("/objects", show).Methods("GET")
	sr.HandleFunc("/variables", show).Methods("GET")
	sr.HandleFunc("/keys", show).Methods("GET")
	sr.HandleFunc("/jobs/{name}", showJob).Methods("GET")
	sr.HandleFunc("/jobs/{name}/output/raw", showJob).Methods("GET")
	sr.HandleFunc("/artifacts", show).Methods("GET")
	sr.HandleFunc("/artifacts/{name}", download).Methods("GET")
	sr.HandleFunc("/tags", show).Methods("GET")
	sr.HandleFunc("/tags", storeTag).Methods("POST")
	sr.HandleFunc("/tags/{name:.+}", destroyTag).Methods("DELETE")
	sr.Use(srv.CSRF)
}
