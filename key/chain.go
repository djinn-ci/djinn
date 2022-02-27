package key

import (
	"bytes"
	"io"
	"os"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/errors"
)

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

	ErrKeyNotExist = errors.New("key does not exist")
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

// Get returns the decrypted key for the given name, and whether or not one
// could be found for that name.
func (c *Chain) Get(name string) ([]byte, bool, error) {
	b0, ok := c.keys[name]

	if !ok {
		return nil, false, nil
	}

	b, err := c.aesgcm.Decrypt(b0)

	if err != nil {
		return nil, false, errors.Err(err)
	}
	return b, true, nil
}

// Config returns the configuration set for the keys in the current chain.
func (c *Chain) Config() string { return c.config }

func (c *Chain) ConfigName() string { return configName }

func (c *Chain) Place(name string, w io.Writer) (int64, error) {
	b, ok, err := c.Get(name)

	if err != nil {
		return 0, errors.Err(err)
	}

	if !ok {
		return 0, ErrKeyNotExist
	}

	n, err := w.Write(b)

	if err != nil {
		return 0, errors.Err(err)
	}
	return int64(n), nil
}

type info struct {
	name    string
	size    int64
	modTime time.Time
}

var _ os.FileInfo = (*info)(nil)

func (i *info) Name() string       { return i.name }
func (i *info) Size() int64        { return i.size }
func (*info) Mode() os.FileMode    { return os.FileMode(0400) }
func (i *info) ModTime() time.Time { return i.modTime }
func (i *info) IsDir() bool        { return false }
func (i *info) Sys() interface{}   { return nil }

func (c *Chain) Stat(name string) (os.FileInfo, error) {
	b, ok, err := c.Get(name)

	if err != nil {
		return nil, errors.Err(err)
	}

	if !ok {
		return nil, ErrKeyNotExist
	}

	return &info{
		name:    name,
		size:    int64(len(b)),
		modTime: time.Now(),
	}, nil
}

func (c *Chain) ConfigFileInfo() os.FileInfo {
	return &info{
		name:    c.ConfigName(),
		size:    int64(len(c.config)),
		modTime: time.Now(),
	}
}
