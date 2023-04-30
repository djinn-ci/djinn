package key

import (
	"bytes"
	"io"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/errors"

	"github.com/andrewpillar/fs"
)

type keyFile struct {
	name    string
	off     int64
	data    []byte
	modTime time.Time
}

func (f *keyFile) Stat() (fs.FileInfo, error) { return f, nil }

func (f *keyFile) Read(p []byte) (int, error) {
	if f.off < 0 {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: fs.ErrInvalid}
	}
	if f.off >= int64(len(f.data)) {
		return 0, io.EOF
	}

	n := copy(p, f.data[f.off:])
	f.off += int64(n)

	return n, nil
}

func (*keyFile) Close() error         { return nil }
func (f *keyFile) Name() string       { return f.name }
func (f *keyFile) Size() int64        { return int64(len(f.data)) }
func (f *keyFile) Mode() fs.FileMode  { return fs.FileMode(0400) }
func (f *keyFile) ModTime() time.Time { return f.modTime }
func (*keyFile) IsDir() bool          { return false }
func (*keyFile) Sys() any             { return nil }

type Chain struct {
	aesgcm *crypto.AESGCM
	config string
	keys   map[string][]byte
}

var (
	configName = "/root/.ssh/config"

	configPreamble = `
StrictHostKeyChecking no
UserKnownHostsFile /dev/null

`

	ErrNotExist = errors.New("key: not in chain")
)

func NewChain(aesgcm *crypto.AESGCM, kk []*Key) *Chain {
	config := bytes.NewBufferString(configPreamble)

	keys := make(map[string][]byte)

	for _, k := range kk {
		keys[k.Name] = k.Key
		config.WriteString(k.Config + "\n")
	}

	return &Chain{
		aesgcm: aesgcm,
		keys:   keys,
		config: config.String(),
	}
}

func (c *Chain) Open(name string) (fs.File, error) {
	if name == configName {
		return &keyFile{
			name:    name,
			data:    []byte(c.config),
			modTime: time.Now(),
		}, nil
	}

	enc, ok := c.keys[name]

	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}

	b, err := c.aesgcm.Decrypt(enc)

	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}

	return &keyFile{
		name:    name,
		data:    b,
		modTime: time.Now(),
	}, nil
}

func (c *Chain) Sub(string) (fs.FS, error) { return c, nil }

func (c *Chain) Stat(name string) (fs.FileInfo, error) {
	f, err := c.Open(name)

	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: errors.Unwrap(err)}
	}

	info, err := f.Stat()

	if err != nil {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: err}
	}
	return info, nil
}

func (c *Chain) Put(fs.File) (fs.File, error) { return nil, fs.ErrPermission }
func (c *Chain) Remove(string) error          { return fs.ErrPermission }

// Config returns the configuration set for the keys in the current chain.
func (c *Chain) Config() string { return c.config }

func (c *Chain) ConfigName() string { return configName }
