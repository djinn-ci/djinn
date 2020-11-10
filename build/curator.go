package build

import (
	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Curator is used for removing old build artifacts whose total size exceed
// the configured limit.
type Curator struct {
	limit     int64
	artifacts block.Store
	store     *ArtifactStore
	users     *user.Store
}

// NewCurator creates a new curator for cleaning up old artifacts from the
// given block store.
func NewCurator(db *sqlx.DB, artifacts block.Store, limit int64) Curator {
	return Curator{
		limit:     limit,
		artifacts: artifacts,
		store:     NewArtifactStore(db),
		users:     user.NewStore(db),
	}
}

// Invoke will remove any artifacts whose total size exceeds the configured
// limit. This will only do it for users who have "cleanup" enabled on their
// account.
func (c *Curator) Invoke() error {
	uu, err := c.users.All(query.Where("cleanup", "=", query.Arg(true)))

	if err != nil {
		return errors.Err(err)
	}

	mm := database.ModelSlice(len(uu), user.Model(uu))

	aa, err := c.store.All(
		query.Where("size", ">", query.Arg(0)),
		query.Where("user_id", "IN", query.List(database.MapKey("id", mm)...)),
		query.OrderAsc("created_at"),
	)

	if err != nil {
		return errors.Err(err)
	}

	sums := make(map[int64]int64)
	curated := make(map[int64][]string)

	for _, a := range aa {
		sum := sums[a.UserID]
		sum += a.Size.Int64

		if sum >= c.limit {
			curated[a.UserID] = append(curated[a.UserID], a.Hash)
		}
	}

	errs := make([]error, 0)

	for _, hashes := range curated {
		for _, hash := range hashes {
			if err := c.artifacts.Remove(hash); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Slice(errs)
}
