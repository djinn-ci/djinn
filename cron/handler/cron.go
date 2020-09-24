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

type Cron struct {
	web.Handler

	Loaders *database.Loaders
	Crons   *cron.Store
	Builds  *build.Store
}

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

func (h Cron) StoreModel(r *http.Request) (*cron.Cron, *cron.Form, error) {
	f := &cron.Form{}

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

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	c, err := crons.Create(f.Name, f.Schedule, f.Manifest)
	return c, f, errors.Err(err)
}

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
	return c, f, nil
}

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

func (h Cron) DeleteModel(r *http.Request) error {
	c, ok := cron.FromContext(r.Context())

	if !ok {
		return errors.New("no cron in request context")
	}
	return errors.Err(h.Crons.Delete(c.ID))
}
