package oauth2

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type App struct {
	ID           int64
	UserID       int64
	ClientID     string
	ClientSecret []byte
	Name         string
	Description  string
	HomeURI      string
	RedirectURI  string
	CreatedAt    time.Time

	User *user.User
}

var _ database.Model = (*App)(nil)

func (a *App) Dest() []interface{} {
	return []interface{}{
		&a.ID,
		&a.UserID,
		&a.ClientID,
		&a.ClientSecret,
		&a.Name,
		&a.Description,
		&a.HomeURI,
		&a.RedirectURI,
		&a.CreatedAt,
	}
}

func (a *App) Bind(m database.Model) {
	if v, ok := m.(*user.User); ok {
		if a.UserID == v.ID {
			a.User = v
		}
	}
}

func (*App) JSON(_ string) map[string]interface{} { return nil }

func (a *App) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/settings/apps/" + a.ClientID + "/" + strings.Join(uri, "/")
	}
	return "/settings/apps/" + a.ClientID
}

func (a *App) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":            a.ID,
		"user_id":       a.UserID,
		"client_id":     a.ClientID,
		"client_secret": a.ClientSecret,
		"name":          a.Name,
		"description":   a.Description,
		"home_uri":      a.HomeURI,
		"redirect_uri":  a.RedirectURI,
		"created_at":    a.CreatedAt,
	}
}

type AppStore struct {
	database.Pool

	AESGCM *crypto.AESGCM
}

var (
	_ database.Loader = (*AppStore)(nil)

	appTable = "oauth_apps"

	ErrAuth = errors.New("authentication failed")
)

// generateSecret generates and encryptes a random 32 byte secret.
func (s *AppStore) generateSecret() ([]byte, error) {
	if s.AESGCM == nil {
		return nil, crypto.ErrNilAESGCM
	}

	secret := make([]byte, 32)

	if _, err := rand.Read(secret); err != nil {
		return nil, errors.Err(err)
	}

	b, err := s.AESGCM.Encrypt(secret)

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

type AppParams struct {
	UserID      int64
	Name        string
	Description string
	HomeURI     string
	RedirectURI string
}

func (s *AppStore) Create(p AppParams) (*App, error) {
	secret, err := s.generateSecret()

	if err != nil {
		return nil, errors.Err(err)
	}

	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		return nil, errors.Err(err)
	}

	clientId := hex.EncodeToString(b)

	now := time.Now()

	q := query.Insert(
		appTable,
		query.Columns("user_id", "client_id", "client_secret", "name", "description", "home_uri", "redirect_uri", "created_at"),
		query.Values(p.UserID, clientId, secret, p.Name, p.Description, p.HomeURI, p.RedirectURI, now),
		query.Returning("id"),
	)

	var id int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}

	return &App{
		ID:           id,
		UserID:       p.UserID,
		ClientID:     clientId,
		ClientSecret: secret,
		Name:         p.Name,
		Description:  p.Description,
		HomeURI:      p.HomeURI,
		RedirectURI:  p.RedirectURI,
		CreatedAt:    now,
	}, nil
}

func (s *AppStore) Reset(id int64) error {
	secret, err := s.generateSecret()

	if err != nil {
		return errors.Err(err)
	}

	q := query.Update(
		appTable,
		query.Set("client_secret", query.Arg(secret)),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *AppStore) Update(id int64, p AppParams) error {
	q := query.Update(
		appTable,
		query.Set("name", query.Arg(p.Name)),
		query.Set("description", query.Arg(p.Description)),
		query.Set("home_uri", query.Arg(p.HomeURI)),
		query.Set("redirect_uri", query.Arg(p.RedirectURI)),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *AppStore) Delete(id int64) error {
	q := query.Delete(appTable, query.Where("id", "=", query.Arg(id)))

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *AppStore) Get(opts ...query.Option) (*App, bool, error) {
	var a App

	ok, err := s.Pool.Get(appTable, &a, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &a, ok, nil
}

func (s *AppStore) All(opts ...query.Option) ([]*App, error) {
	aa := make([]*App, 0)

	new := func() database.Model {
		a := &App{}
		aa = append(aa, a)
		return a
	}

	if err := s.Pool.All(appTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return aa, nil
}

// Auth attempts to authenticate the OAuth app for the given client ID using
// the secret. If it matches, then the App is returned. If authentication
// fails, then ErrAuth is returned.
func (s *AppStore) Auth(id, secret string) (*App, error) {
	if s.AESGCM == nil {
		return nil, crypto.ErrNilAESGCM
	}

	realSecret, err := hex.DecodeString(secret)

	if err != nil {
		return nil, ErrAuth
	}

	a, ok, err := s.Get(query.Where("client_id", "=", query.Arg(id)))

	if err != nil {
		return nil, errors.Err(err)
	}

	if !ok {
		return nil, database.ErrNotFound
	}

	dec, err := s.AESGCM.Decrypt(a.ClientSecret)

	if err != nil {
		return a, errors.Err(err)
	}

	if !bytes.Equal(dec, realSecret) {
		return a, ErrAuth
	}
	return a, nil
}

func (s *AppStore) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	aa, err := s.All(query.Where(pk, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	for _, a := range aa {
		for _, m := range mm {
			m.Bind(a)
		}
	}
	return nil
}
