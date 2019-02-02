package collector

import (
	"io"
	"io/ioutil"

	"github.com/andrewpillar/thrall/errors"
)

var Discard = &discard{}

type discard struct {}

func (c *discard) Collect(name string, r io.Reader) error {
	_, err := io.Copy(ioutil.Discard, r)

	return errors.Err(err)
}
