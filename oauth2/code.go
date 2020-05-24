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

// Code is the struct of the OAuth codes that are issued to generate the final
// token for authentication over the API.
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

// NewCodeStore returns a new CodeStore for querying the oauth_codes table. Each
// model passed to this function will be bound to the returned CodeStore.
func NewCodeStore(db *sqlx.DB, mm ...model.Model) *CodeStore {
	s := &CodeStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// CodeModel is called along with model.Slice to convert the given slice of
// Code models to a slice of model.Model interfaces.
func CodeModel(cc []*Code) func(int) model.Model {
	return func(i int) model.Model {
		return cc[i]
	}
}

// Bind the given models to the current Code. This will only bind the model if
// they are one of the following,
//
// - *app.App
// - *user.User
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

func (c *Code) SetPrimary(id int64) {
	c.ID = id
}

func (c *Code) Primary() (string, int64) { return "id", c.ID }

func (c *Code) IsZero() bool {
	return c == nil || c.ID == 0 &&
		len(c.Code) == 0 &&
		len(c.Scope) == 0 &&
		c.ExpiresAt == time.Time{}
}

func (*Code) JSON(_ string) map[string]interface{} { return map[string]interface{}{} }

// Endpoint is a stub to fulfill the model.Model interface. It returns an empty
// string.
func (*Code) Endpoint(_ ...string) string { return "" }

func (c *Code) Values() map[string]interface{} {
	return map[string]interface{}{
		"code":       c.Code,
		"scope":      c.Scope,
		"expires_at": c.ExpiresAt,
	}
}

// New returns a new Code binding any non-nil models to it from the current
// CodeStore.
func (s *CodeStore) New() *Code {
	c := &Code{
		User: s.User,
	}

	if s.User != nil {
		c.UserID = s.User.ID
	}
	return c
}

// Bind the given models to the current Code. This will only bind the model if
// they are one of the following,
//
// - *app.App
// - *user.User
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

// Create inserts the given Code models into the oauth_codes table.
func (s *CodeStore) Create(cc ...*Code) error {
	mm := model.Slice(len(cc), CodeModel(cc))
	return errors.Err(s.Store.Create(codeTable, mm...))
}

// Delete deletes the given Code models from the oauth_codes table.
func (s *CodeStore) Delete(cc ...*Code) error {
	mm := model.Slice(len(cc), CodeModel(cc))
	return errors.Err(s.Store.Delete(codeTable, mm...))
}

// Get returns a single Code model, applying each query.Option that is given.
// The model.Where option is applied to the bound User model and bound App
// model.
func (s *CodeStore) Get(opts ...query.Option) (*Code, error) {
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
