package namespace

import (
	"net/http"
	"strings"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type Authenticator[M database.Model] struct {
	auth.Authenticator

	store    *database.Store[M]
	resource string
}

func NewAuth[M database.Model](a auth.Authenticator, res string, store *database.Store[M]) *Authenticator[M] {
	return &Authenticator[M]{
		Authenticator: a,
		store:         store,
		resource:      res,
	}
}

func (a *Authenticator[M]) getResourceAndOwner(r *http.Request) (database.Model, int64, bool, error) {
	vars := mux.Vars(r)
	ctx := r.Context()

	var zero M

	if a.resource == "namespace" {
		username := vars["username"]
		path := strings.TrimSuffix(vars["namespace"], "/")

		n, ok, err := NewStore(a.store.Pool).Get(
			ctx,
			query.Where("id", "=", Select(
				query.Columns("root_id"),
				query.Where("user_id", "=", user.Select(
					query.Columns("id"),
					query.Where("username", "=", query.Arg(username)),
				)),
				query.Where("path", "=", query.Arg(path)),
			)),
		)

		if err != nil {
			return zero, 0, false, errors.Err(err)
		}

		if !ok {
			return zero, 0, false, nil
		}
		return n, n.UserID, ok, nil
	}

	if a.resource == "invite" {
		m, ok, err := a.store.SelectOne(
			ctx,
			[]string{"invitee_id", "namespace_id"},
			query.Where("id", "=", query.Arg(vars[a.resource])),
		)

		if err != nil {
			return nil, 0, false, errors.Err(err)
		}

		if !ok {
			return nil, 0, false, nil
		}

		userId := m.Params()["invitee_id"]

		return m, userId.Value.(int64), ok, nil
	}

	m, ok, err := a.store.SelectOne(
		ctx,
		[]string{"user_id", "namespace_id"},
		query.Where("id", "=", query.Arg(vars[a.resource])),
	)

	if err != nil {
		return nil, 0, false, errors.Err(err)
	}

	if !ok {
		return nil, 0, false, nil
	}

	userId := m.Params()["user_id"]

	return m, userId.Value.(int64), ok, nil
}

func (a *Authenticator[M]) Auth(r *http.Request) (*auth.User, error) {
	u, err := a.Authenticator.Auth(r)

	if err != nil {
		if errors.Is(err, auth.ErrAuth) {
			return nil, err
		}
		return nil, errors.Err(err)
	}

	m, userId, ok, err := a.getResourceAndOwner(r)

	if err != nil {
		return nil, errors.Err(err)
	}

	if !ok {
		return nil, database.ErrNoRows
	}

	var root *Namespace

	if n, ok := m.(*Namespace); ok {
		root = n
	}

	if userId == u.ID {
		u.Grant("owner")
	}

	ctx := r.Context()

	params := m.Params()

	if p, ok := params["namespace_id"]; ok {
		var namespaceId int64

		switch v := p.Value.(type) {
		case int64:
			namespaceId = v
		case database.Null[int64]:
			namespaceId = v.Elem
		}

		if namespaceId == 0 {
			if userId != u.ID {
				return nil, auth.ErrAuth
			}
			return u, nil
		}

		root, _, err = NewStore(a.store.Pool).Get(
			ctx,
			query.Where("id", "=", Select(
				query.Columns("root_id"),
				query.Where("id", "=", query.Arg(namespaceId)),
			)),
		)

		if err != nil {
			return nil, errors.Err(err)
		}
	}

	if r.Method == "GET" {
		// Public namespace, so grant read permission.
		if root.Visibility == Public {
			u.Grant(a.resource + ":read")
		}

		if err := root.HasAccess(ctx, a.store.Pool, u); err != nil {
			if errors.Is(err, auth.ErrPermission) {
				return nil, err
			}
			return nil, errors.Err(err)
		}
		return u, nil
	}

	// Only check if user is a collaborator if the resource is not an invite.
	// Invites are the only namespace resource that can be accessed when not in
	// a namespace, so this is a special case.
	if a.resource != "invite" {
		if err := root.IsCollaborator(ctx, a.store.Pool, u); err != nil {
			if errors.Is(err, auth.ErrPermission) {
				return nil, auth.ErrAuth
			}
			return nil, errors.Err(err)
		}
	}
	return u, nil
}
