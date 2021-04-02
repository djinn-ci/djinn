package cron

import (
	"context"
	"fmt"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"

	"github.com/mcmathja/curlyq"
)

// Batcher provides a way of retrieving batches of cron jobs that are ready
// to be executed.
type Batcher struct {
	err       error
	atEOF     bool
	page      int64
	limit     int64
	store     *Store
	builds    *build.Store
	batch     []*Cron
	errh      func(error)
}

// NewBatcher returns a new Batcher using the given Store to retrieve cron jobs
// from, and setting the size of each batch to the given limit.
func NewBatcher(db *sqlx.DB, hasher *crypto.Hasher, limit int64, errh func(error)) *Batcher {
	return &Batcher{
		store:  NewStore(db),
		builds: build.NewStoreWithHasher(db, hasher),
		errh:   errh,
		page:   1,
		limit:  limit,
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

	paginator, err := b.store.Paginate(b.page, b.limit, query.Where("NOW()", ">=", query.Lit("next_run")))

	if err != nil {
		b.err = errors.Err(err)
		return false
	}

	cc, err := b.store.All(
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

	mm := database.ModelSlice(len(cc), Model(cc))

	uu, err := user.NewStore(b.store.DB).All(query.Where("id", "IN", query.List(database.MapKey("user_id", mm)...)))

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
func (b *Batcher) Invoke(ctx context.Context, producers map[string]*curlyq.Producer) int {
	namespaces := namespace.NewStore(b.store.DB)
	users := user.NewStore(b.store.DB)

	n := 0

	for _, c := range b.batch {
		if c.NamespaceID.Valid {
			n, err := namespaces.Get(query.Where("id", "=", query.Arg(c.NamespaceID.Int64)))

			if err != nil {
				b.errh(fmt.Errorf("failed to get namespace: %v", errors.Err(err)))
				continue
			}

			u, err := users.Get(query.Where("user_id", "=", query.Arg(n.UserID)))

			if err != nil {
				b.errh(fmt.Errorf("failed to get namespace owner: %v", errors.Err(err)))
				continue
			}
			c.Manifest.Namespace = n.Path + "@" + u.Username
		}

		build, err := b.store.Invoke(c)

		if err != nil {
			b.errh(fmt.Errorf("failed to invoke cron: %v", errors.Err(err)))
			continue
		}

		typ := build.Manifest.Driver["type"]

		if typ == "qemu" {
			arch := "x86_64"
			typ += "-" + arch
		}

		queue, ok := producers[typ]

		if !ok {
			b.errh(fmt.Errorf("invalid build driver: %v", build.Manifest.Driver["type"]))
			continue
		}

		if err := b.builds.Submit(ctx, queue, "djinn-scheduler", build); err != nil {
			b.errh(fmt.Errorf("failed to submit build: %v", errors.Err(err)))
			continue
		}
		n++
	}
	return n
}
