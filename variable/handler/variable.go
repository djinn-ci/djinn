package handler

import (
	"net/http"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/user"
	"djinn-ci.com/variable"
	"djinn-ci.com/web"

	"github.com/andrewpillar/webutil"
)

// Variable is the base handler that provides shared logic for the UI and API
// handlers for variable creation, and management.
type Variable struct {
	web.Handler

	loaders *database.Loaders
}

func New(h web.Handler) Variable {
	loaders := database.NewLoaders()
	loaders.Put("author", h.Users)
	loaders.Put("user", h.Users)
	loaders.Put("namespace", namespace.NewStore(h.DB))

	return Variable{
		Handler: h,
		loaders: loaders,
	}
}

// IndexWithRelations retrieves a slice of *variable.Variable models for the
// user in the given request context. All of the relations for each variable
// will be loaded into each model we have. If any of the keys have a bound
// namespace, then the namespace's user will be loaded too. A
// database.Paginator will also be returned if there are multiple pages of
// variables.
func (h Variable) IndexWithRelations(r *http.Request) ([]*variable.Variable, database.Paginator, error) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, database.Paginator{}, errors.New("no user in request context")
	}

	vv, paginator, err := variable.NewStore(h.DB, u).Index(r.URL.Query())

	if err != nil {
		return vv, paginator, errors.Err(err)
	}

	if err := variable.LoadRelations(h.loaders, vv...); err != nil {
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

// StoreModel unmarshals the request's data into a variable, validates it and
// stores it in the database. Upon success this will return the newly created
// variable. This also returns the form for creating a variable.
func (h Variable) StoreModel(r *http.Request) (*variable.Variable, variable.Form, error) {
	f := variable.Form{}

	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	variables := variable.NewStore(h.DB, u)

	f.Resource = namespace.Resource{
		Author:     u,
		Namespaces: namespace.NewStore(h.DB, u),
	}
	f.Variables = variables

	if err := webutil.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	v, err := variables.Create(f.Author.ID, f.Key, f.Value)
	return v, f, errors.Err(err)
}

// DeleteModel removes the variable in the given request context from the
// database.
func (h Variable) DeleteModel(r *http.Request) error {
	v, ok := variable.FromContext(r.Context())

	if !ok {
		return errors.New("no variable in request context")
	}
	return errors.Err(variable.NewStore(h.DB).Delete(v.ID))
}
