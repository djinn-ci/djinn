package collector

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

type Database struct {
	runner.Collector

	Build *model.Build
}

func NewDatabase(c runner.Collector) *Database {
	return &Database{
		Collector: c,
	}
}

func (c *Database) Collect(name string, r io.Reader) (int64, error) {
	hmd5 := md5.New()
	hsha256 := sha256.New()

	tee := io.TeeReader(r, io.MultiWriter(hmd5, hsha256))

	n, err := c.Collector.Collect(name, tee)

	if err != nil {
		return n, errors.Err(err)
	}

	a, err := c.Build.ArtifactStore().FindByHash(strings.TrimSuffix(name, ".tar"))

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

func (c *Database) Open(name string) (*os.File, error) {
	f, err := c.Collector.Open(name)

	return f, errors.Err(err)
}
