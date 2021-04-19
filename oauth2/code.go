package oauth2

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Code is the struct of the OAuth codes that are issued to generate the final
// token for authentication over the API.
type Code struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	AppID     int64     `db:"app_id"`
	Code      string    `db:"code"`
	Scope     Scope     `db:"scope"`
	ExpiresAt time.Time `db:"expires_at"`

	User *user.User `db:"-"`
	App  *App       `db:"-"`
}

// CodeStore is the type for creating and modifying Code models in the
// database.
type CodeStore struct {
	database.Store

	// User is the bound user.User model. If not nil this will bind the
	// user.User model to any Code models that are created. If not nil this
	// will append a WHERE clause on the user_id column for all SELECT queries
	// performed.
	User *user.User

	// App is the bound App model. If not nil this will bind the App model to
	// any Code models that are created. If not nil this will append a WHERE
	// clause on the app_id column for all SELECT queries performed.
	App *App
}

var (
	_ database.Model  = (*Code)(nil)
	_ database.Binder = (*CodeStore)(nil)

	codeTable = "oauth_codes"
)

// NewCodeStore returns a new CodeStore for querying the oauth_codes table. Each
// database passed to this function will be bound to the returned CodeStore.
func NewCodeStore(db *sqlx.DB, mm ...database.Model) *CodeStore {
	s := &CodeStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// CodeModel is called along with database.ModelSlice to convert the given slice of
// Code models to a slice of database.Model interfaces.
func CodeModel(cc []*Code) func(int) database.Model {
	return func(i int) database.Model {
		return cc[i]
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if it is a pointer to either a user.User model or an App model.
func (c *Code) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *App:
			c.App = v
		case *user.User:
			c.User = v
		}
	}
}

// SetPrimary implements the database.Model interface.
func (c *Code) SetPrimary(id int64) { c.ID = id }

// Primary implements the database.Model interface.
func (c *Code) Primary() (string, int64) { return "id", c.ID }

// IsZero implements the database.Model interface.
func (c *Code) IsZero() bool {
	return c == nil || c.ID == 0 &&
		len(c.Code) == 0 &&
		len(c.Scope) == 0 &&
		c.ExpiresAt == time.Time{}
}

// JSON implements the database.Model interface. This is a stub method and
// returns an empty map.
func (*Code) JSON(_ string) map[string]interface{} { return map[string]interface{}{} }

// Endpoint implements the database.Model interface. This is a stub method and
// returns an empty string.
func (*Code) Endpoint(_ ...string) string { return "" }

// Values implements the database.Model interface. This will return a map with
// the following values, code, scope, and expires_at.
func (c *Code) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":    c.UserID,
		"app_id":     c.AppID,
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
		App:  s.App,
	}

	if s.User != nil {
		c.UserID = s.User.ID
	}
	if s.App != nil {
		c.AppID = s.App.ID
	}
	return c
}

// Bind implements the database.Binder interface. This will only bind the model
// if it is a pointer to either a user.User model or an App model.
func (s *CodeStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *App:
			s.App = v
		case *user.User:
			s.User = v
		}
	}
}

// Create creates a new code with the given scope. This will set the code's
// expiration to 10 minutes from when this is called.
func (s *CodeStore) Create(scope Scope) (*Code, error) {
	code := make([]byte, 16)

	if _, err := rand.Read(code); err != nil {
		return nil, errors.Err(err)
	}

	c := s.New()
	c.Code = hex.EncodeToString(code)
	c.Scope = scope
	c.ExpiresAt = time.Now().Add(time.Minute * 10)

	err := s.Store.Create(codeTable, c)
	return c, errors.Err(err)
}

// Delete deletes the codes of the given ids from the database.
func (s *CodeStore) Delete(ids ...int64) error {
	mm := make([]database.Model, 0, len(ids))

	for _, id := range ids {
		mm = append(mm, &Code{ID: id})
	}
	return errors.Err(s.Store.Delete(codeTable, mm...))
}

// Get returns a single Code database, applying each query.Option that is given.
// The database.Where option is applied to the bound User database and bound App
// database.
func (s *CodeStore) Get(opts ...query.Option) (*Code, error) {
	c := &Code{
		User: s.User,
		App:  s.App,
	}

	opts = append([]query.Option{
		database.Where(s.User, "user_id"),
		database.Where(s.App, "app_id"),
	}, opts...)

	err := s.Store.Get(c, codeTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return c, errors.Err(err)
}
