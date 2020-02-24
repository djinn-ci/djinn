package core

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/sessions"
)

type Variable struct {
	web.Handler

	Namespaces model.NamespaceStore
	Variables  model.VariableStore
}

func (h Variable) Variable(r *http.Request) *model.Variable {
	val := r.Context().Value("variable")
	v, _ := val.(*model.Variable)
	return v
}

func (h Variable) Index(variables model.VariableStore, r *http.Request, opts ...query.Option) ([]*model.Variable, model.Paginator, error) {
	index := []query.Option{
		model.Search("name", r.URL.Query().Get("search")),
	}

	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	paginator, err := variables.Paginate(page, append(index, opts...)...)

	if err != nil {
		return []*model.Variable{}, paginator, errors.Err(err)
	}

	vv, err := variables.All(append(index, opts...)...)

	if err != nil {
		return vv, paginator, errors.Err(err)
	}

	if err := variables.LoadNamespaces(vv); err != nil {
		return vv, paginator, errors.Err(err)
	}

	nn := make([]*model.Namespace, 0, len(vv))

	for _, v := range vv {
		if v.Namespace != nil {
			nn = append(nn, v.Namespace)
		}
	}

	err = h.Namespaces.LoadUsers(nn)

	return vv, paginator, errors.Err(err)
}

func (h Variable) Store(r *http.Request, sess *sessions.Session) (*model.Variable, error) {
	u := h.User(r)

	variables := u.VariableStore()

	f := &form.Variable{
		User:      u,
		Variables: variables,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); ok {
			return &model.Variable{}, form.ErrValidation
		}
		return &model.Variable{}, errors.Err(err)
	}

	n, err := h.Namespaces.Get(query.Where("path", "=", f.Namespace))

	if err != nil {
		return &model.Variable{}, errors.Err(err)
	}

	v := variables.New()
	v.NamespaceID = sql.NullInt64{
		Int64: n.ID,
		Valid: n.ID > 0,
	}
	v.Key = f.Key
	v.Value = f.Value

	err = h.Variables.Create(v)

	return v, errors.Err(err)
}

func (h Variable) Destroy(r *http.Request) error {
	v := h.Variable(r)

	err := h.Variables.Delete(v)

	return errors.Err(err)
}
