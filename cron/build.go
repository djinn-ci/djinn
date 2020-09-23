package cron

import (
	"time"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/database"

	"github.com/jmoiron/sqlx"
)

type Build struct {
	ID        int64     `db:"id"`
	CronID    int64     `db:"cron_id"`
	BuildID   int64     `db:"build_id"`
	CreatedAt time.Time `db:"created_at"`

	Cron  *Cron        `db:"-"`
	Build *build.Build `db:"-"`
}

type BuildStore struct {
	database.Store

	Cron  *Cron
	Build *build.Build
}

var (
	_ database.Model = (*Build)(nil)

	buildTable = "cron_builds"
)

func NewBuildStore(db *sqlx.DB, mm ...database.Model) *BuildStore {
	s := &BuildStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func (s *BuildStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Cron:
			s.Cron = m.(*Cron)
		case *build.Build:
			s.Build = m.(*build.Build)
		}
	}
}

func (b *Build) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Cron:
			b.Cron = m.(*Cron)
		case *build.Build:
			b.Build = m.(*build.Build)
		}
	}
}

func (b *Build) SetPrimary(_ int64) { }

func (b *Build) Primary() (string, int64) { return "id", b.ID }

func (b *Build) IsZero() bool {
	return b == nil || b.ID == 0 && b.CronID == 0 && b.BuildID == 0 && b.CreatedAt == time.Time{}
}

func (b *Build) JSON(addr string) map[string]interface{} { return b.Build.JSON(addr) }

func (b *Build) Endpoint(uri ...string) string { return "" }

func (b *Build) Values() map[string]interface{} { return map[string]interface{}{} }
