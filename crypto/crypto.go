package crypto

import (
	"time"

	"github.com/andrewpillar/thrall/errors"

	"github.com/speps/go-hashids"
)

var (
	hd *hashids.HashID

	Alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
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
