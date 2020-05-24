package oauth2

import (
	"bytes"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

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

type AppStore struct {
	model.Store

	User *user.User
}

var (
	_ model.Model  = (*App)(nil)
	_ model.Binder = (*AppStore)(nil)

	appTable = "oauth_apps"

	ErrAuth = errors.New("authentication failed")
)

// NewAppStore returns a new AppStore for querying the oauth_apps table. Each
// model passed to this function will be bound to the returned AppStore.
func NewAppStore(db *sqlx.DB, mm ...model.Model) *AppStore {
	s := &AppStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// AppModel is called along with model.Slice to convert the given slice of App
// models to a slice of model.Model interfaces.
func AppModel(aa []*App) func(int) model.Model {
	return func(i int) model.Model {
		return aa[i]
	}
}

// Bind the given models to the current Namespace. This will only bind the
// model if they are one of the following,
//
// - *user.User
func (a *App) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			a.User = m.(*user.User)
		}
	}
}

func (a *App) SetPrimary(id int64) {
	a.ID = id
}

func (a *App) Primary() (string, int64) { return "id", a.ID }

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

func (*App) JSON(_ string) map[string]interface{} { return map[string]interface{}{} }

// Endpoint returns the endpoint for the current App, and appends any of the
// given uri parts to the returned endpoint.
func (a *App) Endpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/settings/apps/%v", a.ID)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
}

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

// Bind the given models to the current Namespace. This will only bind the
// model if they are one of the following,
//
// - *user.User
func (s *AppStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		}
	}
}

// Create inserts the given App models into the oauth_apps table.
func (s *AppStore) Create(aa ...*App) error {
	mm := model.Slice(len(aa), AppModel(aa))
	return errors.Err(s.Store.Create(appTable, mm...))
}

// Update updates the given App models in the oauth_apps table.
func (s *AppStore) Update(aa ...*App) error {
	mm := model.Slice(len(aa), AppModel(aa))
	return errors.Err(s.Store.Update(appTable, mm...))
}

// Delete deletes the given App models from the oauth_apps table.
func (s *AppStore) Delete(aa ...*App) error {
	mm := model.Slice(len(aa), AppModel(aa))
	return errors.Err(s.Store.Delete(appTable, mm...))
}

// All returns a slice of App models, applying each query.Option that is
// given. The model.Where option is applied to the bound User model.
func (s *AppStore) All(opts ...query.Option) ([]*App, error) {
	aa := make([]*App, 0)

	opts = append([]query.Option{
		model.Where(s.User, "user_id"),
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

// Get returns a single App model, applying each query.Option that is given.
// The model.Where option is applied to the bound User model.
func (s *AppStore) Get(opts ...query.Option) (*App, error) {
	a := &App{
		User: s.User,
	}

	opts = append([]query.Option{
		model.Where(s.User, "user_id"),
	}, opts...)

	err := s.Store.Get(a, appTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return a, errors.Err(err)
}

// Auth finds the App model for the given client ID and checks that the given
// client secret matches what is in the database. If it matches, then the App
// model is returned. If authentication fails then ErrAuth is returned, if any
// other errors occur then they are wrapped via errors.Err.
func (s *AppStore) Auth(id, secret []byte) (*App, error) {
	a, err := s.Get(query.Where("client_id", "=", id))

	if err != nil {
		return a, errors.Err(err)
	}

	dec, _ := crypto.Decrypt(a.ClientSecret)

	if !bytes.Equal(dec, secret) {
		return a, ErrAuth
	}
	return a, nil
}
