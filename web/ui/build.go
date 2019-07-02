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
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/build"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"

	"github.com/RichardKnop/machinery/v1"
)

type Build struct {
	web.Handler

	queues     map[string]*machinery.Server
	namespaces *model.NamespaceStore
}

func NewBuild(h web.Handler, queues map[string]*machinery.Server, namespaces *model.NamespaceStore) Build {
	return Build{
		Handler:    h,
		queues:     queues,
		namespaces: namespaces,
	}
}

func (h Build) build(r *http.Request) (*model.Build, error) {
	u, err := h.User(r)

	if err != nil {
		return nil, errors.Err(err)
	}

	vars := mux.Vars(r)

	id, err := strconv.ParseInt(vars["build"], 10, 64)

	if err != nil {
		return &model.Build{}, nil
	}

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
		model.OrderDesc("created_at"),
	)

	p := &build.IndexPage{
		Page: template.Page{
			URI: r.URL.Path,
		},
		Builds: bb,
		Search: search,
		Status: status,
		Tag:    tag,
	}

	d := template.NewDashboard(p, r.URL.Path)

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) Create(w http.ResponseWriter, r *http.Request) {
	p := &build.CreatePage{
		Form: template.Form{
			CSRF:   csrf.TemplateField(r),
			Errors: h.Errors(w, r),
			Form:   h.Form(w, r),
		},
	}

	d := template.NewDashboard(p, r.URL.RequestURI())

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) Store(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	f := &form.Build{}

	if err := h.ValidateForm(f, w, r); err != nil {
		if _, ok := err.(form.Errors); ok {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	m, _ := config.DecodeManifest(strings.NewReader(f.Manifest))

	srv, ok := h.queues[m.Driver.Type]

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
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		b.NamespaceID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}
	}

	if err := b.Create(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	t := b.TriggerStore().New()
	t.Type = model.Manual
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
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h Build) Show(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)

	id, err := strconv.ParseInt(vars["build"], 10, 64)

	if err != nil {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	b, err := u.BuildStore().Show(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if b.IsZero() || u.ID != b.UserID {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	p := &build.ShowPage{
		Page: template.Page{
			URI: r.URL.Path,
		},
		CSRF:  csrf.TemplateField(r),
		Build: b,
	}

	d := template.NewDashboard(p, r.URL.Path)

	web.HTML(w, template.Render(d), http.StatusOK)
}

// ShowMeta displays meta information about the build, manifest, output, etc.
func (h Build) ShowMeta(w http.ResponseWriter, r *http.Request) {
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
		Page: template.Page{
			URI: r.URL.Path,
		},
		Build:        b,
		ShowManifest: filepath.Base(r.URL.Path) == "manifest",
		ShowOutput:   filepath.Base(r.URL.Path) == "output",
	}

	d := template.NewDashboard(p, r.URL.Path)

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) IndexRelation(w http.ResponseWriter, r *http.Request) {
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

	if filepath.Base(r.URL.Path) == "objects" {
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
				Page: template.Page{
					URI: r.URL.Path,
				},
				Build: b,
			},
			Objects: oo,
		}

		d := template.NewDashboard(p, r.URL.Path)

		web.HTML(w, template.Render(d), http.StatusOK)
	} else if filepath.Base(r.URL.Path) == "variables" {
		vv, err := b.BuildVariableStore().All()

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p := &build.VariableIndexPage{
			ShowPage: build.ShowPage{
				Page: template.Page{
					URI: r.URL.Path,
				},
				Build: b,
			},
			Variables: vv,
		}

		d := template.NewDashboard(p, r.URL.Path)

		web.HTML(w, template.Render(d), http.StatusOK)
	}
}

func (h Build) Tag(w http.ResponseWriter, r *http.Request) {
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

	f := &form.Build{}

	if err := form.Unmarshal(f, r); err != nil {
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
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
