package handler

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/web"

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

	id, _ := strconv.ParseInt(mux.Vars(r)["job"], 10, 64)

	j, err := build.NewJobStore(h.DB, b).Get(query.Where("id", "=", query.Arg(id)))

	if err != nil {
		return j, errors.Err(err)
	}

	err = build.LoadJobRelations(h.loaders, j)
	return j, errors.Err(err)
}
