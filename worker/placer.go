package worker

import (
	"io"
	"os"
	"strings"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/errors"
	"djinn-ci.com/runner"
)

type info struct {
	name    string
	size    int64
	modTime time.Time
}

type placer struct {
	block  *crypto.Block
	keycfg []byte
	keys   map[string][]byte
	placer runner.Placer
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

func (p *placer) Place(name string, w io.Writer) (int64, error) {
	if name == "/root/.ssh/config" {
		n, err := w.Write(p.keycfg)
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

	n, err := p.placer.Place(name, w)
	return int64(n), errors.Err(err)
}

func (p *placer) Stat(name string) (os.FileInfo, error) {
	if strings.HasPrefix(name, "keycfg:") {
		return &info{
			name:    "/root/.ssh/config",
			size:    int64(len(p.keycfg)),
			modTime: time.Now(),
		}, nil
	}

	if strings.HasPrefix(name, "key:") {
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

	info, err := p.placer.Stat(name)
	return info, errors.Err(err)
}
