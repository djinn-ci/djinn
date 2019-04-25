package collector

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"io"
	"io/ioutil"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/runner"
)

type Database struct {
	runner.Collector

	Build *model.Build
}

func NewDatabase(c runner.Collector) *Database {
	return &Database{Collector: c}
}

func (c *Database) Collect(name string, r io.Reader) error {
	// TODO: Don't store entire file in memory, should get info about it once
	// collected.
	buf := &bytes.Buffer{}
	tee := io.TeeReader(r, buf)

	if err := c.Collector.Collect(name, tee); err != nil {
		return errors.Err(err)
	}

	md5hash := md5.Sum(buf.Bytes())
	sha256hash := sha256.Sum256(buf.Bytes())

	a := model.Artifact{
		BuildID: c.Build.ID,
		Name:    name,
		Size:   sql.NullInt64{
			Int64: int64(buf.Len()),
			Valid: true,
		},
		MD5:    make([]byte, len(md5hash), len(md5hash)),
		SHA256: make([]byte, len(sha256hash), len(sha256hash)),
	}

	io.Copy(ioutil.Discard, buf)

	for i, b := range md5hash {
		a.MD5[i] = b
	}

	for i, b := range sha256hash {
		a.SHA256[i] = b
	}

	err := a.Create()

	return errors.Err(err)
}
