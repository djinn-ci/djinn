package cron

import (
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"

	"github.com/andrewpillar/query"
)

type Batcher struct {
	err       error
	paginator database.Paginator
	store     *Store
	crons     []*Cron
}

func NewBatcher(s *Store, limit int64) *Batcher {
	return &Batcher{
		store:     s,
		paginator: database.Paginator{
			Page:  1,
			Limit: limit,
		},
	}
}

func (b *Batcher) Next() bool {
	paginator, err := b.store.Paginate(b.paginator.Page, query.WhereRaw("NOW()", ">=", "next_run"))

	if err != nil {
		b.err = errors.Err(err)
		return false
	}

	crons, err := b.store.All(
		query.WhereRaw("NOW()", ">=", "next_run"),
		query.Limit(paginator.Limit),
		query.Offset(paginator.Offset),
	)

	if err != nil {
		b.err = errors.Err(err)
		return false
	}

	if len(crons) == 0 {
		return false
	}

	b.paginator = paginator
	b.crons = crons

	return paginator.Page == paginator.Pages[len(paginator.Pages)-1]
}

func (b *Batcher) Crons() []*Cron { return b.crons }

func (b *Batcher) Err() error { return b.err }
