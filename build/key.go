package build

import (
	"database/sql"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"

	"github.com/andrewpillar/query"
)

// Key represents an SSH key that has been deployed into a build environment.
type Key struct {
	ID       int64
	BuildID  int64
	KeyID    sql.NullInt64
	Name     string
	Key      []byte
	Config   string
	Location string

	Build *Build
}

var _ database.Model = (*Key)(nil)

func (k *Key) Dest() []interface{} {
	return []interface{}{
		&k.ID,
		&k.BuildID,
		&k.KeyID,
		&k.Name,
		&k.Key,
		&k.Config,
		&k.Location,
	}
}

// Bind the given Model to the current Key if it is of type Build, and if there
// is a direct relation between the two.
func (k *Key) Bind(m database.Model) {
	if b, ok := m.(*Build); ok {
		if k.BuildID == b.ID {
			k.Build = b
		}
	}
}

// JSON returns a map[string]interface{} representation of the current Key.
// This will include the Build model if it is non-nil.
func (k *Key) JSON(addr string) map[string]interface{} {
	if k == nil {
		return nil
	}

	json := map[string]interface{}{
		"id":       k.ID,
		"build_id": k.BuildID,
		"config":   k.Config,
		"location": k.Location,
	}

	if k.KeyID.Valid {
		json["key_id"] = k.KeyID.Int64
	}

	if k.Build != nil {
		json["build"] = k.Build.JSON(addr)
	}
	return json
}

// Endpoint implements the database.Model interface. This is a stub method.
func (*Key) Endpoint(...string) string { return "" }

// Values returns all of the values for the current Key.
func (k *Key) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":       k.ID,
		"build_id": k.BuildID,
		"key_id":   k.KeyID,
		"name":     k.Name,
		"key":      k.Key,
		"config":   k.Config,
		"location": k.Location,
	}
}

// KeyStore allows for the retrieval of Keys.
type KeyStore struct {
	database.Pool
}

var keyTable = "build_keys"

// All returns all of the build Keys that can be found with the given query
// options applied.
func (s *KeyStore) All(opts ...query.Option) ([]*Key, error) {
	kk := make([]*Key, 0)

	new := func() database.Model {
		k := &Key{}
		kk = append(kk, k)
		return k
	}

	if err := s.Pool.All(keyTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return kk, nil
}
