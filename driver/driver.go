package driver

import (
	"bytes"
	"fmt"

	"github.com/andrewpillar/thrall/runner"
)

var preamble = "#!/bin/sh\nexec 2>&1\nset -ex\n\n"

func CreateScript(j *runner.Job) *bytes.Buffer {
	buf := bytes.NewBufferString(preamble)

	for _, cmd := range j.Commands {
		fmt.Fprintf(buf, "%s\n", cmd)
	}
	return buf
}
