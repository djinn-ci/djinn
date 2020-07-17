package handler

import (
	"net/http"
	"net/url"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"
)

type Namespace struct {
	web.Handler

	Loaders    *database.Loaders
	Builds     *build.Store
	Namespaces *namespace.Store
}

func (h Namespace) loadParent(n *namespace.Namespace) error {
	if !n.ParentID.Valid {
		return nil
	}

	p, err := h.Namespaces.Get(query.Where("id", "=", n.ParentID))

	if err != nil {
		return nil
	}

	n.Parent = p

	err = h.Users.Load("id", []interface{}{n.Parent.UserID}, database.Bind("user_id", "id", n.Parent))
	return errors.Err(err)
}

func (h Namespace) IndexWithRelations(s *namespace.Store, vals url.Values) ([]*namespace.Namespace, database.Paginator, error) {
	nn, paginator, err := s.Index(vals)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	mm := database.ModelSlice(len(nn), namespace.Model(nn))

	if err := h.Users.Load("id", database.MapKey("user_id", mm), database.Bind("user_id", "id", mm...)); err != nil {
		return nn, paginator, errors.Err(err)
	}

	if err := h.Namespaces.Load("id", database.MapKey("parent_id", mm), database.Bind("parent_id", "id", mm...)); err != nil {
		return nn, paginator, errors.Err(err)
	}

	bb, err := h.Builds.All(
		query.Where("namespace_id", "IN", database.MapKey("id", mm)...),
		query.OrderDesc("created_at"),
	)

	if err != nil {
		return nn, paginator, errors.Err(err)
	}

	if err := build.LoadRelations(h.Loaders, bb...); err != nil {
		return nn, paginator, errors.Err(err)
	}

	m := make(map[int64]*build.Build)

	for _, b := range bb {
		if b.NamespaceID.Valid {
			m[b.NamespaceID.Int64] = b
		}
	}

	for _, n := range nn {
		if b, ok := m[n.ID]; ok {
			n.Build = b
		}
	}
	return nn, paginator, errors.Err(err)
}

func (h Namespace) StoreModel(r *http.Request) (*namespace.Namespace, namespace.Form, error) {
	f := namespace.Form{}

	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	namespaces := namespace.NewStore(h.DB, u)

	f.Namespaces = namespaces

	if err := form.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	n, err := namespaces.Create(f.Parent, f.Name, f.Description, f.Visibility)
	return n, f, errors.Err(err)
}

func (h Namespace) UpdateModel(r *http.Request) (*namespace.Namespace, namespace.Form, error) {
	ctx := r.Context()

	f := namespace.Form{}

	u, ok := user.FromContext(ctx)

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	namespaces := namespace.NewStore(h.DB, u)

	f.Namespaces = namespaces

	n, ok := namespace.FromContext(ctx)

	if !ok {
		return nil, f, errors.New("no namespace in request context")
	}

	f.Namespace = n

	if err := form.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	err := namespaces.Update(n.ID, f.Name, f.Description, f.Visibility)
	return n, f, errors.Err(err)
}

func (h Namespace) DeleteModel(r *http.Request) error {
	n, ok := namespace.FromContext(r.Context())

	if !ok {
		return errors.New("no namespace in request context")
	}
	return errors.Err(h.Namespaces.Delete(n.ID))
}
