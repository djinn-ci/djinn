package core

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type Collaborator struct {
	web.Handler

	Invites model.InviteStore
}

func (h Collaborator) Destroy(r *http.Request) error {
	vars := mux.Vars(r)

	owner, err := h.Users.Get(query.Where("username", "=", vars["username"]))

	if err != nil {
		return errors.Err(err)
	}

	n, err := owner.NamespaceStore().Get(query.Where("path", "=", strings.TrimSuffix(vars["namespace"], "/")))

	if err != nil {
		return errors.Err(err)
	}

	collaborators := n.CollaboratorStore()

	c, err := collaborators.Get(
		query.WhereQuery("user_id", "=",
			query.Select(
				query.Columns("id"),
				query.From(model.UserTable),
				query.Where("username", "=", vars["collaborators"]),
			),
		),
	)

	if err != nil {
		return errors.Err(err)
	}

	if c.IsZero() {
		return nil
	}

	err = collaborators.Delete(c)

	return errors.Err(err)
}

func (h Collaborator) Store(r *http.Request) (*model.Namespace, error) {
	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars["invite"], 10, 64)

	i, err := h.Invites.Get(query.Where("id", "=", id))

	if err != nil {
		return &model.Namespace{}, errors.Err(err)
	}

	if err := i.LoadNamespace(); err != nil {
		return &model.Namespace{}, errors.Err(err)
	}

	collaborators := i.Namespace.CollaboratorStore()

	c := collaborators.New()
	c.UserID = i.InviteeID

	if err := collaborators.Create(c); err != nil {
		return i.Namespace, errors.Err(err)
	}

	err = h.Invites.Delete(i)

	return i.Namespace, errors.Err(err)
}
