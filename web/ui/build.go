package ui

import (
	"database/sql"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/model/query"
	"github.com/andrewpillar/thrall/model/types"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/build"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"

	"github.com/RichardKnop/machinery/v1"
)

type Build struct {
	web.Handler

	Queues map[string]*machinery.Server
}

type BuildObject struct {
	Build
}

type BuildVariable struct {
	Build
}

func (h Build) build(r *http.Request) (*model.Build, error) {
	vars := mux.Vars(r)

	u, err := h.Users.FindByUsername(vars["username"])

	if err != nil {
		return nil, errors.Err(err)
	}

	id, _ := strconv.ParseInt(vars["build"], 10, 64)

	b, err := u.BuildStore().Find(id)

	return b, errors.Err(err)
}

func (h Build) Index(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	tag := r.URL.Query().Get("tag")
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")

	bb, err := u.BuildStore().Index(
		model.BuildTag(tag),
		model.BuildSearch(search),
		model.BuildStatus(status),
		query.OrderDesc("created_at"),
	)

	p := &build.IndexPage{
		BasePage: template.BasePage{
			URI: r.URL.Path,
		},
		Builds: bb,
		Search: search,
		Status: status,
		Tag:    tag,
	}

	d := template.NewDashboard(p, r.URL.Path, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) Create(w http.ResponseWriter, r *http.Request) {
	p := &build.CreatePage{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Errors(w, r),
			Fields: h.Form(w, r),
		},
	}

	d := template.NewDashboard(p, r.URL.RequestURI(), h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) Store(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create build: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	f := &form.Build{}

	if err := h.ValidateForm(f, w, r); err != nil {
		if _, ok := err.(form.Errors); ok {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create build: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	m, _ := config.DecodeManifest(strings.NewReader(f.Manifest))

	srv, ok := h.Queues[m.Driver.Type]

	if !ok {
		errs := form.NewErrors()
		errs.Put("manifest", errors.New("Driver " + m.Driver.Type + " is not supported"))

		h.FlashForm(w, r, f)
		h.FlashErrors(w, r, errs)

		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	b := u.BuildStore().New()
	b.User = u
	b.Manifest = f.Manifest

	if f.Namespace != "" {
		n, err := u.NamespaceStore().FindOrCreate(f.Namespace)

		if err != nil {
			log.Error.Println(errors.Err(err))
			h.FlashAlert(w, r, template.Danger("Failed to create build: " + errors.Cause(err).Error()))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		b.NamespaceID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}
	}

	if err := b.Create(); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create build: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	t := b.TriggerStore().New()
	t.Type = types.Manual
	t.Comment = f.Comment
	t.Data.User = u.Username
	t.Data.Email = u.Email

	if err := t.Create(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := b.Submit(srv); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	tags := b.TagStore()

	for _, name := range f.Tags {
		t := tags.New()
		t.Name = name

		if err := t.Create(); err != nil {
			log.Error.Println(errors.Err(err))
			h.FlashAlert(w, r, template.Danger("Failed to create build: " + errors.Cause(err).Error()))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}
	}

	h.FlashAlert(w, r, template.Success("Build submitted: #" + strconv.FormatInt(b.ID, 10)))

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h Build) Show(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	u, err := h.Users.FindByUsername(vars["username"])

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	id, _ := strconv.ParseInt(vars["build"], 10, 64)

	b, err := u.BuildStore().Show(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if b.IsZero() {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if filepath.Base(r.URL.Path) == "raw" {
		parts := strings.Split(r.URL.Path, "/")
		field := parts[len(parts) - 2]

		if field == "manifest" {
			web.Text(w, b.Manifest, http.StatusOK)
			return
		}

		if field == "output" {
			web.Text(w, b.Output.String, http.StatusOK)
			return
		}
	}

	p := &build.ShowPage{
		BasePage: template.BasePage{
			URI: r.URL.Path,
		},
		Build:        b,
		ShowManifest: filepath.Base(r.URL.Path) == "manifest",
		ShowOutput:   filepath.Base(r.URL.Path) == "output",
	}

	d := template.NewDashboard(p, r.URL.Path, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h BuildObject) Index(w http.ResponseWriter, r *http.Request) {
	b, err := h.build(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if b.IsZero() {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	objects := b.BuildObjectStore()

	oo, err := objects.All()

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := objects.LoadObjects(oo); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &build.ObjectIndexPage{
		ShowPage: build.ShowPage{
			BasePage: template.BasePage{
				URI: r.URL.Path,
			},
			Build: b,
		},
		Objects: oo,
	}

	d := template.NewDashboard(p, r.URL.Path, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h BuildVariable) Index(w http.ResponseWriter, r *http.Request) {
	b, err := h.build(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if b.IsZero() {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	vv, err := b.BuildVariableStore().All()

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &build.VariableIndexPage{
		ShowPage: build.ShowPage{
			BasePage: template.BasePage{
				URI: r.URL.Path,
			},
			Build: b,
		},
		Variables: vv,
	}

	d := template.NewDashboard(p, r.URL.Path, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}
