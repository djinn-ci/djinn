package filesystem

import (
	"io"
	"os"

	"github.com/andrewpillar/thrall/errors"
)

type Filesystem struct {}

func New() *Filesystem {
	return &Filesystem{}
}

func (*Filesystem) Collect(name string, r io.Reader) error {
	f, err := os.Create(name)

	if err != nil {
		return errors.Err(err)
	}

	defer f.Close()

	_, err = io.Copy(f, r)

	return errors.Err(err)
}
