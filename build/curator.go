package build

import (
	"database/sql"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/log"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type curationRecord struct {
	artifact int64
	hash     string
}

// Curator is used for removing old build artifacts whose total size exceed
// the configured limit.
type Curator struct {
	store     fs.Store
	artifacts *ArtifactStore
	users     user.Store
}

// NewCurator creates a new curator for cleaning up old artifacts from the
// given block store.
func NewCurator(db database.Pool, store fs.Store) Curator {
	return Curator{
		store: store,
		artifacts: &ArtifactStore{
			Pool:  db,
			Store: store,
		},
		users: user.Store{Pool: db},
	}
}

// Invoke will remove any artifacts whose total size exceeds the configured
// limit. This will only do it for users who have "cleanup" enabled on their
// account.
func (c *Curator) Invoke(log *log.Logger) error {
	uu, err := c.users.All(query.Where("cleanup", ">", query.Arg(0)))

	if err != nil {
		return errors.Err(err)
	}

	userIds := make([]interface{}, 0, len(uu))

	cleanups := make(map[int64]int64)

	for _, u := range uu {
		log.Debug.Println("cleanup limit for user", u.Username, "is", u.Cleanup)

		userIds = append(userIds, u.ID)
		cleanups[u.ID] = u.Cleanup
	}

	q := query.Select(
		query.Columns("id", "user_id", "size"),
		query.From(artifactTable),
		query.Where("size", ">", query.Arg(0)),
		query.Where("user_id", "IN", database.List(userIds...)),
		query.Where("deleted_at", "IS", query.Lit("NULL")),
		query.OrderDesc("created_at"),
	)

	rows, err := c.artifacts.Query(q.Build(), q.Args()...)

	if err != nil {
		return errors.Err(err)
	}

	curated := make([]int64, 0)
	sumtab := make(map[int64]int64)

	var (
		id, userId int64
		size       sql.NullInt64
	)

	for rows.Next() {
		if err := rows.Scan(&id, &userId, &size); err != nil {
			return errors.Err(err)
		}

		sum := sumtab[userId]
		sum += size.Int64

		if limit, ok := cleanups[userId]; ok && sum >= limit {
			log.Debug.Println("curating artifact", id, "for user", userId)

			curated = append(curated, id)
		}
		sumtab[userId] = sum
	}

	if err := c.artifacts.Deleted(curated...); err != nil {
		return errors.Err(err)
	}
	return err
}
