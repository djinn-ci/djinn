package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
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

	ref := git("rev-parse", "HEAD")
	tag := git("rev-parse", "--abbrev-ref", "HEAD")

	if !strings.HasPrefix(tag, "v") {
		tag = fmt.Sprintf("devel %s", git("log", "-n", "1", "--format=format: +%h %cd", "HEAD"))
	}

	fmt.Printf("-X '%s/version.Ref=%s' ", module, ref)
	fmt.Printf("-X '%s/version.Tag=%s' ", module, tag)
	fmt.Printf("-X '%s/version.Os=%s' ", module, runtime.GOOS)
	fmt.Printf("-X '%s/version.Arch=%s'", module, runtime.GOARCH)
}
