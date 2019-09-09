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
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
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

	return func(u *model.User, r *http.Request) bool {
		vars := mux.Vars(r)

		owner, err := users.FindByUsername(vars["username"])

		if err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		r = r.WithContext(context.WithValue(r.Context(), "owner", owner))

		id, _ := strconv.ParseInt(vars["build"], 10, 64)

		b, err := owner.BuildStore().Find(id)

		if err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		if b.IsZero() {
			return false
		}

		r = r.WithContext(context.WithValue(r.Context(), "build", b))

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

	return func(u *model.User, r *http.Request) bool {
		vars := mux.Vars(r)

		owner, err := users.FindByUsername(vars["username"])

		if err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		r = r.WithContext(context.WithValue(r.Context(), "owner", owner))

		path := strings.TrimSuffix(vars["namespace"], "/")

		n, err := owner.NamespaceStore().FindByPath(path)

		if err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		if n.IsZero() {
			return false
		}

		r = r.WithContext(context.WithValue(r.Context(), "namespace", n))

		if filepath.Base(r.URL.Path) == "edit" {
			return u.ID == n.UserID
		}

		if r.Method == "DELETE" || r.Method == "PATCH" {
			return u.ID == n.UserID
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

	return func(u *model.User, r *http.Request) bool {
		vars := mux.Vars(r)

		id, _ := strconv.ParseInt(vars[name], 10, 64)

		row := model.NewRow()

		q := query.Select(
			query.Columns("*"),
			query.From(resources[name]),
			query.Where("id", "=", id),
		)

		stmt, err := g.store.Preparex(q.Build())

		if err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		defer stmt.Close()

		if err := stmt.QueryRowx(q.Args()...).MapScan(row); err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		namespaceId, ok := row["namespace_id"].(int64)

		if !ok {
			userId, ok := row["user_id"].(int64)

			if !ok {
				return false
			}

			return userId == u.ID
		}

		root, err := namespaces.FindRoot(namespaceId)

		if err != nil {
			log.Error.Println(errors.Err(err))
			return false
		}

		return root.AccessibleBy(u)
	}
}
