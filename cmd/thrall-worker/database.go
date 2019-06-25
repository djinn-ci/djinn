package main

import (
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"io"
	"os"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/runner"
)

type database struct {
	runner.Collector
	runner.Placer

	build *model.Build
	users *model.UserStore
}

func (d *database) findObject(name string) (*model.Object, error) {
	u, err := d.users.Find(d.build.UserID)

	if err != nil {
		return nil, errors.Err(err)
	}

	o, err := u.ObjectStore().FindByName(name)

	return o, errors.Err(err)
}

func (d *database) Collect(name string, r io.Reader) (int64, error) {
	hmd5 := md5.New()
	hsha256 := sha256.New()

	tee := io.TeeReader(r, io.MultiWriter(hmd5, hsha256))

	n, err := d.Collector.Collect(name, tee)

	if err != nil {
		return n, errors.Err(err)
	}

	a, err := d.build.ArtifactStore().FindByHash(strings.TrimSuffix(name, ".tar"))

	if err != nil {
		return n, errors.Err(err)
	}

	a.Size = sql.NullInt64{
		Int64: n,
		Valid: true,
	}
	a.MD5 = hmd5.Sum(nil)
	a.SHA256 = hsha256.Sum(nil)

	return n, errors.Err(a.Update())
}

func (d *database) Place(name string, w io.Writer) (int64, error) {
	o, err := d.findObject(name)

	if err != nil {
		return 0, errors.Err(err)
	}

	if o.IsZero() {
		return 0, errors.Err(errors.New("could not find object '" + name + "'"))
	}

	bo, err := o.BuildObjectStore().First()

	if err != nil {
		return 0, errors.Err(err)
	}

	n, placeErr := d.Placer.Place(name, w)

	bo.Placed = placeErr == nil

	if err := bo.Update(); err != nil {
		return n, errors.Err(err)
	}

	return n, errors.Err(placeErr)
}

func (d *database) Stat(name string) (os.FileInfo, error) {
	o, err := d.findObject(name)

	if err != nil {
		return nil, errors.Err(err)
	}

	if o.IsZero() {
		return nil, errors.Err(errors.New("could not find object '" + name + "'"))
	}

	info, err := d.Placer.Stat(o.Hash)

	return info, errors.Err(err)
}
