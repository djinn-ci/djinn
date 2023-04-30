package build

import (
	"context"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/log"
	"djinn-ci.com/user"

	"github.com/andrewpillar/fs"
	"github.com/andrewpillar/query"
)

// Curator is used for removing old build artifacts whose total size exceed
// the configured limit.
type Curator struct {
	log       *log.Logger
	store     fs.FS
	artifacts *ArtifactStore
	users     *database.Store[*auth.User]
}

// NewCurator creates a new curator for cleaning up old artifacts from the
// given block store.
func NewCurator(log *log.Logger, pool *database.Pool, store fs.FS) Curator {
	return Curator{
		log:   log,
		store: store,
		artifacts: &ArtifactStore{
			Store: NewArtifactStore(pool),
			FS:    store,
		},
		users: user.NewStore(pool),
	}
}

// Invoke will remove any artifacts whose total size exceeds the configured
// limit. This will only do it for users who have "cleanup" enabled on their
// account.
func (c *Curator) Invoke() error {
	ctx := context.Background()

	uu, err := c.users.Select(ctx, []string{"id", "username", "cleanup"}, query.Where("cleanup", ">", query.Arg(0)))

	if err != nil {
		return errors.Err(err)
	}

	cleanuptab := make(map[int64]int64)
	userIds := make([]any, 0, len(uu))

	for _, u := range uu {
		cleanup := user.Cleanup(u)

		c.log.Debug.Println("cleanup limit for user", u.Username, "is", cleanup)

		cleanuptab[u.ID] = cleanup
		userIds = append(userIds, u.ID)
	}

	aa, err := c.artifacts.Select(
		ctx,
		[]string{"id", "user_id", "hash", "size"},
		query.Where("size", ">", query.Arg(0)),
		query.Where("user_id", "IN", query.List(userIds...)),
		query.Where("build_id", "NOT IN", query.Select(
			query.Columns("id"),
			query.From(table),
			query.Where("pinned", "=", query.Arg(true)),
		)),
		query.Where("deleted_at", "IS", query.Lit("NULL")),
		query.OrderDesc("created_at"),
	)

	if err != nil {
		return errors.Err(err)
	}

	sumtab := make(map[int64]int64)
	curated := make([]*Artifact, 0, len(aa))

	for _, a := range aa {
		sum := sumtab[a.UserID]
		sum += a.Size.Elem

		if limit, ok := cleanuptab[a.UserID]; ok && sum > limit {
			c.log.Debug.Println("curating artifact", a.ID, "for user", a.UserID)

			curated = append(curated, a)
		}
		sumtab[a.UserID] = sum
	}

	if err := c.artifacts.Delete(ctx, curated...); err != nil {
		return errors.Err(err)
	}
	return err
}
