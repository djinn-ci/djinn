package namespace

import (
	"testing"

	"github.com/andrewpillar/thrall/errors"
)

func Test_Visibility(t *testing.T) {
	tests := []struct{
		val         []byte
		expected    Visibility
		shouldError bool
	}{
		{[]byte("private"), Private, false},
		{[]byte("internal"), Internal, false},
		{[]byte("public"), Public, false},
		{[]byte("foo"), Visibility(0), true},
	}

	for _, test := range tests {
		var vis Visibility

		if err := vis.Scan(test.val); err != nil {
			if test.shouldError {
				continue
			}
			t.Fatal(errors.Cause(err))
		}
	}
}
