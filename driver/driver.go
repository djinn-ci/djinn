package driver

import (
	"bytes"
	"fmt"

	"github.com/andrewpillar/thrall/runner"
)

var preamble = `#!/bin/sh

set -ex

`

func createScript(j *runner.Job) *bytes.Buffer {
	buf := bytes.NewBufferString(preamble)

	for _, cmd := range j.Commands {
		fmt.Fprintf(buf, "%s\n", cmd)
	}

	return buf
}
