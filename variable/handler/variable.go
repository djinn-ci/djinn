package handler

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/variable"
	"github.com/andrewpillar/thrall/web"
)

type Variable struct {
	web.Handler

	Loaders   *database.Loaders
	Variables *variable.Store
}

func (h Variable) IndexWithRelations(r *http.Request) ([]*variable.Variable, database.Paginator, error) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, database.Paginator{}, errors.New("no user in request context")
	}

	vv, paginator, err := variable.NewStore(h.DB, u).Index(r.URL.Query())

	if err != nil {
		return vv, paginator, errors.Err(err)
	}

	if err := variable.LoadRelations(h.Loaders, vv...); err != nil {
		return vv, paginator, errors.Err(err)
	}

	nn := make([]database.Model, 0, len(vv))

	for _, v := range vv {
		if v.Namespace != nil {
			nn = append(nn, v.Namespace)
		}
	}

	err = h.Users.Load("id", database.MapKey("user_id", nn), database.Bind("user_id", "id", nn...))
	return vv, paginator, errors.Err(err)
}

func (h Variable) StoreModel(r *http.Request) (*variable.Variable, variable.Form, error) {
	f := variable.Form{}

	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	variables := variable.NewStore(h.DB, u)

	f.Resource = namespace.Resource{
		User:       u,
		Namespaces: namespace.NewStore(h.DB, u),
	}
	f.Variables = variables

	if err := form.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	v, err := variables.Create(f.Key, f.Value)
	return v, f, errors.Err(err)
}

func (h Variable) DeleteModel(r *http.Request) error {
	v, ok := variable.FromContext(r.Context())

	if !ok {
		return errors.New("no variable in request context")
	}
	return errors.Err(h.Variables.Delete(v.ID))
}
