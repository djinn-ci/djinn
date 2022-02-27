package integration

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"testing"

	"djinn-ci.com/build"
	"djinn-ci.com/integration/djinn"

	"github.com/mcmathja/curlyq"

	"github.com/vmihailenco/msgpack/v4"
)

func Test_BuildValidation(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	tests := []struct {
		params djinn.BuildParams
		errors []string
	}{
		{
			djinn.BuildParams{},
			[]string{"manifest"},
		},
		{
			djinn.BuildParams{
				Manifest: djinn.Manifest{
					Driver: map[string]string{"type": "docker"},
				},
			},
			[]string{"manifest"},
		},
		{
			djinn.BuildParams{
				Manifest: djinn.Manifest{
					Driver: map[string]string{"type": "docker", "image": "golang"},
				},
			},
			[]string{"manifest"},
		},
		{
			djinn.BuildParams{
				Manifest: djinn.Manifest{
					Driver: map[string]string{"type": "qemu", "image": "debian/stable"},
				},
			},
			[]string{"manifest"},
		},
	}

	for i, test := range tests {
		_, err := djinn.SubmitBuild(cli, test.params)

		if err == nil {
			t.Fatalf("tests[%d] - expected error, got nil\n", i)
		}

		djinnerr, ok := err.(*djinn.Error)

		if !ok {
			t.Fatalf("tests[%d] - unexpected error type, expected=%T, got=%T (%q)\n", i, djinn.Error{}, err, err)
		}

		if len(djinnerr.Params) != len(test.errors) {
			t.Fatalf("tests[%d] - unexpected error count, expected=%d, got=%d\nerrs = %v", i, len(test.errors), len(djinnerr.Params), djinnerr.Params)
		}

		for _, err := range test.errors {
			if _, ok := djinnerr.Params[err]; !ok {
				t.Fatalf("tests[%d] - could not find field %s in field errors\n", i, err)
			}
		}
	}
}

var buildQueue = "builds_docker:data"

func Test_BuildSubmit(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	tags := []string{"Test_BuildSubmit", "build_test"}

	b, err := djinn.SubmitBuild(cli, djinn.BuildParams{
		Manifest: djinn.Manifest{
			Driver: map[string]string{
				"type":      "docker",
				"image":     "golang",
				"workspace": "/go",
			},
		},
		Comment: "Test_BuildSubmit",
		Tags:    tags,
	})

	if err != nil {
		t.Fatal(err)
	}

	if len(b.Tags) != len(tags) {
		t.Fatalf("unexpected number of tags, expected=%d, got=%d\n", len(tags), len(b.Tags))
	}

	for i, tag := range tags {
		if b.Tags[i] != tag {
			t.Fatalf("tag does not match, expected=%q, got=%q\n", tag, b.Tags[i])
		}
	}

	m, err := redis.HGetAll(buildQueue).Result()

	if err != nil {
		t.Fatal(err)
	}

	var (
		qjob    curlyq.Job
		payload build.Payload
		found   bool
	)

	for _, v := range m {
		if err := msgpack.Unmarshal([]byte(v), &qjob); err != nil {
			t.Fatal(err)
		}

		if err := gob.NewDecoder(bytes.NewBuffer(qjob.Data)).Decode(&payload); err != nil {
			t.Fatal(err)
		}

		if payload.BuildID == b.ID {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("could not find build %d in queue %s\n", b.ID, buildQueue)
	}
}

func Test_BuildSubmitToNamespace(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	b, err := djinn.SubmitBuild(cli, djinn.BuildParams{
		Manifest: djinn.Manifest{
			Namespace: "submit/to/namespace",
			Driver: map[string]string{
				"type":      "docker",
				"image":     "golang",
				"workspace": "/go",
			},
		},
		Comment: "Test_BuildSubmitToNamespace",
		Tags:    []string{"build_test", "Test_BuildSubmitToNamespace"},
	})

	if err != nil {
		t.Fatal(err)
	}

	if b.Namespace.Name != "namespace" {
		t.Fatalf("unexpected namespace name, expected=%q, got=%q\n", "namespace", b.Namespace.Name)
	}

	if b.Namespace.Path != "submit/to/namespace" {
		t.Fatalf("unexpected namespace name, expected=%q, got=%q\n", "submit/to/namespace", b.Namespace.Name)
	}

	if _, err := djinn.GetNamespace(cli, b.Namespace.User.Username, b.Namespace.Path); err != nil {
		t.Fatal(err)
	}
}

func Test_BuildTags(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	b, err := djinn.SubmitBuild(cli, djinn.BuildParams{
		Manifest: djinn.Manifest{
			Namespace: "",
			Driver: map[string]string{
				"type":      "docker",
				"image":     "golang",
				"workspace": "/go",
			},
		},
		Comment: "Test_BuildTags",
		Tags:    []string{"tag1", "tag1", "tag2", "tag3"},
	})

	if err != nil {
		t.Fatal(err)
	}

	resp, err := cli.Get(b.TagsURL.Path, "application/json")

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	tt0 := make([]*djinn.BuildTag, 0)

	if err := json.NewDecoder(resp.Body).Decode(&tt0); err != nil {
		t.Fatal(err)
	}

	if l := len(tt0); l != 3 {
		t.Fatalf("unexpected number of tags on build, expected=%d, got=%d\n", 3, l)
	}

	for _, tag := range tt0 {
		if err := tag.Delete(cli); err != nil {
			t.Fatalf("failed to delete tag %q - %s\n", tag.Name, err)
		}
	}

	tt, err := b.Tag(cli, "build_test", "Test_BuildTags")

	if err != nil {
		t.Fatal(err)
	}

	for _, tag := range tt {
		if err := tag.Delete(cli); err != nil {
			t.Fatalf("failed to delete tag %q - %s\n", tag.Name, err)
		}
	}
}
