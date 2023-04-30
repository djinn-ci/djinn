package http

import (
	"fmt"
	"net/http"
	"regexp"

	"djinn-ci.com/alert"
	"djinn-ci.com/auth"
	"djinn-ci.com/build"
	"djinn-ci.com/cron"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/template/form"

	"github.com/andrewpillar/query"
)

type UI struct {
	*Handler
}

func (h UI) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	p, err := h.Handler.Index(u, r)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get cron jobs"))
		return
	}

	sess, _ := h.Session(r)

	ld := namespace.Loader(h.DB)

	mm := database.Map[*cron.Cron, database.Model](p.Items, func(c *cron.Cron) database.Model {
		return c
	})

	if err := ld.Load(r.Context(), "namespace_id", "id", mm...); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to load namespaces"))
		return
	}

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.CronIndex{
		Paginator: template.NewPaginator[*cron.Cron](tmpl.Page, p),
		Crons:     p.Items,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Create(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.CronForm{
		Form: form.New(sess, r),
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	msgfmt := "Cron job has been added: %s it will next trigger on %s"

	sess, _ := h.Session(r)

	c, f, err := h.Handler.Store(u, r)

	if err != nil {
		h.FormError(w, r, f, err)
		return
	}

	alert.Flash(sess, alert.Success, fmt.Sprintf(msgfmt, c.Name, c.NextRun.Format("Mon, 2 Jan 15:04 2006")))
	h.Redirect(w, r, "/cron")
}

func (h UI) Show(u *auth.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	sess, _ := h.Session(r)

	p, err := h.Builds.Index(
		ctx,
		r.URL.Query(),
		query.Where("id", "IN", cron.SelectBuild(
			query.Columns("build_id"),
			query.Where("cron_id", "=", query.Arg(c.ID)),
		)),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to get cron job"))
		return
	}

	if err := build.LoadRelations(ctx, h.DB, p.Items...); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to load build relations"))
		return
	}

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.CronShow{
		Page: tmpl.Page,
		Cron: c,
		Builds: &template.BuildIndex{
			Paginator: template.NewPaginator[*build.Build](tmpl.Page, p),
			Builds:    p.Items,
		},
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Edit(u *auth.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	f := form.New(sess, r)

	f.Fields["name"] = c.Name
	f.Fields["manifest"] = c.Manifest.String()
	f.Fields["schedule"] = c.Schedule.String()

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.CronForm{
		Form: f,
		Cron: c,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Update(u *auth.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	c, f, err := h.Handler.Update(u, c, r)

	if err != nil {
		h.FormError(w, r, f, err)
		return
	}

	alert.Flash(sess, alert.Success, "Cron job has been updated")
	h.Redirect(w, r, c.Endpoint())
}

var reCronUri = regexp.MustCompile("/cron/[0-9]+")

func (h UI) Destroy(u *auth.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.Handler.Destroy(r.Context(), c); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to delete cron job"))
		return
	}

	alert.Flash(sess, alert.Success, "Deleted cron job")

	if reCronUri.Match([]byte(r.Header.Get("Referer"))) {
		h.Redirect(w, r, "/cron")
		return
	}
	h.RedirectBack(w, r)
}

func RegisterUI(a auth.Authenticator, srv *server.Server) {
	ui := UI{
		Handler: NewHandler(srv),
	}

	index := ui.Restrict(a, []string{"cron:read"}, ui.Index)
	create := ui.Restrict(a, []string{"cron:write"}, ui.Create)
	store := ui.Restrict(a, []string{"cron:write"}, ui.Store)

	a = namespace.NewAuth[*cron.Cron](a, "cron", cron.NewStore(srv.DB))

	show := ui.Restrict(a, []string{"cron:read", "build:read"}, ui.Cron(ui.Show))
	edit := ui.Restrict(a, []string{"cron:write"}, ui.Cron(ui.Edit))
	update := ui.Restrict(a, []string{"cron:write"}, ui.Cron(ui.Update))
	destroy := ui.Restrict(a, []string{"cron:delete"}, ui.Cron(ui.Destroy))

	sr := srv.Router.PathPrefix("/cron").Subrouter()
	sr.HandleFunc("", index).Methods("GET")
	sr.HandleFunc("/create", create).Methods("GET")
	sr.HandleFunc("", store).Methods("POST")
	sr.HandleFunc("/{cron:[0-9]+}", show).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}/edit", edit).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}", update).Methods("PATCH")
	sr.HandleFunc("/{cron:[0-9]+}", destroy).Methods("DELETE")
	sr.Use(srv.CSRF)
}
