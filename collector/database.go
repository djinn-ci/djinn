package collector

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"io"
	"os"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/runner"
)

type Database struct {
	runner.Collector

	Build *model.Build
}

func NewDatabase(c runner.Collector) *Database {
	return &Database{
		Collector: c,
	}
}

func (c *Database) Collect(name string, r io.Reader) error {
	md5buf := &bytes.Buffer{}
	sha256buf := &bytes.Buffer{}

	tee := io.TeeReader(r, io.MultiWriter(md5buf, sha256buf))

	wmd5 := md5.New()
	wsha256 := sha256.New()

	go func() {
		io.Copy(wmd5, md5buf)
		io.Copy(wsha256, sha256buf)
	}()

	if err := c.Collector.Collect(name, tee); err != nil {
		return errors.Err(err)
	}

	info, err := c.Collector.Stat(name)

	if err != nil {
		return errors.Err(err)
	}

	a, err := c.Build.ArtifactStore().FindByHash(name)

	if err != nil {
		return errors.Err(err)
	}

	a.Size = sql.NullInt64{
		Int64: info.Size(),
		Valid: true,
	}
	a.MD5 = md5buf.Bytes()
	a.SHA256 = sha256buf.Bytes()

	return errors.Err(a.Update())
}

func (c *Database) Stat(name string) (os.FileInfo, error) {
	info, err := c.Collector.Stat(name)

	return info, errors.Err(err)
}
