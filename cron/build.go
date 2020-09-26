package cron

import (
	"time"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/database"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Build is the type that represents a build trigged via a cron job.
type Build struct {
	ID        int64     `db:"id"`
	CronID    int64     `db:"cron_id"`
	BuildID   int64     `db:"build_id"`
	CreatedAt time.Time `db:"created_at"`

	Cron  *Cron        `db:"-"`
	Build *build.Build `db:"-"`
}

// BuildStore is the type for creating Build models in the database.
type BuildStore struct {
	database.Store

	// Cron is the bound Cron model. If not nil this will bind the Cron model to
	// any Build models that are created. If not nil this will append a WHERE
	// clause on the cron_id column for all SELECT queries performed.
	Cron *Cron

	// Builds is the bound build.Build model. If not nil this will bind the
	// build.Build model to any Build models that are created. If not nil this
	// will append a WHERE clause to the build_id column for all SELECT queries
	// performed.
	Build *build.Build
}

var (
	_ database.Model = (*Build)(nil)

	buildTable = "cron_builds"
)

// NewBuildStore returns a new BuildStore for querying the cron_builds table.
// Each of the given models if bound to the returned BuildStore.
func NewBuildStore(db *sqlx.DB, mm ...database.Model) *BuildStore {
	s := &BuildStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// SelectBuild returns a SELECT query that will select the given column from
// the cron_builds table and apply the given query options.
func SelectBuild(col string, opts ...query.Option) query.Query {
	return query.Select(append([]query.Option{
		query.Columns(col),
		query.From(buildTable),
	}, opts...)...)
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either Cron or build.Build.
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

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either Cron or build.Build.
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

// SetPrimary implements the database.Model interface.
func (b *Build) SetPrimary(_ int64) { }

// Primary implements the database.Model interface.
func (b *Build) Primary() (string, int64) { return "id", b.ID }

// IsZero implements the database.Model interface.
func (b *Build) IsZero() bool {
	return b == nil || b.ID == 0 && b.CronID == 0 && b.BuildID == 0 && b.CreatedAt == time.Time{}
}

// JSON implements the database.Model interface. This will return the JSON of
// the bound build.Build model.
func (b *Build) JSON(addr string) map[string]interface{} { return b.Build.JSON(addr) }

// Endpoint implements the database.Model interface.
func (b *Build) Endpoint(uri ...string) string { return "" }

// Values implements the database.Model interface.
func (b *Build) Values() map[string]interface{} {
	return map[string]interface{}{
		"cron_id":  b.CronID,
		"build_id": b.BuildID,
	}
}
