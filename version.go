// Invoked via "go run version.go" from make.sh.
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func git(name string, args ...string) string {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	cmd := exec.Command("git", append([]string{name}, args...)...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		io.Copy(os.Stderr, &stderr)
		os.Exit(1)
	}
	return strings.TrimSpace(stdout.String())
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: go run version.go [module]\n")
		os.Exit(1)
	}

	module := os.Args[1]

	build := git("rev-parse", "--abbrev-ref", "HEAD")

	if !strings.HasPrefix(build, "v") {
		build = fmt.Sprintf("devel %s", git("log", "-n", "1", "--format=format: +%h %cd", "HEAD"))
	}
	fmt.Printf("-X '%s/version.Build=%s'", module, build)
}
