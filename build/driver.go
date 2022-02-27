package build

import (
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/manifest"

	"github.com/andrewpillar/query"
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

func (d *Driver) Dest() []interface{} {
	return []interface{}{
		&d.ID,
		&d.BuildID,
		&d.Type,
		&d.Config,
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

// JSON implements the database.Model interface. This is a stub method.
func (*Driver) JSON(_ string) map[string]interface{} { return map[string]interface{}{} }

// Endpoint implements the database.Model interface. This is a stub method.
func (*Driver) Endpoint(_ ...string) string { return "" }

// Values returns all of the values for the current Driver.
func (d *Driver) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":       d.ID,
		"build_id": d.BuildID,
		"type":     d.Type,
		"config":   d.Config,
	}
}

// DriverStore allows for the retrieval of Drivers.
type DriverStore struct {
	database.Pool
}

var driverTable = "build_drivers"

// Get returns the singular build Driver that can be found with the given query
// options applied, along with whether or not one could be found.
func (s DriverStore) Get(opts ...query.Option) (*Driver, bool, error) {
	var d Driver

	ok, err := s.Pool.Get(driverTable, &d, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &d, ok, nil
}
