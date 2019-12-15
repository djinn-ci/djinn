package server

import (
	"context"
	"database/sql"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type load func(int64) (model.Interface, error)

type gate struct {
	users      model.UserStore
	namespaces model.NamespaceStore
	images     model.ImageStore
	objects    model.ObjectStore
	variables  model.VariableStore
	keys       model.KeyStore
}

func namespaceRoot(id int64) query.Query {
	return query.Select(
		query.Columns("root_id"),
		query.From(model.NamespaceTable),
		query.Where("id", "=", id),
	)
}

func (g gate) build(u *model.User, r *http.Request) (*http.Request, bool) {
	vars := mux.Vars(r)

	owner, err := g.users.Get(query.Where("username", "=", vars["username"]))

	if err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	r = r.WithContext(context.WithValue(r.Context(), "owner", owner))

	id, _ := strconv.ParseInt(vars["build"], 10, 64)

	b, err := owner.BuildStore().Get(query.Where("id", "=", id))

	if err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	if b.IsZero() {
		return r, false
	}

	r = r.WithContext(context.WithValue(r.Context(), "build", b))

	if !b.NamespaceID.Valid {
		return r, u.ID == b.UserID
	}

	root, err := g.namespaces.Get(
		query.WhereQuery("root_id", "=", namespaceRoot(b.NamespaceID.Int64)),
		query.WhereQuery("id", "=", namespaceRoot(b.NamespaceID.Int64)),
	)

	if err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	if err := root.LoadCollaborators(); err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	return r, root.AccessibleBy(u)
}

func (g gate) key(u *model.User, r *http.Request) (*http.Request, bool) {
	var (
		k   *model.Key
		err error
	)

	load := func(id int64) (model.Interface, error) {
		k, err = g.keys.Get(query.Where("id", "=", id))

		return k, errors.Err(err)
	}

	ok, err := g.resource("key", u, r, load)

	if err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	r = r.WithContext(context.WithValue(r.Context(), "key", k))

	return r, ok
}

func (g gate) namespace(u *model.User, r *http.Request) (*http.Request, bool) {
	vars := mux.Vars(r)

	owner, err := g.users.Get(query.Where("username", "=", vars["username"]))

	if err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	r = r.WithContext(context.WithValue(r.Context(), "owner", owner))

	path := strings.TrimSuffix(vars["namespace"], "/")

	n, err := owner.NamespaceStore().Get(query.Where("path", "=", path))

	if err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	if n.IsZero() {
		return r, false
	}

	r = r.WithContext(context.WithValue(r.Context(), "namespace", n))

	if filepath.Base(r.URL.Path) == "edit" {
		return r, u.ID == n.UserID
	}

	if r.Method == "DELETE" || r.Method == "PATCH" {
		return r, u.ID == n.UserID
	}

	root, err := g.namespaces.Get(
		query.WhereQuery("root_id", "=", namespaceRoot(n.ID)),
		query.WhereQuery("id", "=", namespaceRoot(n.ID)),
	)

	if err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	if err := root.LoadCollaborators(); err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	return r, root.AccessibleBy(u)
}

func (g gate) image(u *model.User, r *http.Request) (*http.Request, bool) {
	var (
		i   *model.Image
		err error
	)

	load := func(id int64) (model.Interface, error) {
		i, err = g.images.Get(query.Where("id", "=", id))

		return i, errors.Err(err)
	}

	ok, err := g.resource("image", u, r, load)

	if err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	r = r.WithContext(context.WithValue(r.Context(), "image", i))

	return r, ok
}

func (g gate) object(u *model.User, r *http.Request) (*http.Request, bool) {
	var (
		o   *model.Object
		err error
	)

	load := func(id int64) (model.Interface, error) {
		o, err = g.objects.Get(query.Where("id", "=", id))

		return o, errors.Err(err)
	}

	ok, err := g.resource("object", u, r, load)

	if err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	r = r.WithContext(context.WithValue(r.Context(), "object", o))

	return r, ok
}

func (g gate) resource(name string, u *model.User, r *http.Request, fn load) (bool, error) {
	base := filepath.Base(r.URL.Path)

	if base == name + "s" || base == "create" {
		return !u.IsZero(), nil
	}

	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars[name], 10, 64)

	m, err := fn(id)

	if err != nil {
		return false, errors.Err(err)
	}

	namespaceId, _ := m.Values()["namespace_id"].(sql.NullInt64)

	if !namespaceId.Valid {
		userId, ok := m.Values()["user_id"].(int64)

		if !ok {
			return false, nil
		}

		return u.ID == userId, nil
	}

	root, err := g.namespaces.Get(
		query.WhereQuery("root_id", "=", namespaceRoot(namespaceId.Int64)),
		query.WhereQuery("id", "=", namespaceRoot(namespaceId.Int64)),
	)

	if err != nil {
		return false, errors.Err(err)
	}

	if err := root.LoadCollaborators(); err != nil {
		log.Error.Println(errors.Err(err))
		return false, errors.Err(err)
	}

	return root.AccessibleBy(u), nil
}

func (g gate) variable(u *model.User, r *http.Request) (*http.Request, bool) {
	var (
		v   *model.Variable
		err error
	)

	load := func(id int64) (model.Interface, error) {
		v, err = g.variables.Get(query.Where("id", "=", id))

		return v, errors.Err(err)
	}

	ok, err := g.resource("variable", u, r, load)

	if err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	r = r.WithContext(context.WithValue(r.Context(), "variable", v))

	return r, ok
}
