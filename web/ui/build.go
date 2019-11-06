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
	u := h.Core.User(r)

	p, err := h.indexPage(u.BuildStore(), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	d := template.NewDashboard(&p, r.URL, h.Core.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) Create(w http.ResponseWriter, r *http.Request) {
	p := &build.CreatePage{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Core.Errors(w, r),
			Fields: h.Core.Form(w, r),
		},
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) Store(w http.ResponseWriter, r *http.Request) {
	b, err := h.Core.UnmarshalAndValidate(w, r)

	if err != nil {
		cause := errors.Cause(err)

		switch cause {
		case core.ErrValidationFailed:
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		case core.ErrUnsupportedDriver:
			errs := form.NewErrors()
			errs.Put("manifest", cause)

			h.Core.FlashErrors(w, r, errs)

			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		case core.ErrAccessDenied:
			h.Core.FlashAlert(w, r, template.Danger("Failed to create build: could not add to namespace"))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		default:
			log.Error.Println(errors.Err(err))

			h.Core.FlashAlert(w, r, template.Danger("Failed to create build: " + cause.Error()))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}
	}

	if err := h.Core.Create(b); err != nil {
		cause := errors.Cause(err)

		h.Core.FlashAlert(w, r, template.Danger("Failed to create build: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if err := h.Core.Submit(b, h.Core.Queues[b.Manifest.Driver["type"]]); err != nil {
		cause := errors.Cause(err)

		h.Core.FlashAlert(w, r, template.Danger("Failed to create build: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("Build submitted: #" + strconv.FormatInt(b.ID, 10)))

	http.Redirect(w, r, b.UIEndpoint(), http.StatusSeeOther)
}

func (h Build) Show(w http.ResponseWriter, r *http.Request) {
	u := h.Core.User(r)

	b, err := h.Core.Show(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	base := filepath.Base(r.URL.Path)

	var p template.Dashboard

	sp := build.ShowPage{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Build: b,
		CSRF:  string(csrf.TemplateField(r)),
	}

	switch base {
	case "manifest":
		p = &build.ShowManifest{
			ShowPage: sp,
		}

		break
	case "output":
		p = &build.ShowOutput{
			ShowPage: sp,
		}

		break
	case "raw":
		parts := strings.Split(r.URL.Path, "/")
		pen := parts[len(parts) - 2]

		if pen == "manifest" {
			web.Text(w, b.Manifest.String(), http.StatusOK)
			return
		}

		if pen == "output" {
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

		p = &build.ShowObjects{
			ShowPage: sp,
			Objects:  oo,
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

		p = &build.ShowArtifacts{
			ShowPage:  sp,
			Search:    search,
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

		p = &build.ShowVariables{
			ShowPage:  sp,
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

		p = &build.ShowKeys{
			ShowPage: sp,
			Keys:     kk,
		}

		break
	case "tags":
		tt, err := b.TagStore().Index()

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p = &build.ShowTags{
			ShowPage: sp,
			CSRF:     string(csrf.TemplateField(r)),
			Tags:     tt,
		}

		break
	default:
		p = &sp
		break
	}

	d := template.NewDashboard(p, r.URL, h.Core.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) Kill(w http.ResponseWriter, r *http.Request) {
	if err := h.Core.Kill(r); err != nil {
		cause := errors.Cause(err)

		h.Core.FlashAlert(w, r, template.Danger("Failed to kill build: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.Core.FlashAlert(w, r, template.Success("Build killed"))
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
