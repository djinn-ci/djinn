package main

import (
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"
)

var resources map[string]string = map[string]string{
	"object":   model.ObjectTable,
	"variable": model.VariableTable,
	"key":      model.KeyTable,
}

type gate struct {
	store model.Store
}

func (g gate) build() web.Gate {
	users := model.UserStore{
		Store: g.store,
	}

	namespaces := model.NamespaceStore{
		Store: g.store,
	}

	return func(u *model.User, vars map[string]string) bool {
		owner, err := users.FindByUsername(vars["username"])

		if err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		id, _ := strconv.ParseInt(vars["build"], 10, 64)

		b, err := owner.BuildStore().Find(id)

		if err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		if b.IsZero() {
			return false
		}

		if !b.NamespaceID.Valid {
			return u.ID == b.UserID
		}

		root, err := namespaces.FindRoot(b.NamespaceID.Int64)

		if err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		return root.AccessibleBy(u)
	}
}

func (g gate) namespace() web.Gate {
	users := model.UserStore{
		Store: g.store,
	}

	namespaces := model.NamespaceStore{
		Store: g.store,
	}

	return func(u *model.User, vars map[string]string) bool {
		owner, err := users.FindByUsername(vars["username"])

		if err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		path := strings.TrimSuffix(vars["namespace"], "/")

		n, err := owner.NamespaceStore().FindByPath(path)

		if err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		if n.IsZero() {
			return false
		}

		root, err := namespaces.FindRoot(n.ID)

		if err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		return root.AccessibleBy(u)
	}
}

func (g gate) resource(name string) web.Gate {
	namespaces := model.NamespaceStore{
		Store: g.store,
	}

	return func(u *model.User, vars map[string]string) bool {
		id, _ := strconv.ParseInt(vars[name], 10, 64)

		r := model.NewRow()

		if err := g.store.FindBy(&r, resources[name], "id", id); err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		val, ok := r["namespace_id"]

		if !ok {
			return false
		}

		namespaceId, ok := val.(int64)

		if !ok {
			return false
		}

		root, err := namespaces.FindRoot(namespaceId)

		if err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		if root.IsZero() {
			userId, ok := r["user_id"].(int64)

			if !ok {
				return false
			}

			return userId == u.ID
		}

		return root.AccessibleBy(u)
	}
}