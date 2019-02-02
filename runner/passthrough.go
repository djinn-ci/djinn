package runner

import "io"

type Passthrough interface {
	Place(name string, w io.Writer) error

	Collect(name string, r io.Reader) error
}
