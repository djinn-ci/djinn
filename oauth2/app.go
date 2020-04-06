package oauth2

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"

	"golang.org/x/crypto/bcrypt"
)

type App struct {
	ID           int64  `db:"id"`
	UserID       int64  `db:"user_id"`
	ClientID     []byte `db:"client_id"`
	ClientSecret []byte `db:"client_secret"`
	Name         string `db:"name"`
	Description  string `db:"description"`
	Domain       string `db:"domain"`
	HomeURI      string `db:"home_uri"`
	RedirectURI  string `db:"redirect_uri"`

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

func NewAppStore(db *sqlx.DB, mm ...model.Model) AppStore {
	s := AppStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func AppModel(aa []*App) func(int) model.Model {
	return func(i int) model.Model {
		return aa[i]
	}
}

func (a *App) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			a.User = m.(*user.User)
		}
	}
}

func (a *App) Kind() string { return "oauth2_app" }

func (a *App) SetPrimary(id int64) {
	if a == nil {
		return
	}
	a.ID = id
}

func (a *App) Primary() (string, int64) {
	if a == nil {
		return "id", 0
	}
	return "id", a.ID
}

func (a *App) IsZero() bool {
	return a == nil || a.ID == 0 &&
		a.UserID == 0 &&
		len(a.ClientID) == 0 &&
		len(a.ClientSecret) == 0 &&
		a.Name == "" &&
		a.Description == "" &&
		a.Domain == "" &&
		a.HomeURI == "" &&
		a.RedirectURI == ""
}

func (a *App) Endpoint(_ ...string) string { return "" }

func (a *App) Values() map[string]interface{} {
	if a == nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"user_id":       a.UserID,
		"client_id":     a.ClientID,
		"client_secret": a.ClientSecret,
		"name":          a.Name,
		"description":   a.Description,
		"domain":        a.Domain,
		"home_uri":      a.HomeURI,
		"redirect_uri":  a.RedirectURI,
	}
}

func (s AppStore) New() *App {
	a := &App{
		User: s.User,
	}

	if s.User != nil {
		a.UserID = s.User.ID
	}
	return a
}

func (s *AppStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		}
	}
}

func (s AppStore) Create(aa ...*App) error {
	mm := model.Slice(len(aa), AppModel(aa))
	return errors.Err(s.Store.Create(appTable, mm...))
}

func (s AppStore) Update(aa ...*App) error {
	mm := model.Slice(len(aa), AppModel(aa))
	return errors.Err(s.Store.Update(appTable, mm...))
}

func (s AppStore) Delete(aa ...*App) error {
	mm := model.Slice(len(aa), AppModel(aa))
	return errors.Err(s.Store.Delete(appTable, mm...))
}

func (s AppStore) All(opts ...query.Option) ([]*App, error) {
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

func (s AppStore) Get(opts ...query.Option) (*App, error) {
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

func (s AppStore) Auth(id, secret []byte) (*App, error) {
	a, err := s.Get(query.Where("client_id", "=", id))

	if err != nil {
		return a, errors.Err(err)
	}

	if err := bcrypt.CompareHashAndPassword(a.ClientSecret, secret); err != nil {
		return a, ErrAuth
	}
	return a, nil
}
