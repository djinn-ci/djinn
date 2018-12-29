package runner

import "io"

type Collector interface {
	Collect(name string, r io.Reader) error
}
