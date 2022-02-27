package driver

import "testing"

func Test_Type(t *testing.T) {
	tests := []struct {
		val         string
		expected    Type
		shouldError bool
	}{
		{"ssh", SSH, false},
		{"qemu", QEMU, false},
		{"docker", Docker, false},
		{"foo", Type(0), true},
	}

	for i, test := range tests {
		var typ Type

		if err := typ.Scan(test.val); err != nil {
			if test.shouldError {
				continue
			}
			t.Fatal(err)
		}

		if typ != test.expected {
			t.Errorf("test[%d] - expected = '%s' actual = '%s'\n", i, test.expected, typ)
		}
	}
}
