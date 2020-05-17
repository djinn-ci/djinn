package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/build"
	buildtemplate "github.com/andrewpillar/thrall/build/template"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/runner"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type UI struct {
	Build
}

func (h UI) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	bb, paginator, err := h.IndexWithRelations(build.NewStore(h.DB, u), r.URL.Query())

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()

	tag := q.Get("tag")
	search := q.Get("search")
	status := q.Get("status")

	p := &buildtemplate.Index{
		BasePage:  template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Paginator: paginator,
		Builds:    bb,
		Search:    search,
		Status:    status,
		Tag:       tag,
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), string(csrf.TemplateField(r)))
	save(r,w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	p := &buildtemplate.Create{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.FormErrors(sess),
			Fields: h.FormFields(sess),
		},
	}

	d := template.NewDashboard(p, r.URL, h.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	b, err := h.StoreModel(r, sess)

	if err != nil {
		cause := errors.Cause(err)

		if _, ok := cause.(form.Errors); ok {
			h.RedirectBack(w, r)
			return
		}

		switch cause {
		case build.ErrDriver:
			errs := form.NewErrors()
			errs.Put("manifest", cause)

			sess.AddFlash(errs, "form_errors")
			h.RedirectBack(w, r)
			return
		case namespace.ErrPermission:
			sess.AddFlash(template.Danger("Failed to create build: could not add to namespace"), "alert")
			h.RedirectBack(w, r)
			return
		default:
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create build"), "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	if err := h.Submit(b, h.Queues[b.Manifest.Driver["type"]]); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create build"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Build submitted: #" + strconv.FormatInt(b.ID, 10)), "alert")
	h.Redirect(w, r, b.Endpoint())
}

func (h UI) Show(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	b, err := h.ShowWithRelations(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	base := filepath.Base(r.URL.Path)
	csrfField := csrf.TemplateField(r)

	p := &buildtemplate.Show{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Build: b,
		CSRF:  string(csrf.TemplateField(r)),
	}

	switch base {
	case "manifest":
		p.Section = &buildtemplate.Manifest{Build: b}
	case "raw":
		parts := strings.Split(r.URL.Path, "/")
		attr := parts[len(parts)-2]

		if attr == "manifest" {
			web.Text(w, b.Manifest.String(), http.StatusOK)
			return
		}
		if attr == "output" {
			web.Text(w, b.Output.String, http.StatusOK)
			return
		}
	case "objects":
		objects := build.NewObjectStore(h.DB, b)

		oo, err := objects.All()

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		mm := model.Slice(len(oo), build.ObjectModel(oo))

		if err := h.Objects.Load("id", model.MapKey("object_id", mm), model.Bind("object_id", "id", mm...)); err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &buildtemplate.Objects{
			Build:   b,
			Objects: oo,
		}
	case "artifacts":
		search := r.URL.Query().Get("search")

		aa, err := build.NewArtifactStore(h.DB, b).All(model.Search("name", search))

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &buildtemplate.Artifacts{
			URL:       r.URL,
			Search:    search,
			Build:     b,
			Artifacts: aa,
		}
	case "variables":
		vv, err := build.NewVariableStore(h.DB, b).All()

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &buildtemplate.Variables{
			Build:     b,
			Variables: vv,
		}
	case "keys":
		kk, err := build.NewKeyStore(h.DB, b).All()

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &buildtemplate.Keys{
			Build: b,
			Keys:  kk,
		}
	case "tags":
		tt, err := build.NewTagStore(h.DB, b).All()

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		mm := model.Slice(len(tt), build.TagModel(tt))

		if err := h.Users.Load("id", model.MapKey("user_id", mm), model.Bind("user_id", "id", mm...)); err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &buildtemplate.Tags{
			CSRF:  string(csrfField),
			Build: b,
			Tags:  tt,
		}
	}

	d := template.NewDashboard(p, r.URL, h.Alert(sess), string(csrfField))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Kill(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	b := h.Model(r)

	if b.Status != runner.Running {
		sess.AddFlash(template.Danger("Build not running"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if _, err := h.Client.Publish(fmt.Sprintf("kill-%v", b.ID), b.Secret.String).Result(); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to kill build"), "alert")
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Success("Build killed"), "alert")
	h.RedirectBack(w, r)
}

func (h UI) TagStore(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if _, err := h.TagStoreModel(r); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to tag build"), "alert")
		h.RedirectBack(w, r)
		return
	}
	h.RedirectBack(w, r)
}

func (h UI) TagDestroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.TagDelete(r); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to tag build"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Tag has been deleted"), "alert")
	h.RedirectBack(w, r)
}

func (h UI) JobShow(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	j, err := h.JobGet(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if filepath.Base(r.URL.Path) == "raw" {
		web.Text(w, j.Output.String, http.StatusOK)
		return
	}

	p := &buildtemplate.Job{
		BasePage: template.BasePage{URL: r.URL},
		Job:      j,
	}

	d := template.NewDashboard(p, r.URL, h.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) ArtifactShow(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	vars := mux.Vars(r)

	buildId, _ := strconv.ParseInt(vars["build"], 10, 64)
	artifactId, _ := strconv.ParseInt(vars["artifact"], 10, 64)

	b, err := build.NewStore(h.DB, u).Get(query.Where("id", "=", buildId))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	a, err := build.NewArtifactStore(h.DB, b).Get(query.Where("id", "=", artifactId))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if a.IsZero() || a.Name != vars["name"] {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	f, err := h.FileStore.Open(a.Hash)

	if err != nil {
		if os.IsNotExist(errors.Cause(err)) {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	defer f.Close()

	http.ServeContent(w, r, a.Name, a.CreatedAt, f)
}
