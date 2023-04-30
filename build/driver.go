package build

import (
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/manifest"
)

// Driver represents the Driver being used by a build.
type Driver struct {
	ID      int64
	BuildID int64
	Type    driver.Type
	Config  manifest.Driver

	Build *Build
}

var _ database.Model = (*Driver)(nil)

func (d *Driver) Primary() (string, any) { return "id", d.ID }

func (d *Driver) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":       &d.ID,
		"build_id": &d.BuildID,
		"type":     &d.Type,
		"config":   &d.Config,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (d *Driver) Params() database.Params {
	return database.Params{
		"id":       database.ImmutableParam(d.ID),
		"build_id": database.CreateOnlyParam(d.BuildID),
		"type":     database.CreateOnlyParam(d.Type),
		"config":   database.CreateOnlyParam(d.Config),
	}
}

// Bind the given Model to the current Driver if it is of type Build.
func (d *Driver) Bind(m database.Model) {
	if b, ok := m.(*Build); ok {
		if d.BuildID == b.ID {
			d.Build = b
		}
	}
}

func (*Driver) MarshalJSON() ([]byte, error) { return nil, nil }
func (*Driver) Endpoint(...string) string    { return "" }

type DriverStore struct {
	*database.Store[*Driver]
}

const driverTable = "build_drivers"

func NewDriverStore(pool *database.Pool) DriverStore {
	return DriverStore{
		Store: database.NewStore[*Driver](pool, driverTable, func() *Driver {
			return &Driver{}
		}),
	}
}
