package collector

import (
	"io"
	"io/ioutil"

	"github.com/andrewpillar/thrall/errors"
)

var Discard = &collector{}

type Collector interface {
	Collect(name string, r io.Reader) error
}

type collector struct {}

func (c *collector) Collect(name string, r io.Reader) error {
	_, err := io.Copy(ioutil.Discard, r)

	return errors.Err(err)
}
