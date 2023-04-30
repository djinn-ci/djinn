package build

import (
	"context"
	"encoding/json"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/key"
	"djinn-ci.com/object"

	"github.com/andrewpillar/fs"
	"github.com/andrewpillar/query"
)

type Object struct {
	ID        int64
	BuildID   int64
	ObjectID  database.Null[int64]
	Source    string
	Name      string
	Placed    bool
	CreatedAt time.Time

	Build  *Build
	Object *object.Object
}

var _ database.Model = (*Object)(nil)

func (o *Object) Primary() (string, any) { return "id", o.ID }

func (o *Object) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":         &o.ID,
		"build_id":   &o.BuildID,
		"object_id":  &o.ObjectID,
		"source":     &o.Source,
		"name":       &o.Name,
		"placed":     &o.Placed,
		"created_at": &o.CreatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (o *Object) Params() database.Params {
	return database.Params{
		"id":         database.ImmutableParam(o.ID),
		"build_id":   database.CreateOnlyParam(o.BuildID),
		"object_id":  database.CreateOnlyParam(o.ObjectID),
		"source":     database.CreateOnlyParam(o.Source),
		"name":       database.CreateOnlyParam(o.Name),
		"placed":     database.UpdateOnlyParam(o.Placed),
		"created_at": database.CreateOnlyParam(o.CreatedAt),
	}
}

func (o *Object) Bind(m database.Model) {
	switch v := m.(type) {
	case *Build:
		if o.BuildID == v.ID {
			o.Build = v
		}
	case *object.Object:
		if o.ObjectID.Elem == v.ID {
			o.Object = v
		}
	}
}

func (o *Object) MarshalJSON() ([]byte, error) {
	if o == nil {
		return []byte("null"), nil
	}

	raw := map[string]any{
		"build_id":   o.BuildID,
		"source":     o.Source,
		"name":       o.Name,
		"type":       nil,
		"md5":        nil,
		"sha256":     nil,
		"placed":     o.Placed,
		"build":      o.Build,
		"object_url": nil,
	}

	if o.Object != nil {
		raw["type"] = o.Object.Type
		raw["md5"] = o.Object.MD5
		raw["sha256"] = o.Object.SHA256
		raw["object_url"] = env.DJINN_API_SERVER + o.Object.Endpoint()
	}

	b, err := json.Marshal(raw)

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (*Object) Endpoint(...string) string { return "" }

type ObjectStore struct {
	*database.Store[*Object]

	FS fs.FS
}

const objectTable = "build_objects"

func SelectObject(expr query.Expr, opts ...query.Option) query.Query {
	return query.Select(expr, append([]query.Option{query.From(objectTable)}, opts...)...)
}

func NewObjectStore(pool *database.Pool) *database.Store[*Object] {
	return database.NewStore[*Object](pool, objectTable, func() *Object {
		return &Object{}
	})
}

type objectFilestore struct {
	fs.FS

	store *ObjectStore
	build *Build
	chain *key.Chain
}

func (s *objectFilestore) Open(name string) (fs.File, error) {
	if name == s.chain.ConfigName() {
		f, err := s.chain.Open(name)

		if err != nil {
			return nil, err
		}
		return f, nil
	}

	f, err := s.chain.Open(name)

	if err == nil {
		return f, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return f, errors.Err(err)
	}

	o, ok, err := object.NewStore(s.store.Pool).Get(
		context.Background(),
		query.Where("id", "=", query.Select(
			query.Columns("object_id"),
			query.From(objectTable),
			query.Where("build_id", "=", query.Arg(s.build.ID)),
			query.Where("source", "=", query.Arg(name)),
		)),
	)

	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	f, err = o.Open(s.store.FS)

	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: errors.Unwrap(err)}
	}
	return f, nil
}

func (s *ObjectStore) Filestore(b *Build, chain *key.Chain) fs.FS {
	return fs.ReadOnly(&objectFilestore{
		FS:    s.FS,
		store: s,
		build: b,
		chain: chain,
	})
}
