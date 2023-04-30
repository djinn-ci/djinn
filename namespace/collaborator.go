package namespace

import (
	"encoding/json"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"

	"github.com/andrewpillar/query"
)

type Collaborator struct {
	ID          int64
	NamespaceID int64
	UserID      int64
	CreatedAt   time.Time

	User      *auth.User
	Namespace *Namespace
}

var _ database.Model = (*Collaborator)(nil)

func (c *Collaborator) Primary() (string, any) { return "id", c.ID }

func (c *Collaborator) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":           &c.ID,
		"namespace_id": &c.NamespaceID,
		"user_id":      &c.UserID,
		"created_at":   &c.CreatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (c *Collaborator) Params() database.Params {
	return database.Params{
		"id":           database.ImmutableParam(c.ID),
		"namespace_id": database.CreateOnlyParam(c.NamespaceID),
		"user_id":      database.CreateOnlyParam(c.UserID),
		"created_at":   database.CreateOnlyParam(c.CreatedAt),
	}
}

func (c *Collaborator) Bind(m database.Model) {
	switch v := m.(type) {
	case *Namespace:
		if c.NamespaceID == v.RootID.Elem {
			c.Namespace = v
		}
	case *auth.User:
		if c.UserID == v.ID {
			c.User = v
		}
	}
}

func (c *Collaborator) MarshalJSON() ([]byte, error) {
	if c.User == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"id":         c.User.ID,
		"email":      c.User.Email,
		"username":   c.User.Username,
		"created_at": c.CreatedAt,
		"url":        env.DJINN_API_SERVER + c.Endpoint(),
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (c *Collaborator) Endpoint(elems ...string) string {
	if c.Namespace == nil {
		return ""
	}
	if c.User == nil {
		return ""
	}
	return c.Namespace.Endpoint(append(elems, "collaborators", c.User.Username)...)
}

const collaboratorTable = "namespace_collaborators"

func NewCollaboratorStore(pool *database.Pool) *database.Store[*Collaborator] {
	return database.NewStore[*Collaborator](pool, collaboratorTable, func() *Collaborator {
		return &Collaborator{}
	})
}

func SelectCollaborator(expr query.Expr, opts ...query.Option) query.Query {
	return query.Select(expr, append([]query.Option{query.From(collaboratorTable)}, opts...)...)
}

func WhereCollaborator(userId int64) query.Option {
	return func(q query.Query) query.Query {
		return query.Options(
			query.Where("namespace_id", "IN",
				query.Select(
					query.Columns("id"),
					query.From(table),
					query.Where("root_id", "IN",
						query.Union(
							query.Select(
								query.Columns("namespace_id"),
								query.From(collaboratorTable),
								query.Where("user_id", "=", query.Arg(userId)),
							),
							query.Select(
								query.Columns("id"),
								query.From(table),
								query.Where("user_id", "=", query.Arg(userId)),
							),
						),
					),
				),
			),
			query.OrWhere("user_id", "=", query.Arg(userId)),
		)(q)
	}
}
