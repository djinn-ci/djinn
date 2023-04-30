package runner

import (
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/andrewpillar/fs"

	"gopkg.in/yaml.v2"
)

func genfile(t *testing.T, name string) *os.File {
	f, err := os.Create(name)

	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 4096)

	if _, err := rand.Read(buf); err != nil {
		t.Fatal(err)
	}

	if _, err := f.Write(buf); err != nil {
		t.Fatal(err)
	}
	return f
}

func Test_Passthrough(t *testing.T) {
	var s struct {
		Objects Passthrough
	}

	in := []byte(`
objects:
- x/y => z
- a/b/c
- a/b/c/d => e
`)

	if err := yaml.Unmarshal(in, &s); err != nil {
		t.Fatal(err)
	}

	t.Log(s)

	out, err := yaml.Marshal(s)

	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(out))
}

func Test_Runner(t *testing.T) {
	genfile(t, "a")
	genfile(t, "c")

	dir, err := os.Getwd()

	if err != nil {
		t.Fatal(err)
	}

	r := Default(Passthrough{
		filepath.Join(dir, "a"): "b",
		filepath.Join(dir, "c"): "d",
	})

	r.Artifacts = fs.New("")
	r.Objects = fs.New("")

	defer func() {
		os.Remove(filepath.Join(dir, "a"))
		os.Remove(filepath.Join(dir, "c"))

		for _, name := range []string{"b", "d", "e"} {
			os.Remove(name)
		}
	}()

	stage := Stage{
		Name: "list",
	}

	stage.Add(&Job{
		Writer: os.Stdout,
		Commands: []string{
			"ls -l",
		},
		Artifacts: Passthrough{
			"d": "e",
		},
	})

	r.Add(&stage)

	d := OS{
		Writer: os.Stdout,
		Chdir: func() error {
			dir, err := os.MkdirTemp("", t.Name())

			if err != nil {
				return err
			}
			return os.Chdir(dir)
		},
	}

	if err := r.Run(context.Background(), d); err != nil {
		t.Fatal(err)
	}

	if status := r.Status(); status != Passed {
		t.Fatalf("unexpected status, expected=%s, got=%s\n", Passed, status)
	}
}
