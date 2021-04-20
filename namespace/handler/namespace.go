package handler

import (
	"net/http"
	"net/url"

	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/user"
	"djinn-ci.com/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

// Namespace is the base handler that provides shared logic for the UI and API
// handlers for namespace creation, and management.
type Namespace struct {
	web.Handler

	loaders *database.Loaders // used for loading in build relations on a namespace
}

func New(h web.Handler) Namespace {
	loaders := database.NewLoaders()
	loaders.Put("user", h.Users)
	loaders.Put("build_tag", build.NewTagStore(h.DB))
	loaders.Put("build_trigger", build.NewTriggerStore(h.DB))

	return Namespace{
		Handler: h,
		loaders: loaders,
	}
}

// loadParent loads the immediate parent for the given namespace.
func (h Namespace) loadParent(n *namespace.Namespace) error {
	if !n.ParentID.Valid {
		return nil
	}

	p, err := namespace.NewStore(h.DB).Get(query.Where("id", "=", query.Arg(n.ParentID)))

	if err != nil {
		return nil
	}

	n.Parent = p

	err = h.Users.Load("id", []interface{}{n.Parent.UserID}, database.Bind("user_id", "id", n.Parent))
	return errors.Err(err)
}

// IndexWithRelations retreives a slice of *namespace.Namespace models for the
// user in the given request context. All of the relations for each namespace
// will be loaded into each model we have. A database.Paginator will also be
// returned if there are multiple pages of namespaces.
func (h Namespace) IndexWithRelations(s *namespace.Store, vals url.Values) ([]*namespace.Namespace, database.Paginator, error) {
	nn, paginator, err := s.Index(vals)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	mm := database.ModelSlice(len(nn), namespace.Model(nn))

	if err := h.Users.Load("id", database.MapKey("user_id", mm), database.Bind("user_id", "id", mm...)); err != nil {
		return nn, paginator, errors.Err(err)
	}

	if err := namespace.NewStore(h.DB).Load("id", database.MapKey("parent_id", mm), database.Bind("parent_id", "id", mm...)); err != nil {
		return nn, paginator, errors.Err(err)
	}

	bb, err := build.NewStore(h.DB).All(
		query.Where("namespace_id", "IN", database.List(database.MapKey("id", mm)...)),
		query.OrderDesc("created_at"),
	)

	if err != nil {
		return nn, paginator, errors.Err(err)
	}

	if err := build.LoadRelations(h.loaders, bb...); err != nil {
		return nn, paginator, errors.Err(err)
	}

	m := make(map[int64]*build.Build)

	for _, b := range bb {
		if b.NamespaceID.Valid {
			if _, ok := m[b.NamespaceID.Int64]; !ok {
				m[b.NamespaceID.Int64] = b
			}
		}
	}

	for _, n := range nn {
		if b, ok := m[n.ID]; ok && n.Build == nil {
			n.Build = b
		}
	}
	return nn, paginator, errors.Err(err)
}

// StoreModel unmarshals the request's data into a namespace, validates it and
// stores it in the database. Upon success this will return the newly created
// namespace. This also returns the form for creating a namespace.
func (h Namespace) StoreModel(r *http.Request) (*namespace.Namespace, namespace.Form, error) {
	f := namespace.Form{}

	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	namespaces := namespace.NewStore(h.DB, u)

	f.Namespaces = namespaces

	if err := webutil.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	n, err := namespaces.Create(f.Parent, f.Name, f.Description, f.Visibility)
	return n, f, errors.Err(err)
}

// UpdateModel unmarshals the request's data into a namespace, validates it and
// updates the existing namespace in the database with the content in the form.
// Upon success the updated namespace is returned. This also returns the form
// for modifying a namespace.
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

	if err := webutil.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	err := namespaces.Update(n.ID, f.Description, f.Visibility)
	return n, f, errors.Err(err)
}

// DeleteModel removes the namespace and all of its children in the given
// request context from the database.
func (h Namespace) DeleteModel(r *http.Request) error {
	n, ok := namespace.FromContext(r.Context())

	if !ok {
		return errors.New("no namespace in request context")
	}
	return errors.Err(namespace.NewStore(h.DB).Delete(n.ID))
}
