package core

import (
	"database/sql"
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"
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

func (h Variable) Index(variables model.VariableStore, r *http.Request, opts ...query.Option) ([]*model.Variable, error) {
	index := []query.Option{
		model.Search("name", r.URL.Query().Get("search")),
	}

	vv, err := variables.All(append(index, opts...)...)

	if err != nil {
		return vv, errors.Err(err)
	}

	if err := variables.LoadNamespaces(vv); err != nil {
		return vv, errors.Err(err)
	}

	nn := make([]*model.Namespace, 0, len(vv))

	for _, v := range vv {
		if v.Namespace != nil {
			nn = append(nn, v.Namespace)
		}
	}

	err = h.Namespaces.LoadUsers(nn)

	return vv, errors.Err(err)
}

func (h Variable) Store(w http.ResponseWriter, r *http.Request) (*model.Variable, error) {
	u := h.User(r)

	variables := u.VariableStore()
	namespaces := u.NamespaceStore()

	f := &form.Variable{
		Variables: variables,
	}

	if err := form.Unmarshal(f, r); err != nil {
		return &model.Variable{}, errors.Err(err)
	}

	var err error

	n := &model.Namespace{}

	if f.Namespace != "" {
		n, err = namespaces.FindOrCreate(f.Namespace)

		if err != nil {
			return &model.Variable{}, errors.Err(err)
		}

		f.Variables = n.VariableStore()
	}

	if err := f.Validate(); err != nil {
		if ferr, ok := err.(form.Errors); ok {
			h.FlashErrors(w, r, ferr)
			h.FlashForm(w, r, f)

			return &model.Variable{}, ErrValidationFailed
		}

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
