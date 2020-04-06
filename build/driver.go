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

func NewDriverStore(db *sqlx.DB, mm ...model.Model) DriverStore {
	s := DriverStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func DriverModel(dd []*Driver) func(int) model.Model {
	return func(i int) model.Model {
		return dd[i]
	}
}

func (d *Driver) Bind(mm ...model.Model) {
	if d == nil {
		return
	}

	for _, m := range mm {
		switch m.(type) {
		case *Build:
			d.Build = m.(*Build)
		}
	}
}

func (*Driver) Kind() string { return "build_driver "}

func (d *Driver) SetPrimary(id int64) {
	if d == nil {
		return
	}
	d.ID = id
}

func (d *Driver) Primary() (string, int64) {
	if d == nil {
		return "id", 0
	}
	return "id", d.ID
}

func (d *Driver) IsZero() bool {
	return d == nil ||  d.ID == 0 && d.BuildID == 0 && d.Type == driver.Type(0) && d.Config == ""
}

func (*Driver) Endpoint(_ ...string) string { return "" }

func (d *Driver) Values() map[string]interface{} {
	if d == nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"build_id": d.BuildID,
		"type":     d.Type,
		"config":   d.Config,
	}
}

func (s *DriverStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		}
	}
}

func (s DriverStore) New() *Driver {
	d := &Driver{
		Build: s.Build,
	}

	if s.Build != nil {
		d.BuildID = s.Build.ID
	}
	return d
}

func (s DriverStore) Create(dd ...*Driver) error {
	models := model.Slice(len(dd), DriverModel(dd))
	return errors.Err(s.Store.Create(driverTable, models...))
}

func (s DriverStore) Get(opts ...query.Option) (*Driver, error) {
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

func (s DriverStore) All(opts ...query.Option) ([]*Driver, error) {
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

func (s DriverStore) Load(key string, vals []interface{}, load model.LoaderFunc) error {
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
