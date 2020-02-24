package ui

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/build"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/core"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
)

type Build struct {
	Core core.Build
}

func (h Build) indexPage(builds model.BuildStore, r *http.Request, opts ...query.Option) (build.IndexPage, error) {
	u := h.Core.User(r)

	bb, paginator, err := h.Core.Index(builds, r, opts...)

	if err != nil {
		return build.IndexPage{}, errors.Err(err)
	}

	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")
	tag := r.URL.Query().Get("tag")

	p := build.IndexPage{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Paginator: paginator,
		Builds:    bb,
		Search:    search,
		Status:    status,
		Tag:       tag,
	}

	return p, nil
}

func (h Build) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	u := h.Core.User(r)

	p, err := h.indexPage(u.BuildStore(), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	d := template.NewDashboard(&p, r.URL, h.Core.Alert(sess), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	p := &build.CreatePage{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Core.FormErrors(sess),
			Fields: h.Core.FormFields(sess),
		},
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(sess), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) Store(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	b, err := h.Core.Store(r, sess)

	if err != nil {
		cause := errors.Cause(err)

		switch cause {
		case form.ErrValidation:
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		case model.ErrDriver:
			errs := form.NewErrors()
			errs.Put("manifest", cause)

			sess.AddFlash(errs, "form_errors")
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		case model.ErrPermission:
			sess.AddFlash(template.Danger("Failed to create build: could not add to namespace"), "alert")
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		default:
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create build: " + cause.Error()), "alert")
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}
	}

	if err := h.Core.Submit(b, h.Core.Queues[b.Manifest.Driver["type"]]); err != nil {
		cause := errors.Cause(err)
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create build: " + cause.Error()), "alert")
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	sess.AddFlash(template.Success("Build submitted: #" + strconv.FormatInt(b.ID, 10)), "alert")
	http.Redirect(w, r, b.UIEndpoint(), http.StatusSeeOther)
}

func (h Build) Show(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	u := h.Core.User(r)

	b, err := h.Core.Show(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	base := filepath.Base(r.URL.Path)

	p := &build.ShowPage{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Build: b,
		CSRF:  string(csrf.TemplateField(r)),
	}

	switch base {
	case "manifest":
		p.Section = &build.ShowManifest{
			Build: b,
		}

		break
	case "raw":
		parts := strings.Split(r.URL.Path, "/")
		attr := parts[len(parts) - 2]

		if attr == "manifest" {
			web.Text(w, b.Manifest.String(), http.StatusOK)
			return
		}

		if attr == "output" {
			web.Text(w, b.Output.String, http.StatusOK)
			return
		}

		break
	case "objects":
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

		p.Section = &build.ShowObjects{
			Build:   b,
			Objects: oo,
		}

		break
	case "artifacts":
		search := r.URL.Query().Get("search")

		aa, err := b.ArtifactStore().Index(model.Search("name", search))

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &build.ShowArtifacts{
			URL:       r.URL,
			Search:    search,
			Build:     b,
			Artifacts: aa,
		}

		break
	case "variables":
		vv, err := b.BuildVariableStore().All()

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &build.ShowVariables{
			Build:     b,
			Variables: vv,
		}

		break
	case "keys":
		keys := b.BuildKeyStore()

		kk, err := keys.All()

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &build.ShowKeys{
			Build: b,
			Keys:  kk,
		}

		break
	case "tags":
		tt, err := b.TagStore().Index()

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &build.ShowTags{
			CSRF:  string(csrf.TemplateField(r)),
			Build: b,
			Tags:  tt,
		}

		break
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(sess), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) Kill(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Core.Session(r)
	defer save(r, w)

	if err := h.Core.Kill(r); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		sess.AddFlash(template.Danger("Failed to kill build: " + cause.Error()), "alert")
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	sess.AddFlash(template.Success("Build killed"), "alert")
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
