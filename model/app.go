package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"

	"github.com/andrewpillar/query"
)

type App struct {
	Model

	UserID       int64  `db:"user_id"`
	ClientID     []byte `db:"client_id"`
	ClientSecret []byte `db:"client_secret"`
	Name         string `db:"name"`
	Description  string `db:"description"`
	Domain       string `db:"domain"`
	HomeURI      string `db:"home_uri"`
	RedirectURI  string `db:"redirect_uri"`

	User *User `db:"-"`
}

type AppStore struct {
	Store

	User *User
}

func appToInterface(aa []*App) func(i int) Interface {
	return func(i int) Interface {
		return aa[i]
	}
}

func (a App) Values() map[string]interface{} {
	return map[string]interface{}{
	}
}

func (s AppStore) All(opts ...query.Option) ([]*App, error) {
	aa := make([]*App, 0)

	opts = append([]query.Option{ForUser(s.User)}, opts...)

	err := s.Store.All(&aa, AppTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, a := range aa {
		a.DB = s.DB
	}

	return aa, errors.Err(err)
}

func (s AppStore) Create(aa ...*App) error {
	models := interfaceSlice(len(aa), appToInterface(aa))

	return errors.Err(s.Store.Create(AppTable, models...))
}

func (s AppStore) Get(opts ...query.Option) (*App, error) {
	a := &App{
		Model: Model{
			DB: s.DB,
		},
		User: s.User,
	}

	baseOpts := []query.Option{
		query.Columns("*"),
		query.From(AppTable),
		ForUser(s.User),
	}

	q := query.Select(append(baseOpts, opts...)...)

	err := s.Store.Get(a, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return a, errors.Err(err)
}

func (s AppStore) New() *App {
	a := &App{
		Model: Model{
			DB: s.DB,
		},
		User: s.User,
	}

	if s.User != nil {
		a.UserID = s.User.ID
	}

	return a
}
