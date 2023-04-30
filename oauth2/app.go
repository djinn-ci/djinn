package oauth2

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"

	"github.com/andrewpillar/query"
)

type App struct {
	loaded []string

	ID           int64
	UserID       int64
	ClientID     string
	ClientSecret database.Bytea
	Name         string
	Description  string
	HomeURI      string
	RedirectURI  string
	CreatedAt    time.Time

	User *auth.User
}

var _ database.Model = (*App)(nil)

func (a *App) Primary() (string, any) { return "id", a.ID }

func (a *App) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":            &a.ID,
		"user_id":       &a.UserID,
		"client_id":     &a.ClientID,
		"client_secret": &a.ClientSecret,
		"name":          &a.Name,
		"description":   &a.Description,
		"home_uri":      &a.HomeURI,
		"redirect_uri":  &a.RedirectURI,
		"created_at":    &a.CreatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (a *App) Params() database.Params {
	params := database.Params{
		"id":            database.ImmutableParam(a.ID),
		"user_id":       database.CreateOnlyParam(a.UserID),
		"client_id":     database.CreateUpdateParam(a.ClientID),
		"client_secret": database.CreateUpdateParam(a.ClientSecret),
		"name":          database.CreateUpdateParam(a.Name),
		"description":   database.CreateUpdateParam(a.Description),
		"home_uri":      database.CreateUpdateParam(a.HomeURI),
		"redirect_uri":  database.CreateUpdateParam(a.RedirectURI),
		"created_at":    database.CreateOnlyParam(a.CreatedAt),
	}

	if len(a.loaded) > 0 {
		params.Only(a.loaded...)
	}
	return params
}

func (_ *App) Bind(database.Model) {}

func (*App) MarshalJSON() ([]byte, error) { return nil, nil }

func (a *App) Endpoint(elems ...string) string {
	if len(elems) > 0 {
		return "/settings/apps/" + a.ClientID + "/" + strings.Join(elems, "/")
	}
	return "/settings/apps/" + a.ClientID
}

const appTable = "oauth_apps"

type AppStore struct {
	*database.Store[*App]

	AESGCM *crypto.AESGCM
}

func AppLoader(pool *database.Pool) database.Loader {
	return database.ModelLoader(pool, appTable, func() database.Model {
		return &App{}
	})
}

func NewAppStore(pool *database.Pool) *database.Store[*App] {
	return database.NewStore[*App](pool, appTable, func() *App {
		return &App{}
	})
}

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
	User        *auth.User
	Name        string
	Description string
	HomeURI     string
	RedirectURI string
}

func (s *AppStore) Create(ctx context.Context, p *AppParams) (*App, error) {
	secret, err := s.generateSecret()

	if err != nil {
		return nil, errors.Err(err)
	}

	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		return nil, errors.Err(err)
	}

	a := App{
		UserID:       p.User.ID,
		ClientID:     hex.EncodeToString(b),
		ClientSecret: secret,
		Name:         p.Name,
		Description:  p.Description,
		HomeURI:      p.HomeURI,
		RedirectURI:  p.RedirectURI,
		CreatedAt:    time.Now(),
	}

	if err := s.Store.Create(ctx, &a); err != nil {
		return nil, errors.Err(err)
	}
	return &a, nil
}

func (s *AppStore) Reset(ctx context.Context, a *App) error {
	secret, err := s.generateSecret()

	if err != nil {
		return errors.Err(err)
	}

	a.ClientSecret = secret

	if err := s.Update(ctx, a); err != nil {
		return errors.Err(err)
	}
	return nil
}

// Auth attempts to authenticate the OAuth app for the given client ID using
// the secret. If it matches, then the App is returned. If authentication
// fails, then ErrAuth is returned.
func (s *AppStore) Auth(ctx context.Context, id, secret string) (*App, error) {
	if s.AESGCM == nil {
		return nil, crypto.ErrNilAESGCM
	}

	realSecret, err := hex.DecodeString(secret)

	if err != nil {
		return nil, auth.ErrAuth
	}

	a, ok, err := s.Get(ctx, query.Where("client_id", "=", query.Arg(id)))

	if err != nil {
		return nil, errors.Err(err)
	}

	if !ok {
		return nil, database.ErrNoRows
	}

	dec, err := s.AESGCM.Decrypt(a.ClientSecret)

	if err != nil {
		return nil, errors.Err(err)
	}

	if !bytes.Equal(dec, realSecret) {
		return nil, auth.ErrAuth
	}
	return a, nil
}
