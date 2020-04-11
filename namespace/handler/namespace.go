package handler

import (
	"database/sql"
	"net/http"
	"net/url"
	"strings"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/sessions"
)

type Namespace struct {
	web.Handler

	Loaders    model.Loaders
	Builds     build.Store
	Namespaces namespace.Store
}

func (h Namespace) Model(r *http.Request) *namespace.Namespace {
	val := r.Context().Value("namespace")
	n, _ := val.(*namespace.Namespace)
	return n
}

func (h Namespace) Delete(r *http.Request) error {
	n := h.Model(r)
	err := h.Namespaces.Delete(n)
	return errors.Err(err)
}

func (h Namespace) IndexWithRelations(s namespace.Store, vals url.Values) ([]*namespace.Namespace, model.Paginator, error) {
	nn, paginator, err := s.Index(vals)

	if err != nil {
		return []*namespace.Namespace{}, paginator, errors.Err(err)
	}

	mm := model.Slice(len(nn), namespace.Model(nn))

	if err := h.Users.Load("id", model.MapKey("user_id", mm), model.Bind("user_id", "id", mm...)); err != nil {
		return nn, paginator, errors.Err(err)
	}

	bb, err := h.Builds.All(
		query.Where("namespace_id", "IN", model.MapKey("id", mm)...),
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

func (h Namespace) StoreModel(r *http.Request, sess *sessions.Session) (*namespace.Namespace, error) {
	u := h.User(r)

	namespaces := namespace.NewStore(h.DB, u)

	f := &namespace.Form{
		Namespaces: namespaces,
		UserID:     u.ID,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		return &namespace.Namespace{}, errors.Err(err)
	}

	parent, err := namespaces.Get(query.Where("path", "=", f.Parent))

	if err != nil {
		return &namespace.Namespace{}, errors.Err(err)
	}

	n := namespaces.New()
	n.Name = f.Name
	n.Path = f.Name
	n.Description = f.Description
	n.Visibility = f.Visibility

	if !parent.IsZero() {
		n.RootID = parent.RootID
		n.ParentID = sql.NullInt64{
			Int64: parent.ID,
			Valid: true,
		}
		n.Path = strings.Join([]string{parent.Path, f.Name}, "/")
		n.Level = parent.Level + 1
		n.Visibility = parent.Visibility
	}

	if n.Level >= namespace.MaxDepth {
		return n, namespace.ErrDepth
	}

	if err := namespaces.Create(n); err != nil {
		return n, errors.Err(err)
	}

	if parent.IsZero() {
		n.RootID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}

		err := namespaces.Update(n)
		return n, errors.Err(err)
	}
	return n, nil
}

func (h Namespace) Get(r *http.Request) (*namespace.Namespace, error) {
	n := h.Model(r)

	err := h.Users.Load("id", []interface{}{n.UserID}, model.Bind("user_id", "id", n))
	return n, errors.Err(err)
}

func (h Namespace) UpdateModel(r *http.Request, sess *sessions.Session) (*namespace.Namespace, error) {
	u := h.User(r)
	n := h.Model(r)

	f := &namespace.Form{
		Namespaces: namespace.NewStore(h.DB, u),
		Namespace:  n,
		UserID:     n.UserID,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		return n, errors.Err(err)
	}

	if n.ParentID.Valid {
		parent, err := h.Namespaces.Get(query.Where("id", "=", n.ParentID))

		if err != nil {
			return n, errors.Err(err)
		}
		n.Parent = parent
	}

	n.Name = f.Name
	n.Description = f.Description
	n.Visibility = f.Visibility

	if !n.Parent.IsZero() {
		n.Visibility = n.Parent.Visibility
	} else {
		if err := n.CascadeVisibility(h.DB); err != nil {
			return n, errors.Err(err)
		}
	}
	err := h.Namespaces.Update(n)
	return n, errors.Err(err)
}
