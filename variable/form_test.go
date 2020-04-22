package variable

import (
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
)

func Test_FormValidate(t *testing.T) {
	tests := []struct{
		form        Form
		errs        []string
		shouldError bool
	}{
		{
			Form{Key: "PGADDR", Value: "postgres://root:secret@localhost/thrall"},
			[]string{},
			false,
		},
		{
			Form{Key: "0PGADDR", Value: "postgres://root:secret@localhost/thrall"},
			[]string{"key"},
			true,
		},
		{
			Form{Key: "0PGADDR", Value: ""},
			[]string{"key", "value"},
			true,
		},
		{
			Form{Key: "", Value: ""},
			[]string{"key", "value"},
			true,
		},
	}

	for _, test := range tests {
		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				if len(test.errs) == 0 {
					continue
				}

				ferrs, ok := err.(form.Errors)

				if !ok {
					t.Fatalf("expected error to be form.Errors, it was not\n%s\n", errors.Cause(err))
				}

				for _, err := range test.errs {
					if _, ok := ferrs[err]; !ok {
						t.Fatalf("expected field '%s' to be in form.Errors, it was not\n", err)
					}
				}
				continue
			}
			t.Fatal(errors.Cause(err))
		}
	}
}
