package build

import (
	"encoding/json"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
)

// Key represents an SSH key that has been deployed into a build environment.
type Key struct {
	ID       int64
	BuildID  int64
	KeyID    database.Null[int64]
	Name     string
	Key      []byte
	Config   string
	Location string

	Build *Build
}

var _ database.Model = (*Key)(nil)

func (k *Key) Primary() (string, any) { return "id", k.ID }

func (k *Key) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":       &k.ID,
		"build_id": &k.BuildID,
		"key_id":   &k.KeyID,
		"name":     &k.Name,
		"key":      &k.Key,
		"config":   &k.Config,
		"location": &k.Location,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (k *Key) Params() database.Params {
	return database.Params{
		"id":       database.ImmutableParam(k.ID),
		"build_id": database.CreateOnlyParam(k.BuildID),
		"key_id":   database.CreateOnlyParam(k.KeyID),
		"name":     database.CreateOnlyParam(k.Name),
		"key":      database.CreateOnlyParam(k.Key),
		"config":   database.CreateOnlyParam(k.Config),
		"location": database.CreateOnlyParam(k.Location),
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

func (k *Key) MarshalJSON() ([]byte, error) {
	if k == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"build_id": k.BuildID,
		"key_id":   k.KeyID,
		"config":   k.Config,
		"location": k.Location,
		"build":    k.Build,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (*Key) Endpoint(...string) string { return "" }

type KeyStore struct {
	*database.Store[*Key]
}

func NewKeyStore(pool *database.Pool) *database.Store[*Key] {
	return database.NewStore[*Key](pool, "build_keys", func() *Key {
		return &Key{}
	})
}
