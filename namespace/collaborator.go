package namespace

import (
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type ChownFunc func(database.Pool, int64, int64) error

type Collaborator struct {
	ID          int64
	NamespaceID int64
	UserID      int64
	CreatedAt   time.Time

	User      *user.User
	Namespace *Namespace
}

var _ database.Model = (*Collaborator)(nil)

func (c *Collaborator) Dest() []interface{} {
	return []interface{}{
		&c.ID,
		&c.NamespaceID,
		&c.UserID,
		&c.CreatedAt,
	}
}

func (c *Collaborator) Bind(m database.Model) {
	switch v := m.(type) {
	case *Namespace:
		if c.NamespaceID == v.RootID.Int64 {
			c.Namespace = v
		}
	case *user.User:
		if c.UserID == v.ID {
			c.User = v
		}
	}
}
func (c *Collaborator) JSON(addr string) map[string]interface{} {
	if c.User == nil {
		return nil
	}
	json := c.User.JSON(addr)
	json["created_at"] = c.CreatedAt.Format(time.RFC3339)
	json["url"] = addr + c.Endpoint()
	return json
}

func (c *Collaborator) Endpoint(uri ...string) string {
	if c.Namespace == nil {
		return ""
	}
	if c.User == nil {
		return ""
	}
	return c.Namespace.Endpoint(append(uri, "collaborators", c.User.Username)...)
}

func (c *Collaborator) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":           c.ID,
		"namespace_id": c.NamespaceID,
		"user_id":      c.UserID,
		"created_at":   c.CreatedAt,
	}
}

type CollaboratorStore struct {
	database.Pool
}

var collaboratorTable = "namespace_collaborators"

func CollaboratorSelect(col string, opts ...query.Option) query.Query {
	return query.Select(query.Columns(col), append([]query.Option{query.From(collaboratorTable)}, opts...)...)
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

func getNamespaceOwnerId(db database.Pool, namespaceId int64) (int64, error) {
	q := query.Select(
		query.Columns("user_id"),
		query.From(table),
		query.Where("id", "=", query.Arg(namespaceId)),
	)

	var ownerId int64

	if err := db.QueryRow(q.Build(), q.Args()...).Scan(&ownerId); err != nil {
		return 0, errors.Err(err)
	}
	return ownerId, nil
}

func (s CollaboratorStore) Delete(namespaceId, userId int64, chowns ...ChownFunc) error {
	ownerId, err := getNamespaceOwnerId(s.Pool, namespaceId)

	if err != nil {
		return errors.Err(err)
	}

	q := query.Delete(
		collaboratorTable,
		query.Where("namespace_id", "=", query.Arg(namespaceId)),
		query.Where("user_id", "=", query.Arg(userId)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}

	for _, chown := range chowns {
		if err := chown(s.Pool, userId, ownerId); err != nil {
			return errors.Err(err)
		}
	}
	return nil
}

func (s CollaboratorStore) Get(opts ...query.Option) (*Collaborator, bool, error) {
	var c Collaborator

	ok, err := s.Pool.Get(collaboratorTable, &c, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &c, ok, nil
}

func (s CollaboratorStore) All(opts ...query.Option) ([]*Collaborator, error) {
	cc := make([]*Collaborator, 0)

	new := func() database.Model {
		c := &Collaborator{}
		cc = append(cc, c)
		return c
	}

	if err := s.Pool.All(collaboratorTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return cc, nil
}
