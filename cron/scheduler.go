package cron

import (
	"context"
	"io"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/build"
	"djinn-ci.com/config"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/log"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type Scheduler struct {
	log    *log.Logger
	crons  Store
	users  *database.Store[*auth.User]
	builds *build.Store
	ticker *time.Ticker
	page   int
	batch  []*Cron
}

func NewScheduler(cfg *config.Scheduler) *Scheduler {
	db := cfg.DB()

	return &Scheduler{
		log:   cfg.Log(),
		crons: Store{Store: NewStore(db)},
		users: user.NewStore(db),
		builds: &build.Store{
			Store:  build.NewStore(db),
			Hasher: cfg.Hasher(),
			Queues: cfg.DriverQueues(),
		},
		ticker: time.NewTicker(cfg.Interval()),
	}
}

func (s *Scheduler) load(ctx context.Context, batchSize int) error {
	s.batch = s.batch[0:0]

	s.log.Debug.Println("loading next batch of", batchSize, "from page", s.page)

	p, err := s.crons.Paginate(ctx, s.page, batchSize, query.Where("NOW()", ">=", query.Lit("next_run")))

	if err != nil {
		return errors.Err(err)
	}

	if err := p.Load(ctx, s.crons.Store); err != nil {
		return errors.Err(err)
	}

	if len(p.Items) == 0 {
		return io.EOF
	}

	s.page = p.Page()

	ids := database.Map[*Cron, any](p.Items, func(c *Cron) any {
		return c.UserID
	})

	uu, err := s.users.All(ctx, query.Where("id", "IN", query.List(ids...)))

	if err != nil {
		return errors.Err(err)
	}

	usertab := make(map[int64]*auth.User)

	for _, u := range uu {
		usertab[u.ID] = u
	}

	for _, c := range p.Items {
		c.User = usertab[c.UserID]
	}

	s.batch = p.Items
	return nil
}

func (s *Scheduler) runBatch(ctx context.Context) (int, error) {
	n := 0

	for _, c := range s.batch {
		b, err := s.crons.Invoke(ctx, c)

		if err != nil {
			return n, errors.Err(err)
		}

		if err := s.builds.Submit(ctx, "djinn-scheduler", b); err != nil {
			return n, errors.Err(err)
		}
		n++
	}
	return n, nil
}

func (s *Scheduler) Run(ctx context.Context, batchSize int, errh func(error)) {
	for {
		select {
		case <-ctx.Done():
			s.log.Info.Println("stopping scheduler")
			s.ticker.Stop()
			return
		case <-s.ticker.C:
			for {
				s.log.Debug.Println("loading batch of size", batchSize)

				if err := s.load(ctx, batchSize); err != nil {
					if !errors.Is(err, io.EOF) {
						errh(errors.Err(err))
					}
					break
				}

				n, err := s.runBatch(ctx)

				s.log.Debug.Println("submitted", n, "build(s)")

				if err != nil {
					errh(errors.Err(err))
				}
			}
		}
	}
}
