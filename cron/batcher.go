package cron

import (
	"context"
	"fmt"

	"djinn-ci.com/build"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/mcmathja/curlyq"
)

// Batcher provides a way of retrieving batches of cron jobs that are ready
// to be executed.
type Batcher struct {
	err        error
	atEOF      bool
	page       int64
	limit      int64
	crons      Store
	builds     *build.Store
	users      user.Store
	namespaces namespace.Store
	batch      []*Cron
	errh       func(error)
}

// NewBatcher returns a new Batcher using the given Store to retrieve cron jobs
// from, and setting the size of each batch to the given limit.
func NewBatcher(db database.Pool, hasher *crypto.Hasher, drivers map[string]*curlyq.Producer, limit int64, errh func(error)) *Batcher {
	return &Batcher{
		crons: Store{Pool: db},
		builds: &build.Store{
			Pool:         db,
			Hasher:       hasher,
			DriverQueues: drivers,
		},
		users:      user.Store{Pool: db},
		namespaces: namespace.Store{Pool: db},
		errh:       errh,
		page:       1,
		limit:      limit,
	}
}

// Load will load in the next batch of cron jobs to be executed
// (WHERE NOW() >= next_run). This will return false if it reaches the end of
// the batches in the table. If the end of the table is reached, or if an error
// happens then false is returned.
func (b *Batcher) Load() bool {
	b.batch = b.batch[0:0]

	if b.atEOF {
		return false
	}

	paginator, err := b.crons.Paginate(b.page, b.limit, query.Where("NOW()", ">=", query.Lit("next_run")))

	if err != nil {
		b.err = errors.Err(err)
		return false
	}

	cc, err := b.crons.All(
		query.Where("NOW()", ">=", query.Lit("next_run")),
		query.Limit(paginator.Limit),
		query.Offset(paginator.Offset),
	)

	if err != nil {
		b.err = errors.Err(err)
		return false
	}

	if len(cc) == 0 {
		return false
	}

	userIds := make([]interface{}, 0, len(cc))

	for _, c := range cc {
		userIds = append(userIds, c.UserID)
	}

	uu, err := b.users.All(query.Where("id", "IN", query.List(userIds...)))

	if err != nil {
		b.err = errors.Err(err)
		return false
	}

	users := make(map[int64]*user.User)

	for _, u := range uu {
		users[u.ID] = u
	}

	for _, c := range cc {
		c.User = users[c.UserID]
	}

	b.batch = cc

	if paginator.Page == paginator.Next {
		b.atEOF = true
	}
	return true
}

func (b *Batcher) Batch() []*Cron { return b.batch }

// Err returns the current error, if any, that occurred when loading a batch.
func (b *Batcher) Err() error { return b.err }

// Invoke will submit a build for each job in the current batch.
func (b *Batcher) Invoke(ctx context.Context) int {
	n := 0

	for _, c := range b.batch {
		if c.NamespaceID.Valid {
			n, _, err := b.namespaces.Get(query.Where("id", "=", query.Arg(c.NamespaceID.Int64)))

			if err != nil {
				b.errh(fmt.Errorf("failed to get namespace: %v", errors.Err(err)))
				continue
			}

			u, _, err := b.users.Get(query.Where("user_id", "=", query.Arg(n.UserID)))

			if err != nil {
				b.errh(fmt.Errorf("failed to get namespace owner: %v", errors.Err(err)))
				continue
			}
			c.Manifest.Namespace = n.Path + "@" + u.Username
		}

		build, err := b.crons.Invoke(c)

		if err != nil {
			b.errh(fmt.Errorf("failed to invoke cron: %v", errors.Err(err)))
			continue
		}

		typ := build.Manifest.Driver["type"]

		if typ == "qemu" {
			arch := "x86_64"
			typ += "-" + arch
		}

		if err := b.builds.Submit(ctx, "djinn-scheduler", build); err != nil {
			b.errh(fmt.Errorf("failed to submit build: %v", errors.Err(err)))
			continue
		}
		n++
	}
	return n
}
