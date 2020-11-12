package oauth2

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"

	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// App is the type that represents an OAuth app created by a user.
type App struct {
	ID           int64     `db:"id"`
	UserID       int64     `db:"user_id"`
	ClientID     []byte    `db:"client_id"`
	ClientSecret []byte    `db:"client_secret"`
	Name         string    `db:"name"`
	Description  string    `db:"description"`
	HomeURI      string    `db:"home_uri"`
	RedirectURI  string    `db:"redirect_uri"`
	CreatedAt    time.Time `db:"created_at"`

	User *user.User `db:"-"`
}

// AppStore is the type for creating and modfiying App models in the database.
// The AppStore type can have an underlying crypto.Block for encrypting the
// App's secret.
type AppStore struct {
	database.Store

	block *crypto.Block

	// User is the bound user.User model. If not nil this will bind the
	// user.User model to any App models that are created. If not nil this
	// will append a WHERE clause on the user_id column for all SELECT queries
	// performed.
	User *user.User
}

var (
	_ database.Model  = (*App)(nil)
	_ database.Binder = (*AppStore)(nil)

	appTable = "oauth_apps"

	ErrAuth = errors.New("authentication failed")
)

// generateSecret generates a random 32 byte secret and encrypts it using the
// given encryption function.
func generateSecret(encrypt func([]byte) ([]byte, error)) ([]byte, error) {
	secret := make([]byte, 32)

	if _, err := rand.Read(secret); err != nil {
		return nil, errors.Err(err)
	}

	b, err := encrypt(secret)
	return b, errors.Err(err)
}

// NewAppStore returns a new AppStore for querying the oauth_apps table. Each
// database passed to this function will be bound to the returned AppStore.
func NewAppStore(db *sqlx.DB, mm ...database.Model) *AppStore {
	s := &AppStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// NewAppStoreWithBlock is functionally the same as NewAppStore, however it gets
// the crypto.Block to use on the returned AppStore. This will allow for
// encryption of the App's secret.
func NewAppStoreWithBlock(db *sqlx.DB, block *crypto.Block, mm ...database.Model) *AppStore {
	s := NewAppStore(db, mm...)
	s.block = block
	return s
}

// AppModel is called along with database.ModelSlice to convert the given slice of App
// models to a slice of database.Model interfaces.
func AppModel(aa []*App) func(int) database.Model {
	return func(i int) database.Model {
		return aa[i]
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if it is a pointer to a user.User model.
func (a *App) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *user.User:
			a.User = v
		}
	}
}

// SetPrimary implements the database.Model interface.
func (a *App) SetPrimary(id int64) { a.ID = id }

// Primary implements the database.Model interface.
func (a *App) Primary() (string, int64) { return "id", a.ID }

// IsZero implements the database.Model interface.
func (a *App) IsZero() bool {
	return a == nil || a.ID == 0 &&
		a.UserID == 0 &&
		len(a.ClientID) == 0 &&
		len(a.ClientSecret) == 0 &&
		a.Name == "" &&
		a.Description == "" &&
		a.HomeURI == "" &&
		a.RedirectURI == ""
}

// JSON implements the database.Model interface. This is a stub method and
// returns an empty map.
func (*App) JSON(_ string) map[string]interface{} { return map[string]interface{}{} }

// Endpoint returns the endpoint for the current App, and appends any of the
// given uri parts to the returned endpoint.
func (a *App) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/settings/apps/" + hex.EncodeToString(a.ClientID) + "/" + strings.Join(uri, "/")
	}
	return "/settings/apps/" + hex.EncodeToString(a.ClientID)
}

// Values implements the database.Model interface. This will return a map with
// the following values, user_id, client_id, client_secret, name, description,
// home_uri and redirect_uri.
func (a *App) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":       a.UserID,
		"client_id":     a.ClientID,
		"client_secret": a.ClientSecret,
		"name":          a.Name,
		"description":   a.Description,
		"home_uri":      a.HomeURI,
		"redirect_uri":  a.RedirectURI,
	}
}

// New returns a new App binding any non-nil models to it from the current
// AppStore.
func (s *AppStore) New() *App {
	a := &App{
		User: s.User,
	}

	if s.User != nil {
		a.UserID = s.User.ID
	}
	return a
}

// Bind implements the database.Binder interface. This will only bind the model
// if it is a pointer to a user.User model.
func (s *AppStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *user.User:
			s.User = v
		}
	}
}

// Create creates a new app with the given name and description, and homepage
// and redirect URIs. This will generate a random ID for the newly created App,
// and secret.
func (s *AppStore) Create(name, description, homepage, redirect string) (*App, error) {
	if s.block == nil {
		return nil, errors.New("nil block cipher")
	}

	secret, err := generateSecret(s.block.Encrypt)

	if err != nil {
		return nil, errors.Err(err)
	}

	a := s.New()
	a.ClientID = make([]byte, 16)
	a.ClientSecret = secret
	a.Name = name
	a.Description = description
	a.HomeURI = homepage
	a.RedirectURI = redirect

	if _, err := rand.Read(a.ClientID); err != nil {
		return nil, errors.Err(err)
	}

	err = s.Store.Create(appTable, a)
	return a, errors.Err(err)
}

// Reset generates a new secret for the App of the given id, and updates it in
// the database. This will error if the underlying crypto.Block is not set.
func (s *AppStore) Reset(id int64) error {
	if s.block == nil {
		return errors.New("nil block cipher")
	}

	secret, err := generateSecret(s.block.Encrypt)

	if err != nil {
		return errors.Err(err)
	}

	q := query.Update(
		appTable,
		query.Set("client_secret", query.Arg(secret)),
		query.Where("id", "=", query.Arg(id)),
	)

	_, err = s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Update updates the App of the given id, and updates the name, description,
// homepage, and redirect properties in the database to what is given.
func (s *AppStore) Update(id int64, name, description, homepage, redirect string) error {
	q := query.Update(
		appTable,
		query.Set("name", query.Arg(name)),
		query.Set("description", query.Arg(description)),
		query.Set("home_uri", query.Arg(homepage)),
		query.Set("redirect_uri", query.Arg(redirect)),
		query.Where("id", "=", query.Arg(id)),
	)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Delete delets the Apps of the given ids from the database.
func (s *AppStore) Delete(ids ...int64) error {
	vals := make([]interface{}, 0, len(ids))

	for _, id := range ids {
		vals = append(vals, id)
	}

	q := query.Delete(appTable, query.Where("id", "IN", query.List(vals...)))

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// All returns a slice of App models, applying each query.Option that is
// given. The database.Where option is applied to the bound User database.
func (s *AppStore) All(opts ...query.Option) ([]*App, error) {
	aa := make([]*App, 0)

	opts = append([]query.Option{
		database.Where(s.User, "user_id"),
	}, opts...)

	err := s.Store.All(&aa, appTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, a := range aa {
		a.User = s.User
	}
	return aa, errors.Err(err)
}

// Get returns a single App database, applying each query.Option that is given.
// The database.Where option is applied to the bound User database.
func (s *AppStore) Get(opts ...query.Option) (*App, error) {
	a := &App{
		User: s.User,
	}

	opts = append([]query.Option{
		database.Where(s.User, "user_id"),
	}, opts...)

	err := s.Store.Get(a, appTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return a, errors.Err(err)
}

// Auth finds the App database for the given client ID and checks that the given
// client secret matches what is in the database. If it matches, then the App
// database is returned. If authentication fails then ErrAuth is returned, if any
// other errors occur then they are wrapped via errors.Err.
func (s *AppStore) Auth(id, secret string) (*App, error) {
	realId, err := hex.DecodeString(id)

	if err != nil {
		return nil, ErrAuth
	}

	realSecret, err := hex.DecodeString(secret)

	if err != nil {
		return nil, ErrAuth
	}

	a, err := s.Get(query.Where("client_id", "=", query.Arg(realId)))

	if err != nil {
		return a, errors.Err(err)
	}

	dec, err := s.block.Decrypt(a.ClientSecret)

	if err != nil {
		return a, errors.Err(err)
	}

	if !bytes.Equal(dec, realSecret) {
		return a, ErrAuth
	}
	return a, nil
}
