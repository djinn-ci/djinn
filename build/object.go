package build

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/key"
	"djinn-ci.com/object"
	"djinn-ci.com/runner"

	"github.com/andrewpillar/query"
)

type Object struct {
	ID        int64
	BuildID   int64
	ObjectID  sql.NullInt64
	Source    string
	Name      string
	Placed    bool
	CreatedAt time.Time

	Build  *Build
	Object *object.Object
}

var _ database.Model = (*Object)(nil)

type ObjectStore struct {
	database.Pool
	fs.Store
}

var (
	_ database.Loader = (*ObjectStore)(nil)
	_ runner.Placer   = (*ObjectStore)(nil)

	objectTable = "build_objects"
)

func SelectBuildIDForObject(objectId int64) query.Query {
	return query.Select(
		query.Columns("build_id"),
		query.From(objectTable),
		query.Where("object_id", "=", query.Arg(objectId)),
	)
}

func (o *Object) Dest() []interface{} {
	return []interface{}{
		&o.ID,
		&o.BuildID,
		&o.ObjectID,
		&o.Source,
		&o.Name,
		&o.Placed,
		&o.CreatedAt,
	}
}

func (o *Object) Bind(m database.Model) {
	switch v := m.(type) {
	case *Build:
		if o.BuildID == v.ID {
			o.Build = v
		}
	case *object.Object:
		if o.ObjectID.Int64 == v.ID {
			o.Object = v
		}
	}
}

func (o *Object) Endpoint(_ ...string) string { return "" }

func (o *Object) JSON(addr string) map[string]interface{} {
	if o == nil {
		return nil
	}

	json := map[string]interface{}{
		"id":       o.ID,
		"build_id": o.BuildID,
		"source":   o.Source,
		"name":     o.Name,
		"type":     nil,
		"md5":      nil,
		"sha256":   nil,
		"placed":   o.Placed,
	}

	if o.Build != nil {
		json["build"] = o.Build.JSON(addr)
	}

	if o.Object != nil {
		json["type"] = o.Object.Type
		json["md5"] = fmt.Sprintf("%x", o.Object.MD5)
		json["sha256"] = fmt.Sprintf("%x", o.Object.SHA256)
		json["object_url"] = addr + o.Object.Endpoint()
	}
	return json
}

func (o *Object) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":        o.ID,
		"build_id":  o.BuildID,
		"object_id": o.ObjectID,
		"source":    o.Source,
		"name":      o.Name,
		"placed":    o.Placed,
	}
}

func (s *ObjectStore) Get(opts ...query.Option) (*Object, bool, error) {
	var o Object

	ok, err := s.Pool.Get(objectTable, &o, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &o, ok, nil
}

func (s *ObjectStore) All(opts ...query.Option) ([]*Object, error) {
	oo := make([]*Object, 0)

	new := func() database.Model {
		o := &Object{}
		oo = append(oo, o)
		return o
	}

	if err := s.Pool.All(objectTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return oo, nil
}

func (s *ObjectStore) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	oo, err := s.All(query.Where(pk, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	for _, o := range oo {
		for _, m := range mm {
			m.Bind(o)
		}
	}
	return nil
}

func (s *ObjectStore) getObjectToPlace(buildId int64, name string) (*Object, error) {
	o, ok, err := s.Get(
		query.Where("build_id", "=", query.Arg(buildId)),
		query.Where("source", "=", query.Arg(name)),
	)

	if err != nil {
		return nil, errors.Err(err)
	}

	if !ok {
		return nil, &fs.PathError{
			Op:   "place",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	objects := object.Store{Pool: s.Pool}

	o.Object, ok, err = objects.Get(query.Where("id", "=", query.Arg(o.ObjectID)))

	if err != nil {
		return o, errors.Err(err)
	}

	if !ok {
		return nil, &fs.PathError{
			Op:   "place",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}
	return o, nil
}

type placer struct {
	store   *ObjectStore
	chain   *key.Chain
	userId  int64
	buildId int64
}

func (p *placer) Stat(name string) (os.FileInfo, error) {
	if name == p.chain.ConfigName() {
		return p.chain.ConfigFileInfo(), nil
	}

	if strings.HasPrefix(name, "key:") {
		info, err := p.chain.Stat(strings.TrimPrefix(name, "key:"))

		if err != nil {
			return nil, errors.Err(err)
		}
		return info, nil
	}

	o, err := p.store.getObjectToPlace(p.buildId, name)

	if err != nil {
		return nil, errors.Err(err)
	}

	info, err := p.store.Stat(o.Object.Hash)

	if err != nil {
		return nil, errors.Err(err)
	}
	return info, nil
}

func (p *placer) Place(name string, w io.Writer) (int64, error) {
	if name == p.chain.ConfigName() {
		n, err := io.WriteString(w, p.chain.Config())

		if err != nil {
			return 0, errors.Err(err)
		}
		return int64(n), nil
	}

	if strings.HasPrefix(name, "key:") {
		n, err := p.chain.Place(strings.TrimPrefix(name, "key:"), w)

		if err != nil {
			return 0, errors.Err(err)
		}
		return n, nil
	}

	o, err := p.store.getObjectToPlace(p.buildId, name)

	if err != nil {
		return 0, errors.Err(err)
	}

	store, err := p.store.Partition(p.userId)

	if err != nil {
		return 0, errors.Err(err)
	}

	n, err := store.Place(o.Object.Hash, w)

	if err != nil {
		return 0, errors.Err(err)
	}
	return int64(n), nil
}

func (s *ObjectStore) Placer(b *Build, aesgcm *crypto.AESGCM, kk []*Key) runner.Placer {
	chain := make([]*key.Key, 0, len(kk))

	for _, k := range kk {
		chain = append(chain, &key.Key{
			ID:     k.KeyID.Int64,
			Name:   k.Name,
			Key:    k.Key,
			Config: k.Config,
		})
	}

	return &placer{
		store:   s,
		chain:   key.NewChain(aesgcm, chain),
		userId:  b.UserID,
		buildId: b.ID,
	}
}
