package handler

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/variable"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/sessions"
)

type Variable struct {
	web.Handler

	Loaders    model.Loaders
	Namespaces *namespace.Store
	Variables  *variable.Store
}

func (h Variable) Model(r *http.Request) *variable.Variable {
	val := r.Context().Value("variable")
	v, _ := val.(*variable.Variable)
	return v
}

func (h Variable) IndexWithRelations(s *variable.Store, r *http.Request) ([]*variable.Variable, model.Paginator, error) {
	vv, paginator, err := s.Index(r.URL.Query())

	if err != nil {
		return vv, paginator, errors.Err(err)
	}

	if err := variable.LoadRelations(h.Loaders, vv...); err != nil {
		return vv, paginator, errors.Err(err)
	}

	nn := make([]model.Model, 0, len(vv))

	for _, v := range vv {
		if v.Namespace != nil {
			nn = append(nn, v.Namespace)
		}
	}

	err = h.Users.Load("id", model.MapKey("user_id", nn), model.Bind("user_id", "id", nn...))
	return vv, paginator, errors.Err(err)
}

func (h Variable) StoreModel(r *http.Request, sess *sessions.Session) (*variable.Variable, error) {
	u := h.User(r)
	variables := variable.NewStore(h.DB, u)

	f := &variable.Form{
		Resource:  namespace.Resource{
			User:       u,
			Namespaces: namespace.NewStore(h.DB, u),
		},
		Variables: variables,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		return &variable.Variable{}, errors.Err(err)
	}

	if f.Namespace != "" {
		n, err := h.Namespaces.GetByPath(f.Namespace)

		if err != nil {
			return &variable.Variable{}, errors.Err(err)
		}

		if !n.CanAdd(u) {
			return &variable.Variable{}, namespace.ErrPermission
		}
		variables.Bind(n)
	}

	v := variables.New()
	v.Key = f.Key
	v.Value = f.Value

	err := h.Variables.Create(v)
	return v, errors.Err(err)
}
