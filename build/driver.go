package build

import (
	"database/sql"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/driver"
	"github.com/andrewpillar/djinn/errors"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Driver is the type that represents the driver being used by a build.
type Driver struct {
	ID      int64         `db:"id"`
	BuildID int64         `db:"build_id"`
	Type    driver.Type   `db:"type"`
	Config  driver.Config `db:"config"`

	Build *Build `db:"-"`
}

// DriverStore is the type for creating Driver models in the database.
type DriverStore struct {
	database.Store

	// Build is the bound Build model. If not nil this will bind the Build
	// model to any Driver models that are created. If not nil this will append
	// a WHERE clause on the build_id column for all SELECT queries performed.
	Build *Build
}

var (
	_ database.Model  = (*Driver)(nil)
	_ database.Binder = (*DriverStore)(nil)
	_ database.Loader = (*DriverStore)(nil)

	driverTable = "build_drivers"
)

// NewDriverStore returns a new DriverStore for querying the build_drivers
// table. Each database passed to this function will be bound to the returned
// DriverStore.
func NewDriverStore(db *sqlx.DB, mm ...database.Model) *DriverStore {
	s := &DriverStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// DriverModel is called along with database.ModelSlice to convert the given slice of
// Driver models to a slice of database.Model interfaces.
func DriverModel(dd []*Driver) func(int) database.Model {
	return func(i int) database.Model {
		return dd[i]
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to Build.
func (d *Driver) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *Build:
			d.Build = v
		}
	}
}

// SetPrimary implements the database.Model interface.
func (d *Driver) SetPrimary(id int64) { d.ID = id }

// Primary implements the database.Model interface.
func (d *Driver) Primary() (string, int64) { return "id", d.ID }

// IsZero implements the database.Model interface.
func (d *Driver) IsZero() bool {
	return d == nil || d.ID == 0 && d.BuildID == 0 && d.Type == driver.Type(0) && len(d.Config) == 0
}

// JSON implements the database.Model interface. This returns an empty map.
func (*Driver) JSON(_ string) map[string]interface{} { return map[string]interface{}{} }

// Endpoint implements the database.Model interface. This returns an empty string.
func (*Driver) Endpoint(_ ...string) string { return "" }

// Values implements the database.Model interface. This will return a map with
// the following values, build_id, type, and config.
func (d *Driver) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id": d.BuildID,
		"type":     d.Type,
		"config":   d.Config,
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to Build.
func (s *DriverStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *Build:
			s.Build = v
		}
	}
}

// New returns a new Driver binding any non-nil models to it from the current
// DriverStore.
func (s *DriverStore) New() *Driver {
	d := &Driver{
		Build: s.Build,
	}

	if s.Build != nil {
		d.BuildID = s.Build.ID
	}
	return d
}

// Create creates a new driver using the given configuration.
func (s *DriverStore) Create(cfg map[string]string) (*Driver, error) {
	d := s.New()
	d.Config = cfg

	if err := d.Type.UnmarshalText([]byte(cfg["type"])); err != nil {
		return d, errors.Err(err)
	}

	err := s.Store.Create(driverTable, d)
	return d, errors.Err(err)
}

// Get returns a single Driver database, applying each query.Option that is
// given. The database.Where option is applied to the *Build bound database.
func (s *DriverStore) Get(opts ...query.Option) (*Driver, error) {
	d := &Driver{
		Build: s.Build,
	}

	opts = append([]query.Option{
		database.Where(s.Build, "build_id"),
	}, opts...)

	err := s.Store.Get(d, driverTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return d, errors.Err(err)
}

// All returns a slice of Driver models, applying each query.Option that is
// given. The database.Where option is applied to the *Build bound database.
func (s *DriverStore) All(opts ...query.Option) ([]*Driver, error) {
	dd := make([]*Driver, 0)

	opts = append([]query.Option{
		database.Where(s.Build, "build_id"),
	}, opts...)

	err := s.Store.All(&dd, driverTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, d := range dd {
		d.Build = s.Build
	}
	return dd, errors.Err(err)
}

// Load loads in a slice of Driver models where the given key is in the list
// of given vals. Each database is loaded individually via a call to the given
// load callback. This method calls DriverStore.All under the hood, so any
// bound models will impact the models being loaded.
func (s *DriverStore) Load(key string, vals []interface{}, load database.LoaderFunc) error {
	dd, err := s.All(query.Where(key, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, d := range dd {
			load(i, d)
		}
	}
	return nil
}
