package build

import (
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
	limit     int64
	store     fs.Store
	artifacts *ArtifactStore
	users     user.Store
}

// NewCurator creates a new curator for cleaning up old artifacts from the
// given block store.
func NewCurator(db database.Pool, store fs.Store, limit int64) Curator {
	return Curator{
		limit: limit,
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
	uu, err := c.users.All(query.Where("cleanup", "=", query.Arg(true)))

	if err != nil {
		return errors.Err(err)
	}

	userIds := make([]interface{}, 0, len(uu))

	for _, u := range uu {
		userIds = append(userIds, u.ID)
	}

	aa, err := c.artifacts.All(
		query.Where("size", ">", query.Arg(0)),
		query.Where("user_id", "IN", database.List(userIds...)),
		query.Where("deleted_at", "IS", query.Lit("NULL")),
		query.OrderDesc("created_at"),
	)

	if err != nil {
		return errors.Err(err)
	}

	sums := make(map[int64]int64)
	curated := make(map[int64][]curationRecord)
	deleted := make([]int64, 0, len(aa))

	for _, a := range aa {
		sum := sums[a.UserID]
		sum += a.Size.Int64

		if sum >= c.limit {
			curated[a.UserID] = append(curated[a.UserID], curationRecord{
				artifact: a.ID,
				hash:     a.Hash,
			})
		}
		sums[a.UserID] = sum
	}

	for userId, records := range curated {
		part, err := c.store.Partition(userId)

		if err != nil {
			log.Error.Println("failed to partition artifact store", err)
			continue
		}

		for _, r := range records {
			log.Debug.Println("removing artifact", r.hash)

			if err := part.Remove(r.hash); err != nil {
				log.Error.Println("failed to remove artifact", r.hash, err)
				continue
			}
			deleted = append(deleted, r.artifact)
		}
	}

	if err := c.artifacts.Deleted(deleted...); err != nil {
		return errors.Err(err)
	}
	return err
}
