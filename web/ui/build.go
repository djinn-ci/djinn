package ui

import (
	"database/sql"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/model/types"
	"github.com/andrewpillar/thrall/runner"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/build"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"

	"github.com/go-redis/redis"

	"github.com/RichardKnop/machinery/v1"
)

type Build struct {
	web.Handler

	Client  *redis.Client
	Builds  model.BuildStore
	Drivers map[string]struct{}
	Queue   *machinery.Server
}

func (h Build) Build(r *http.Request) *model.Build {
	val := r.Context().Value("build")

	b, _ := val.(*model.Build)

	return b
}

func (h Build) indexPage(builds model.BuildStore, r *http.Request, opts ...query.Option) (build.IndexPage, error) {
	u := h.User(r)

	tag := r.URL.Query().Get("tag")
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")

	indexOpts := []query.Option{
		model.BuildTag(tag),
		model.BuildSearch(search),
		model.BuildStatus(status),
		query.OrderDesc("created_at"),
	}

	bb, err := builds.Index(append(opts, indexOpts...)...)

	if err != nil {
		return build.IndexPage{}, errors.Err(err)
	}

	p := build.IndexPage{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Builds: bb,
		Search: search,
		Status: status,
		Tag:    tag,
	}

	return p, nil
}

func (h Build) Index(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	p, err := h.indexPage(u.BuildStore(), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	d := template.NewDashboard(&p, r.URL, h.Alert(w, r))

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

	d := template.NewDashboard(p, r.URL, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) Store(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

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

	if _, ok := h.Drivers[m.Driver["type"]]; !ok {
		errs := form.NewErrors()
		errs.Put("manifest", errors.New("Driver " + m.Driver["type"] + " is not supported"))

		h.FlashForm(w, r, f)
		h.FlashErrors(w, r, errs)

		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	secret, err := crypto.HashNow()

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create build: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	builds := u.BuildStore()

	b := builds.New()
	b.User = u
	b.Manifest = f.Manifest
	b.Secret = sql.NullString{
		String: secret,
		Valid:  true,
	}

	if f.Namespace != "" {
		n, err := u.NamespaceStore().FindOrCreate(f.Namespace)

		if err != nil {
			log.Error.Println(errors.Err(err))
			h.FlashAlert(w, r, template.Danger("Failed to create build: " + errors.Cause(err).Error()))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		b.Namespace = n
		b.NamespaceID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}
	}

	if err := builds.Create(b); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create build: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	triggers := b.TriggerStore()

	t := triggers.New()
	t.Type = types.Manual
	t.Comment = f.Comment
	t.Data.User = u.Username
	t.Data.Email = u.Email

	if err := triggers.Create(t); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := b.Submit(h.Queue); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	tags := b.TagStore()
	tt := make([]*model.Tag, len(f.Tags), len(f.Tags))

	for i, name := range f.Tags {
		t := tags.New()
		t.UserID = u.ID
		t.Name = name

		tt[i] = t
	}

	if err := tags.Create(tt...); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create build: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Build submitted: #" + strconv.FormatInt(b.ID, 10)))

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h Build) Show(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)
	b := h.Build(r)

	if err := b.Show(); err != nil {
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
			web.Text(w, b.Manifest, http.StatusOK)
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

	d := template.NewDashboard(p, r.URL, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) Kill(w http.ResponseWriter, r *http.Request) {
	b := h.Build(r)

	if b.Status != runner.Running {
		h.FlashAlert(w, r, template.Danger("Failed to kill build: build is not running"))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if _, err := h.Client.Publish("kill", b.Secret.String).Result(); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to kill build: " + err.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Build killed"))
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
