package crypto

import (
	"bytes"
	"testing"

	"github.com/andrewpillar/thrall/errors"
)

func Test_InitHashingHash(t *testing.T) {
	tests := []struct{
		salt        string
		length      int
		i           []int
		shouldInit  bool
		shouldHash  bool
	}{
		{"salt", 16, []int{1, 2, 3}, true, true},
		{"", 16, []int{1, 2, 3}, true, false},
	}

	for i, test := range tests {
		err := InitHashing(test.salt, test.length)

		if err != nil {
			if test.shouldInit {
				t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
			}
		}

		if _, err := Hash(test.i); err != nil {
			if test.shouldHash {
				t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
			}
		}
	}
}

func Test_EncryptDecrypt(t *testing.T) {
	tests := []struct{
		key         []byte
		b           []byte
		shouldError bool
	}{
		{[]byte("secret"), []byte("password"), true},
		{[]byte("secretkey16chars"), []byte("password"), false},
	}

	for i, test := range tests {
		Key = test.key

		encrypted, err := Encrypt(test.b)

		if err != nil {
			if test.shouldError {
				continue
			}
			t.Fatal(err)
		}

		decrypted, err := Decrypt(encrypted)

		if err != nil {
			t.Fatalf("test[%d] - %s\n", i, err)
		}

		if !bytes.Equal(test.b, decrypted) {
			t.Fatalf("test[%d] - expected = '%s actual = '%s'\n", i, string(test.b), string(decrypted))
		}
	}
}
