package build

import (
	"database/sql"

	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Driver struct {
	ID      int64       `db:"id"`
	BuildID int64       `db:"build_id"`
	Type    driver.Type `db:"type"`
	Config  string      `db:"config"`

	Build *Build `db:"-"`
}

type DriverStore struct {
	model.Store

	Build *Build
}

var (
	_ model.Model  = (*Driver)(nil)
	_ model.Binder = (*DriverStore)(nil)
	_ model.Loader = (*DriverStore)(nil)

	driverTable = "build_drivers"
)

// NewDriverStore returns a new DriverStore for querying the build_drivers
// table. Each model passed to this function will be bound to the returned
// DriverStore.
func NewDriverStore(db *sqlx.DB, mm ...model.Model) *DriverStore {
	s := &DriverStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// DriverModel is called along with model.Slice to convert the given slice of
// Driver models to a slice of model.Model interfaces.
func DriverModel(dd []*Driver) func(int) model.Model {
	return func(i int) model.Model {
		return dd[i]
	}
}

// Bind the given models to the current Driver. This will only bind the model if
// they are one of the following,
//
// - *Build
func (d *Driver) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			d.Build = m.(*Build)
		}
	}
}

func (d *Driver) SetPrimary(id int64) {
	d.ID = id
}

func (d *Driver) Primary() (string, int64) {
	return "id", d.ID
}

func (d *Driver) IsZero() bool {
	return d == nil ||  d.ID == 0 && d.BuildID == 0 && d.Type == driver.Type(0) && d.Config == ""
}

// Endpoint is a stub to fulfill the model.Model interface. It returns an empty
// string.
func (*Driver) Endpoint(_ ...string) string { return "" }

func (d *Driver) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id": d.BuildID,
		"type":     d.Type,
		"config":   d.Config,
	}
}

// Bind the given models to the current DriverStore. This will only bind the
// model if they are one of the following,
//
// - *Build
func (s *DriverStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
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

// Create inserts the given Driver models into the build_drivers table.
func (s *DriverStore) Create(dd ...*Driver) error {
	models := model.Slice(len(dd), DriverModel(dd))
	return errors.Err(s.Store.Create(driverTable, models...))
}

// Get returns a single Driver model, applying each query.Option that is
// given. The model.Where option is applied to the *Build bound model.
func (s *DriverStore) Get(opts ...query.Option) (*Driver, error) {
	d := &Driver{
		Build: s.Build,
	}

	opts = append([]query.Option{
		model.Where(s.Build, "build_id"),
	}, opts...)

	err := s.Store.Get(d, driverTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return d, errors.Err(err)
}

// All returns a slice of Driver models, applying each query.Option that is
// given. The model.Where option is applied to the *Build bound model.
func (s *DriverStore) All(opts ...query.Option) ([]*Driver, error) {
	dd := make([]*Driver, 0)

	opts = append([]query.Option{
		model.Where(s.Build, "build_id"),
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
// of given vals. Each model is loaded individually via a call to the given
// load callback. This method calls DriverStore.All under the hood, so any
// bound models will impact the models being loaded.
func (s *DriverStore) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	dd, err := s.All(query.Where(key, "IN", vals...))

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
