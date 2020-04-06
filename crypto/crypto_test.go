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

	for _, test := range tests {
		err := InitHashing(test.salt, test.length)

		if err != nil {
			if test.shouldInit {
				t.Fatal(errors.Cause(err))
			}
		}

		if _, err := Hash(test.i); err != nil {
			if test.shouldHash {
				t.Fatal(errors.Cause(err))
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

	for _, test := range tests {
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
			t.Fatal(err)
		}

		if !bytes.Equal(test.b, decrypted) {
			t.Fatalf("decryption mismatch\n\texpected = '%s'\n\t   actual = '%s'\n", string(test.b), string(decrypted))
		}
	}
}
