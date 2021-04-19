// Package crypto implements functions for hashing, encrypting, and decrypting
// data.
//
// Encryption, and decryption uses the underlying crypto/aes, crypto/cipher,and
// crypto/rand packages from the stdlib.
//
// Hashing is supported via use of the speps/go-hashids library.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"time"

	"djinn-ci.com/errors"

	"github.com/speps/go-hashids"

	"golang.org/x/crypto/pbkdf2"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Hasher is the type for generating hash ids based on a sequence of numbers,
// based on the specified alphabet and length.
type Hasher struct {
	hashid *hashids.HashID

	// Salt is the additional data to use when generating a hash.
	Salt string

	// Length is the minimum length of hashes that should be generated.
	Length int

	// Alphabet is the sequence of characters that should be chosen from when a
	// hash is being generated. If empty then the default a-zA-Z0-9 alphabet
	// will be used.
	Alphabet string
}

// Block is the type for encrypting and decrypting payloads of data. This uses
// the AES block cipher, wrapped in Galois Counter Mode.
type Block struct {
	gcm cipher.AEAD
}

// CheckCSPRNG will see if it's possible to generate a cryptographically secure
// pseudorandom number. If not then this will panic.
func CheckCSPRNG() {
	b := make([]byte, 1)

	if _, err := rand.Read(b); err != nil {
		panic("csprng not available: " + err.Error())
	}
}

// NewBlock returns a new block for encrypting and authenticating payloads
// of data. The given password and salt are used to generate a key for
// encryption.
func NewBlock(password, salt []byte) (*Block, error) {
	key := pbkdf2.Key(password, salt, 4096, 32, sha256.New)

	ciph, err := aes.NewCipher(key)

	if err != nil {
		return nil, errors.Err(err)
	}

	gcm, err := cipher.NewGCM(ciph)

	if err != nil {
		return nil, errors.Err(err)
	}

	return &Block{
		gcm: gcm,
	}, nil
}

// Decrypt returns the decrypted bytes of the given payload.
func (b *Block) Decrypt(p []byte) ([]byte, error) {
	size := b.gcm.NonceSize()

	if len(p) < size {
		return nil, errors.New("cipher text is too short")
	}

	nonce := p[:size]
	text := p[size:]

	d, err := b.gcm.Open(nil, nonce, text, nil)
	return d, errors.Err(err)
}

// Encrypt returns the encrypted bytes of the given payload.
func (b *Block) Encrypt(p []byte) ([]byte, error) {
	nonce := make([]byte, b.gcm.NonceSize())

	if _, err := rand.Read(nonce); err != nil {
		return nil, errors.Err(err)
	}
	return b.gcm.Seal(nonce, nonce, p, nil), nil
}

// Init initializes the hasher for hasing of numbers.
func (h *Hasher) Init() error {
	var err error

	if h.Alphabet == "" {
		h.Alphabet = alphabet
	}

	h.hashid, err = hashids.NewWithData(&hashids.HashIDData{
		Alphabet:  h.Alphabet,
		MinLength: h.Length,
		Salt:      h.Salt,
	})
	return errors.Err(err)
}

// Hash returns the hash of the given numbers. This will return an error if the
// Hasher has not yet been initialized, or any other underlying errors from
// hashing.
func (h *Hasher) Hash(i ...int) (string, error) {
	if h.hashid == nil {
		return "", errors.New("hasher not initialized")
	}

	id, err := h.hashid.Encode(i)
	return id, errors.Err(err)
}

// HashNow returns the hash of the current UNIX nano timestamp. This will
// return an error if the Hash has not yet been initialized, or any other
// underlying errors from hashing.
func (h *Hasher) HashNow() (string, error) {
	if h.hashid == nil {
		return "", errors.New("hasher not initialized")
	}

	i := make([]int, 0)
	now := time.Now().UnixNano()

	for now != 0 {
		i = append(i, int(now%10))
		now /= 10
	}

	id, err := h.Hash(i...)
	return id, errors.Err(err)
}
