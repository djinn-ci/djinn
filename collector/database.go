package collector

import (
//	"bytes"
//	"crypto/md5"
//	"crypto/sha256"
//	"database/sql"
	"io"
//	"io/ioutil"

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
	return errors.Err(c.Collector.Collect(name, r))
//	// TODO: Don't store entire file in memory, should get info about it once
//	// collected.
//	buf := &bytes.Buffer{}
//	tee := io.TeeReader(r, buf)
//
//	if err := c.Collector.Collect(name, tee); err != nil {
//		return errors.Err(err)
//	}
//
//	md5hash := md5.Sum(buf.Bytes())
//	sha256hash := sha256.Sum256(buf.Bytes())
//
//	artifacts := c.Build.ArtifactStore()
//
//	a := artifacts.New()
//	a.Name = name
//	a.Size = sql.NullInt64{
//		Int64: int64(buf.Len()),
//		Valid: true,
//	}
//	a.MD5 = make([]byte, len(md5hash), len(md5hash))
//	a.SHA256 = make([]byte, len(sha256hash), len(sha256hash))
//
//	io.Copy(ioutil.Discard, buf)
//
//	for i, b := range md5hash {
//		a.MD5[i] = b
//	}
//
//	for i, b := range sha256hash {
//		a.SHA256[i] = b
//	}
//
//	return errors.Err(a.Create())
}
