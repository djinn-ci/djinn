package build

import (
	"testing"

	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/errors"
)

func Test_BuildForm(t *testing.T) {
	tests := []struct {
		form        Form
		shouldError bool
	}{
		{
			Form{
				Manifest: config.Manifest{
					Driver: map[string]string{
						"type":      "docker",
						"image":     "golang",
						"workspace": "/go",
					},
				},
			},
			false,
		},
		{
			Form{
				Manifest: config.Manifest{
					Driver: map[string]string{
						"type": "docker",
					},
				},
			},
			true,
		},
		{
			Form{
				Manifest: config.Manifest{
					Driver: map[string]string{
						"type":      "docker",
						"workspace": "/go",
					},
				},
			},
			true,
		},
	}

	for i, test := range tests {
		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				continue
			}
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}
	}
}

func Test_TagForm(t *testing.T) {
	tests := []struct {
		form        TagForm
		shouldError bool
	}{
		{
			TagForm{
				Tags: tags([]string{"tag1", "tag2", "tag3"}),
			},
			false,
		},
		{
			TagForm{
				Tags: tags([]string{}),
			},
			false,
		},
	}

	for i, test := range tests {
		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				continue
			}
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}
	}
}
