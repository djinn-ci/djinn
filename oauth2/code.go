package oauth2

import (
	"database/sql"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Code struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	AppID     int64     `db:"app_id"`
	Code      []byte    `db:"code"`
	Scope     Scope     `db:"scope"`
	ExpiresAt time.Time `db:"expires_at"`

	User *user.User `db:"-"`
	App  *App       `db:"-"`
}

type CodeStore struct {
	model.Store

	User *user.User
	App  *App
}

var (
	_ model.Model  = (*Code)(nil)
	_ model.Binder = (*CodeStore)(nil)

	codeTable = "oauth_codes"
)

func NewCodeStore(db *sqlx.DB, mm ...model.Model) CodeStore {
	s := CodeStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func CodeModel(cc []*Code) func(int) model.Model {
	return func(i int) model.Model {
		return cc[i]
	}
}

func (c *Code) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *App:
			c.App = m.(*App)
		case *user.User:
			c.User = m.(*user.User)
		}
	}
}

func (c *Code) Kind() string { return "oauth_code" }

func (c *Code) SetPrimary(id int64) {
	if c == nil {
		return
	}
	c.ID = id
}

func (c *Code) Primary() (string, int64) {
	if c == nil {
		return "id", 0
	}
	return "id", c.ID
}

func (c *Code) IsZero() bool {
	return c == nil || c.ID == 0 &&
		len(c.Code) == 0 &&
		len(c.Scope) == 0 &&
		c.ExpiresAt == time.Time{}
}

func (*Code) Endpoint(_ ...string) string { return "" }

func (c *Code) Values() map[string]interface{} {
	if c == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"code":       c.Code,
		"scope":      c.Scope,
		"expires_at": c.ExpiresAt,
	}
}

func (s CodeStore) New() *Code {
	c := &Code{
		User: s.User,
	}

	if s.User != nil {
		c.UserID = s.User.ID
	}
	return c
}

func (s *CodeStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *App:
			s.App = m.(*App)
		case *user.User:
			s.User = m.(*user.User)
		}
	}
}

func (s CodeStore) Create(cc ...*Code) error {
	mm := model.Slice(len(cc), CodeModel(cc))
	return errors.Err(s.Store.Create(codeTable, mm...))
}

func (s CodeStore) Delete(cc ...*Code) error {
	mm := model.Slice(len(cc), CodeModel(cc))
	return errors.Err(s.Store.Delete(codeTable, mm...))
}

func (s CodeStore) Get(opts ...query.Option) (*Code, error) {
	c := &Code{
		User: s.User,
		App:  s.App,
	}

	opts = append([]query.Option{
		model.Where(s.User, "user_id"),
		model.Where(s.App, "app_id"),
	}, opts...)

	err := s.Store.Get(c, codeTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return c, errors.Err(err)
}
