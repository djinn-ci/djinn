package main

import (
	"io"
	"os"
	"strings"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"

	"github.com/jmoiron/sqlx"
)

type placer struct {
	db      *sqlx.DB
	build   *build.Build
	objects runner.Placer
}

var _ runner.Placer = (*placer)(nil)

// Place will write the contents of either the given object or key to the given
// io.Writer. If the name is prefixed with "key:" then the underlying placer
// will attempt to place the key by the given name, otherwise it will default
// to placing an object. Once the object has been placed, the corresponding
// build.Object model will be updated.
func (p *placer) Place(name string, w io.Writer) (int64, error) {
	var pl runner.Placer = build.NewObjectStoreWithPlacer(p.db, p.objects, p.build)

	if strings.HasPrefix("key:", name) {
		name = strings.SplitN(name, ":", 2)[1]
		pl = build.NewKeyStore(p.db, p.build)
	}

	n, err := pl.Place(name, w)
	return n, errors.Err(err)
}

// Stat will return the os.FileInfo for the given object or key. If the name is
// prefixed with "key:", then the information regarding the SSH key will be
// returned.
func (p *placer) Stat(name string) (os.FileInfo, error) {
	var pl runner.Placer = build.NewObjectStoreWithPlacer(p.db, p.objects, p.build)

	if strings.HasPrefix("key:", name) {
		name = strings.SplitN(name, ":", 2)[1]
		pl = build.NewKeyStore(p.db, p.build)
	}

	info, err := pl.Stat(name)
	return info, errors.Err(err)
}
