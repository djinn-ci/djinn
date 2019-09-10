package main

import (
	"context"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"

	"github.com/gorilla/mux"
)

type load func(int64) (model.Interface, error)

type gate struct {
	users      model.UserStore
	namespaces model.NamespaceStore
	objects    model.ObjectStore
	variables  model.VariableStore
	keys       model.KeyStore
}

func (g gate) build(u *model.User, r *http.Request) (*http.Request, bool) {
	vars := mux.Vars(r)

	owner, err := g.users.FindByUsername(vars["username"])

	if err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	r = r.WithContext(context.WithValue(r.Context(), "owner", owner))

	id, _ := strconv.ParseInt(vars["build"], 10, 64)

	b, err := owner.BuildStore().Find(id)

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

	root, err := g.namespaces.FindRoot(b.NamespaceID.Int64)

	if err != nil {
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
		k, err = g.keys.Find(id)

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

	owner, err := g.users.FindByUsername(vars["username"])

	if err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	r = r.WithContext(context.WithValue(r.Context(), "owner", owner))

	path := strings.TrimSuffix(vars["namespace"], "/")

	n, err := owner.NamespaceStore().FindByPath(path)

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

	root, err := g.namespaces.FindRoot(n.ID)

	if err != nil {
		log.Error.Println(errors.Err(err))
		return r, false
	}

	return r, root.AccessibleBy(u)
}

func (g gate) object(u *model.User, r *http.Request) (*http.Request, bool) {
	var (
		o   *model.Object
		err error
	)

	load := func(id int64) (model.Interface, error) {
		o, err = g.objects.Find(id)

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
	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars[name], 10, 64)

	m, err := fn(id)

	if err != nil {
		return false, errors.Err(err)
	}

	namespaceId, ok := m.Values()["namespace_id"].(int64)

	if !ok {
		userId, ok := m.Values()["user_id"].(int64)

		if !ok {
			return false, nil
		}

		return u.ID == userId, nil
	}

	root, err := g.namespaces.FindRoot(namespaceId)

	if err != nil {
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
		v, err = g.variables.Find(id)

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
