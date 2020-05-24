package driver

import "testing"

func Test_Type(t *testing.T) {
	tests := []struct{
		val         []byte
		expected    Type
		shouldError bool
	}{
		{[]byte("ssh"), TypeSSH, false},
		{[]byte("qemu"), TypeQEMU, false},
		{[]byte("docker"), TypeDocker, false},
		{[]byte("foo"), Type(0), true},
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
