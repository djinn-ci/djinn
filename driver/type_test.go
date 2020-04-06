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

	for _, test := range tests {
		var typ Type

		if err := typ.Scan(test.val); err != nil {
			if test.shouldError {
				continue
			}
			t.Fatal(err)
		}

		if typ != test.expected {
			t.Errorf("mismatch driverType\n\texpected = '%s'\n\t  actual = '%s'\n", test.expected, typ)
		}
	}
}
