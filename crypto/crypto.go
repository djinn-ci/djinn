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

	Key      []byte
	Alphabet string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func InitHashing(salt string, l int) error {
	var err error

	hd, err = hashids.NewWithData(&hashids.HashIDData{
		Alphabet:  Alphabet,
		MinLength: l,
		Salt:      salt,
	})

	return errors.Err(err)
}

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
		return nil, errors.Err(errors.New("cipher text is too short"))
	}

	nonce := b[:size]
	text := b[size:]

	d, err := gcm.Open(nil, nonce, text, nil)

	return d, errors.Err(err)
}

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

func Hash(i []int) (string, error) {
	if hd == nil {
		return "", errors.Err(errors.New("hashing not initialized"))
	}

	id, err := hd.Encode(i)

	return id, errors.Err(err)
}

// Create a hash of the current UNIX nano timestamp.
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
