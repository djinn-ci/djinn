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

type Job struct {
	web.Handler

	Loaders *database.Loaders
}

// IndexWithRelations returns all of the jobs with their relationships loaded
// into each return job.
func (h Job) IndexWithRelations(s *build.JobStore, vals url.Values) ([]*build.Job, error) {
	jj, err := s.Index(vals)

	if err != nil {
		return jj, errors.Err(err)
	}

	err = build.LoadJobRelations(h.Loaders, jj...)
	return jj, errors.Err(err)
}

// ShowWithRelations returns a job with all of the relations loaded for that
// job.
func (h Job) ShowWithRelations(r *http.Request) (*build.Job, error) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		return nil, errors.New("failed to get build from request context")
	}

	if err := build.LoadRelations(h.Loaders, b); err != nil {
		return nil, errors.Err(err)
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["job"], 10, 64)

	j, err := build.NewJobStore(h.DB, b).Get(query.Where("id", "=", id))

	if err != nil {
		return j, errors.Err(err)
	}

	err = build.LoadJobRelations(h.Loaders, j)
	return j, errors.Err(err)
}
