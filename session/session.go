// Package session provides an implementation of the sessions.Session interface
// for Redis.
package session

import (
	"bytes"
	"encoding/base32"
	"encoding/gob"
	"net/http"
	"strings"
	"time"

	"github.com/andrewpillar/djinn/errors"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"

	"github.com/go-redis/redis"
)

var (
	prefix    = "session_"
	expire    = 86400 * 30
	maxAge    = 60 * 30
	maxLength = 4096
)

type Store struct {
	client *redis.Client

	Codecs    []securecookie.Codec
	Options   *sessions.Options
	IsNew     bool
	MaxAge    int
	MaxLength int
}

func deserialize(b []byte, sess *sessions.Session) error {
	dec := gob.NewDecoder(bytes.NewBuffer(b))
	return dec.Decode(&sess.Values)
}

func serialize(sess *sessions.Session) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)

	if err := enc.Encode(sess.Values); err != nil {
		return nil, errors.Err(err)
	}
	return buf.Bytes(), nil
}

// New returns a new sessions.Store implementation fo Redis using the given
// redis.Client, and the given keyPairs for the secure cookie.
func New(client *redis.Client, keyPairs ...[]byte) *Store {
	return &Store{
		client: client,
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: expire,
		},
		MaxAge:    maxAge,
		MaxLength: maxLength,
	}
}

// Get returns the named session from the given *http.Request.
func (s *Store) Get(r *http.Request, name string) (*sessions.Session, error) {
	sess, err := sessions.GetRegistry(r).Get(s, name)
	return sess, errors.Err(err)
}

// New returns a new session for the given request with the given name.
func (s *Store) New(r *http.Request, name string) (*sessions.Session, error) {
	sess := sessions.NewSession(s, name)

	options := *s.Options

	sess.Options = &options
	sess.IsNew = true

	c, err := r.Cookie(name)

	if err != nil {
		return sess, errors.Err(err)
	}

	if err := securecookie.DecodeMulti(name, c.Value, &sess.ID, s.Codecs...); err != nil {
		return sess, errors.Err(err)
	}

	data, err := s.client.Get(prefix + sess.ID).Result()

	if err != nil {
		return sess, errors.Err(err)
	}

	if data != "" {
		if err := deserialize([]byte(data), sess); err != nil {
			return sess, errors.Err(err)
		}
		sess.IsNew = false
	}
	return sess, errors.Err(err)
}

// Save saves the given session, updating the cookie that stores the session
// data in the given request and response.
func (s *Store) Save(r *http.Request, w http.ResponseWriter, sess *sessions.Session) error {
	if sess.Options.MaxAge <= 0 {
		if _, err := s.client.Del(prefix + sess.ID).Result(); err != nil {
			return errors.Err(err)
		}

		http.SetCookie(w, sessions.NewCookie(sess.Name(), "", sess.Options))
		return nil
	}

	if sess.ID == "" {
		key := base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32))

		sess.ID = strings.TrimRight(key, "=")
	}

	b, err := serialize(sess)

	if err != nil {
		return errors.Err(err)
	}

	if s.MaxLength != 0 && len(b) > s.MaxLength {
		return errors.New("session value too big")
	}

	duration := time.Duration(time.Second * time.Duration(s.MaxAge))

	_, err = s.client.Set(prefix+sess.ID, b, duration).Result()

	if err != nil {
		return errors.Err(err)
	}

	encoded, err := securecookie.EncodeMulti(sess.Name(), sess.ID, s.Codecs...)

	if err != nil {
		return errors.Err(err)
	}

	http.SetCookie(w, sessions.NewCookie(sess.Name(), encoded, sess.Options))
	return nil
}
