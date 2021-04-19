package handler

import (
	"net/http"
	"net/url"

	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

// Job is the base handler that provides shared logic for the UI and API
// handlers for working with build jobs.
type Job struct {
	web.Handler

	loaders *database.Loaders
}

func NewJob(h web.Handler) Job {
	loaders := database.NewLoaders()
	loaders.Put("build_stage", build.NewStageStore(h.DB))
	loaders.Put("build_artifact", build.NewArtifactStore(h.DB))
	loaders.Put("build_trigger", build.NewTriggerStore(h.DB))

	return Job{
		Handler: h,
		loaders: loaders,
	}
}

// IndexWithRelations returns all of the jobs with their relationships loaded
// into each return job.
func (h Job) IndexWithRelations(s *build.JobStore, vals url.Values) ([]*build.Job, error) {
	jj, err := s.Index(vals)

	if err != nil {
		return jj, errors.Err(err)
	}

	err = build.LoadJobRelations(h.loaders, jj...)
	return jj, errors.Err(err)
}

// ShowWithRelations returns a job with all of the relations loaded for that
// job.
func (h Job) ShowWithRelations(r *http.Request) (*build.Job, error) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		return nil, errors.New("failed to get build from request context")
	}

	if err := build.LoadRelations(h.loaders, b); err != nil {
		return nil, errors.Err(err)
	}

	name := mux.Vars(r)["name"]

	j, err := build.NewJobStore(h.DB, b).Get(query.Where("name", "=", query.Arg(name)))

	if err != nil {
		return j, errors.Err(err)
	}

	err = build.LoadJobRelations(h.loaders, j)
	return j, errors.Err(err)
}
