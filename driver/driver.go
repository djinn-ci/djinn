package driver

import (
	"bytes"
	"io"
	"fmt"

	"github.com/andrewpillar/thrall/runner"

	"github.com/pelletier/go-toml"
)

type ConfigureFunc func(io.Writer, *toml.Tree, ...Option) runner.Driver

type ValidatorFunc func(*toml.Tree) error

type Option func(runner.Driver) runner.Driver

var preamble = "#!/bin/sh\nexec 2>&1\nset -ex\n\n"

func CreateScript(j *runner.Job) *bytes.Buffer {
	buf := bytes.NewBufferString(preamble)

	for _, cmd := range j.Commands {
		fmt.Fprintf(buf, "%s\n", cmd)
	}
	return buf
}
