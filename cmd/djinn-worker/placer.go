package main

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/runner"

	"github.com/jmoiron/sqlx"
)

type placer struct {
	db      *sqlx.DB
	block   *crypto.Block
	build   *build.Build
	keycfg  string
	keys    map[string][]byte
	objects runner.Placer
}

// info implements the os.FileInfo interface so we can use it for returning
// calls to Stat on keys and keycfg.
type info struct {
	name    string
	size    int64
	modTime time.Time
}

var (
	_ runner.Placer = (*placer)(nil)
	_ os.FileInfo   = (*info)(nil)
)

func (i *info) Name() string       { return i.name }
func (i *info) Size() int64        { return i.size }
func (*info) Mode() os.FileMode    { return os.FileMode(0600) }
func (i *info) ModTime() time.Time { return i.modTime }
func (i *info) IsDir() bool        { return false }
func (i *info) Sys() interface{}   { return nil }

// Place will write the contents of either the given object or key to the given
// io.Writer. If the name is prefixed with "key:" then the underlying placer
// will attempt to place the key by the given name, otherwise it will default
// to placing an object. Once the object has been placed, the corresponding
// build.Object model will be updated.
func (p *placer) Place(name string, w io.Writer) (int64, error) {
	var pl runner.Placer = build.NewObjectStoreWithPlacer(p.db, p.objects, p.build)

	if strings.HasPrefix(name, "keycfg:") {
		n, err := w.Write([]byte(p.keycfg))
		return int64(n), errors.Err(err)
	}

	if strings.HasPrefix(name, "key:") {
		enc, ok := p.keys[name]

		if !ok {
			return 0, nil
		}

		dec, err := p.block.Decrypt(enc)

		if err != nil {
			return 0, errors.Err(err)
		}

		n, err := w.Write(dec)
		return int64(n), errors.Err(err)
	}

	n, err := pl.Place(name, w)
	return int64(n), errors.Err(err)
}

// Stat will return the os.FileInfo for the given object or key. If the name is
// prefixed with "key:", then the information regarding the SSH key will be
// returned.
func (p *placer) Stat(name string) (os.FileInfo, error) {
	var pl runner.Placer = build.NewObjectStoreWithPlacer(p.db, p.objects, p.build)

	if strings.HasPrefix(name, "keycfg:") {
		return &info{
			name:    "/root/.ssh/config",
			size:    int64(len(p.keycfg)),
			modTime: time.Now(),
		}, nil
	}

	if strings.HasPrefix("key:", name) {
		k, ok := p.keys[name]

		if !ok {
			return nil, errors.New("no such key")
		}
		return &info{
			name:    name,
			size:    int64(len(k)),
			modTime: time.Now(),
		}, nil
	}

	info, err := pl.Stat(name)
	return info, errors.Err(err)
}
