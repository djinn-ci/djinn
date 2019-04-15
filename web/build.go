package web

import (
	"database/sql"
	"fmt"
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
	"github.com/andrewpillar/thrall/queue"

	"github.com/gorilla/mux"
)

type Build struct {
	Handler
}

func NewBuild(h Handler) Build {
	return Build{Handler: h}
}

func (h Build) Create(w http.ResponseWriter, r *http.Request) {
	p := &build.CreatePage{
		Errors: h.errors(w, r),
		Form:   h.form(w, r),
	}

	d := template.NewDashboard(p, r.URL.RequestURI())

	HTML(w, template.Render(d), http.StatusOK)
}

func (h Build) Store(w http.ResponseWriter, r *http.Request) {
	u, err := h.userFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	f := &form.Build{}

	if err := h.handleRequestData(f, w, r); err != nil {
		return
	}

	b := model.Build{
		UserID:   u.ID,
		Manifest: f.Manifest,
	}

	if f.Namespace != "" {
		n, err := u.FindOrCreateNamespace(f.Namespace)

		if err != nil {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		b.NamespaceID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}
	}

	if err := b.Create(); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	// Passed form validation, so the YAML is valid.
	manifest, _ := config.DecodeManifest(strings.NewReader(f.Manifest))

	// Create initial setup stage. Will contain the output of driver creation
	// and cloning of source repositories.
	setupStage := model.Stage{
		BuildID: b.ID,
		Name:    fmt.Sprintf("setup - #%v", b.ID),
	}

	if err := setupStage.Create(); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	createJob := model.Job{
		BuildID: b.ID,
		StageID: setupStage.ID,
		Name:    "create driver",
	}

	if err := createJob.Create(); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	for i := range manifest.Sources {
		name := fmt.Sprintf("clone.%d", i + 1)

		j := model.Job{
			BuildID: b.ID,
			StageID: setupStage.ID,
			Name:    name,
		}

		if err := j.Create(); err != nil {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	for _, name := range manifest.Stages {
		canFail := false

		for _, allowed := range manifest.AllowFailures {
			if name == allowed {
				canFail = true
				break
			}
		}

		s := model.Stage{
			BuildID: b.ID,
			Name:    name,
			CanFail: canFail,
		}

		if err := s.Create(); err != nil {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		jobId := 1

		for _, manifestJob := range manifest.Jobs {
			if manifestJob.Stage != s.Name {
				continue
			}

			if manifestJob.Name == "" {
				manifestJob.Name = fmt.Sprintf("%s.%d", manifestJob.Stage, jobId)
			}

			j := model.Job{
				BuildID: b.ID,
				StageID: s.ID,
				Name:    manifestJob.Name,
			}

			if err := j.Create(); err != nil {
				log.Error.Println(errors.Err(err))
				HTMLError(w, "Something went wrong", http.StatusInternalServerError)
				return
			}
		}
	}

	tags := make([]model.Tag, len(f.Tags), len(f.Tags))

	for i, name := range f.Tags {
		if name == "" {
			continue
		}

		tags[i] = model.Tag{
			UserID:  u.ID,
			BuildID: b.ID,
			Name:    strings.TrimSpace(name),
		}

		if err := tags[i].Create(); err != nil {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	if _, err := queue.SendTask(queue.Builds, b.Signature()); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h Build) Show(w http.ResponseWriter, r *http.Request) {
	u, err := h.userFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)

	id, err := strconv.ParseInt(vars["build"], 10, 64)

	if err != nil {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	b, err := u.FindBuild(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if b.IsZero() || u.ID != b.UserID {
		HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	p := &build.ShowPage{
		Page: &template.Page{
			URI: r.URL.Path,
		},
		Build: b,
	}

	if filepath.Base(r.URL.Path) == "manifest" {
		mp := &build.ShowManifestPage{
			ShowPage: p,
		}

		d := template.NewDashboard(mp, r.URL.Path)

		HTML(w, template.Render(d), http.StatusOK)
		return
	}

	if filepath.Base(r.URL.Path) == "raw" {
		Text(w, b.Manifest, http.StatusOK)
		return
	}

	if err := b.LoadRelations(); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := model.LoadStageJobs(b.Stages); err != nil {
		log.Error.Println(errors.Err(err))
		HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	d := template.NewDashboard(p, r.URL.Path)

	HTML(w, template.Render(d), http.StatusOK)
}
