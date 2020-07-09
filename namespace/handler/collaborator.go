package handler

import (
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type Collaborator struct {
	web.Handler
}

func (h Collaborator) DeleteModel(r *http.Request) error {
	vars := mux.Vars(r)

	owner, err := h.Users.Get(query.Where("username", "=", vars["username"]))

	if err != nil {
		return errors.Err(err)
	}

	if owner.IsZero() {
		return database.ErrNotFound
	}

	path := strings.TrimSuffix(vars["namespace"], "/")

	n, err := namespace.NewStore(h.DB, owner).Get(query.Where("path", "=", path))

	if err != nil {
		return errors.Err(err)
	}

	if n.IsZero() {
		return database.ErrNotFound
	}

	collaborators := namespace.NewCollaboratorStore(h.DB, n)

	selectq := user.Select("id", query.Where("username", "=", vars["collaborator"]))

	c, err := collaborators.Get(query.WhereQuery("user_id", "=", selectq))

	if err != nil {
		return errors.Err(err)
	}

	if c.IsZero() {
		return nil
	}

	err = collaborators.Delete(c)
	return errors.Err(err)
}
