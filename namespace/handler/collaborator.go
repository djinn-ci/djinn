package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type Collaborator struct {
	web.Handler

	Invites    *namespace.InviteStore
	Namespaces *namespace.Store
}

func (h Collaborator) StoreModel(r *http.Request) (*namespace.Namespace, error) {
	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars["invite"], 10, 64)

	i, err := h.Invites.Get(query.Where("id", "=", id))

	if err != nil {
		return &namespace.Namespace{}, errors.Err(err)
	}

	err = h.Namespaces.Load(
		"id",
		[]interface{}{i.NamespaceID},
		model.Bind("namespace_id", "id", i),
	)

	if err != nil {
		return &namespace.Namespace{}, errors.Err(err)
	}

	collaborators := namespace.NewCollaboratorStore(h.DB, i.Namespace)

	c := collaborators.New()
	c.UserID = i.InviteeID

	if err := collaborators.Create(c); err != nil {
		return &namespace.Namespace{}, errors.Err(err)
	}

	err = h.Invites.Delete(i)
	return i.Namespace, errors.Err(err)
}

func (h Collaborator) Delete(r *http.Request) error {
	vars := mux.Vars(r)

	owner, err := h.Users.Get(query.Where("username", "=", vars["username"]))

	if err != nil {
		return errors.Err(err)
	}

	if owner.IsZero() {
		return model.ErrNotFound
	}

	path := strings.TrimSuffix(vars["namespace"], "/")

	n, err := namespace.NewStore(h.DB, owner).Get(query.Where("path", "=", path))

	if err != nil {
		return errors.Err(err)
	}

	if n.IsZero() {
		return model.ErrNotFound
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
