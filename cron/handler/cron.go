package handler

import (
	"net/http"
	"net/url"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/cron"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"
)

// Cron is the base handler that provides shared logic for the UI and API
// handlers for cron job creation, and management.
type Cron struct {
	web.Handler

	// Loaders are the relationship loaders to use for loading the
	// relationships we need when working with cron jobs.
	Loaders *database.Loaders

	// Crons is the store used for deletion of cron jobs.
	Crons *cron.Store

	// Builds is the store used for retrieving any builds that were submitted
	// via a scheduled cron job.
	Builds *build.Store
}

// IndexWithRelations retrieves a slice of *cron.Cron models for the user in
// the given request context. All of the relations for each cron will be loaded
// into each model we have. If any of the crons have a bound namespace, then
// the namespace's user will be loaded too. A database.Paginator will also be
// returned if there are multiple pages of cron jobs.
func (h Cron) IndexWithRelations(s *cron.Store, vals url.Values) ([]*cron.Cron, database.Paginator, error) {
	cc, paginator, err := s.Index(vals)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	if err := cron.LoadRelations(h.Loaders, cc...); err != nil {
		return nil, paginator, errors.Err(err)
	}

	nn := make([]database.Model, 0, len(cc))

	for _, c := range cc {
		if c.Namespace != nil {
			nn = append(nn, c.Namespace)
		}
	}

	err = h.Users.Load("id", database.MapKey("user_id", nn), database.Bind("user_id", "id", nn...))
	return cc, paginator, errors.Err(err)
}

// StoreModel unmarshals the request's data into a cron job, validates it and
// stores it in the database. Upon success this will return the newly created
// cron. This also returns the form for creating a cron job.
func (h Cron) StoreModel(r *http.Request) (*cron.Cron, cron.Form, error) {
	var f cron.Form

	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	crons := cron.NewStore(h.DB, u)

	f.Resource = namespace.Resource{
		User:       u,
		Namespaces: namespace.NewStore(h.DB, u),
	}
	f.Crons = crons

	if err := form.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	c, err := crons.Create(f.Name, f.Schedule, f.Manifest)
	return c, f, errors.Err(err)
}

// ShowWithRelations retrieves the *cron.Cron model from the context of the
// given request. All of the relations for the cron will be loaded into the
// model we have. If the cron has a namespace bound to it, then the
// namespace's user will be loaded to the namespace.
func (h Cron) ShowWithRelations(r *http.Request) (*cron.Cron, error) {
	var err error

	c, ok := cron.FromContext(r.Context())

	if !ok {
		return nil, errors.New("no cron in request context")
	}

	if err := cron.LoadRelations(h.Loaders, c); err != nil {
		return nil, errors.Err(err)
	}

	if c.Namespace != nil {
		err = h.Users.Load(
			"id", []interface{}{c.Namespace.Values()["user_id"]}, database.Bind("user_id", "id", c.Namespace),
		)
	}
	return c, errors.Err(err)
}

// UpdateModel unmarshals the request's data into a cron job, validates it and
// updates the existing cron job in the database with the content in the form.
// Upon success the updated cron job is returned. This also returns the form
// for modifying a cron job.
func (h Cron) UpdateModel(r *http.Request) (*cron.Cron, *cron.Form, error) {
	ctx := r.Context()
	f := &cron.Form{}

	u, ok := user.FromContext(ctx)

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	c, ok := cron.FromContext(ctx)

	if !ok {
		return nil, f, errors.New("no cron in request context")
	}

	crons := cron.NewStore(h.DB, u)

	f.Cron = c
	f.Crons = crons

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	if err := crons.Update(c.ID, f.Name, f.Schedule, f.Manifest); err != nil {
		return nil, f, errors.Err(err)
	}

	c.Name = f.Name
	c.Schedule = f.Schedule
	c.Manifest = f.Manifest
	return c, f, nil
}

// DeleteModel removes the cron in the given request context from the database.
func (h Cron) DeleteModel(r *http.Request) error {
	c, ok := cron.FromContext(r.Context())

	if !ok {
		return errors.New("no cron in request context")
	}
	return errors.Err(h.Crons.Delete(c.ID))
}
