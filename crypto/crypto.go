// Package crypto implements functions for hashing, encrypting, and decrypting
// data.
//
// Encryption, and decyption uses the underlying crypto/aes, crypto/cipher,and
// crypto/rand packages from the stdlib.
//
// Hashing is supported via use of the speps/go-hashids library.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"time"

	"github.com/andrewpillar/thrall/errors"

	"github.com/speps/go-hashids"
)

var (
	hd *hashids.HashID

	// Key specifies the key to use for encryption.
	Key []byte

	// Alphabet specifies the characters to use for hashing.
	Alphabet string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// InitHashing initializes hashing with the given salt and minimum hash length.
func InitHashing(salt string, l int) error {
	var err error

	hd, err = hashids.NewWithData(&hashids.HashIDData{
		Alphabet:  Alphabet,
		MinLength: l,
		Salt:      salt,
	})
	return errors.Err(err)
}

// Decrypt decrypts the given bytes using the key specified via Key.
func Decrypt(b []byte) ([]byte, error) {
	ciph, err := aes.NewCipher(Key)

	if err != nil {
		return nil, errors.Err(err)
	}

	gcm, err := cipher.NewGCM(ciph)

	if err != nil {
		return nil, errors.Err(err)
	}

	size := gcm.NonceSize()

	if len(b) < size {
		return nil, errors.New("cipher text is too short")
	}

	nonce := b[:size]
	text := b[size:]

	d, err := gcm.Open(nil, nonce, text, nil)
	return d, errors.Err(err)
}

// Encrypt encrypts the given bytes using the key specified via Key.
func Encrypt(b []byte) ([]byte, error) {
	ciph, err := aes.NewCipher(Key)

	if err != nil {
		return nil, errors.Err(err)
	}

	gcm, err := cipher.NewGCM(ciph)

	if err != nil {
		return nil, errors.Err(err)
	}

	nonce := make([]byte, gcm.NonceSize())

	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errors.Err(err)
	}
	return gcm.Seal(nonce, nonce, b, nil), nil
}

// Hash returns a hash of the given integers.
func Hash(i []int) (string, error) {
	if hd == nil {
		return "", errors.New("hashing not initialized")
	}

	id, err := hd.Encode(i)
	return id, errors.Err(err)
}

// HashNow returns a hash of the current UNIX nano timestamp.
func HashNow() (string, error) {
	i := make([]int, 0)

	now := time.Now().UnixNano()

	for now != 0 {
		i = append(i, int(now % 10))
		now /= 10
	}

	h, err := Hash(i)
	return h, errors.Err(err)
}
