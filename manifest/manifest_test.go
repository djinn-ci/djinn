package manifest

import (
	"reflect"
	"testing"

	"djinn-ci.com/errors"
)

func unmarshal(s string) func(interface{}) error {
	return func(dst interface{}) error {
		v := reflect.ValueOf(dst)

		if v.Kind() != reflect.Ptr {
			return errors.New("cannot unmarshal into non-pointer")
		}
		v.Elem().Set(reflect.ValueOf(s))
		return nil
	}
}

func Test_ManifestSource(t *testing.T) {
	tests := []struct {
		source      string
		expectedURL string
		expectedRef string
		expectedDir string
	}{
		{
			"https://github.com/git/git",
			"https://github.com/git/git",
			"",
			"git",
		},
		{
			"https://github.com/git/git v1.2.3",
			"https://github.com/git/git",
			"v1.2.3",
			"git",
		},
		{
			"https://github.com/git/git v1.2.3 => git-v1.2.3",
			"https://github.com/git/git",
			"v1.2.3",
			"git-v1.2.3",
		},
		{
			"https://github.com/git/git => git-build",
			"https://github.com/git/git",
			"",
			"git-build",
		},
	}

	for i, test := range tests {
		var s Source

		if err := s.UnmarshalYAML(unmarshal(test.source)); err != nil {
			t.Fatal(err)
		}

		if s.URL != test.expectedURL {
			t.Errorf("tests[%d](%v) - url mismatch, expected=%q, got=%q\n", i, test, test.expectedURL, s.URL)
		}
		if s.Ref != test.expectedRef {
			t.Errorf("tests[%d](%v) - ref mismatch, expected=%q, got=%q\n", i, test, test.expectedRef, s.Ref)
		}
		if s.Dir != test.expectedDir {
			t.Errorf("tests[%d](%v) - dir mismatch, expected=%q, got=%q\n", i, test, test.expectedDir, s.Dir)
		}
	}
}
