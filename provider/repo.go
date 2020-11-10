package provider

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Repo is the type that represents a Repo from a remote Git hosting provider.
type Repo struct {
	ID           int64         `db:"id"`
	UserID       int64         `db:"user_id"`
	ProviderID   int64         `db:"provider_id"`
	HookID       sql.NullInt64 `db:"hook_id"`
	RepoID       int64         `db:"repo_id"`
	ProviderName string        `db:"provider_name"`
	Enabled      bool          `db:"enabled"`
	Name         string        `db:"name"`
	Href         string        `db:"href"`

	User     *user.User `db:"-"`
	Provider *Provider  `db:"-"`
}

// RepoStore is the type for creating and modifying Repo models in the database.
type RepoStore struct {
	database.Store

	// User is the bound user.User model. If not nil this will bind the
	// user.User model to any Repo models that are created. If not nil this
	// will append a WHERE clause on the user_id column for all SELECT queries
	// performed.
	User *user.User

	// Provider is the bound Provider model. If not nil this will bind the
	// Provider model to any Repo models that are created. If not nil this will
	// append a WHERE clause on the provider_id column for all SELECT queries
	// performed.
	Provider *Provider
}

var (
	_ database.Model  = (*Repo)(nil)
	_ database.Binder = (*RepoStore)(nil)

	repoTable = "provider_repos"
	relations = map[string]database.RelationFunc{
		"user":     database.Relation("user_id", "id"),
		"provider": database.Relation("provider_id", "id"),
	}
)

// NewRepoStore returns a new RepoStore for querying the provider_repos table. Each
// database passed to this function will be bound to the returned RepoStore.
func NewRepoStore(db *sqlx.DB, mm ...database.Model) *RepoStore {
	s := &RepoStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// RepoFromContext returns the Repo model from the given context, if any.
func RepoFromContext(ctx context.Context) (*Repo, bool) {
	r, ok := ctx.Value("repo").(*Repo)
	return r, ok
}

// RepoModel is called along with database.ModelSlice to convert the given
// slice of Repo models to a slice of database.Model interfaces.
func RepoModel(rr []*Repo) func(int) database.Model {
	return func(i int) database.Model {
		return rr[i]
	}
}

// Bind implements the database.Binder interface. This will only bind the model
// if it is a pointer to either a user.User model or Provider model.
func (r *Repo) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			r.User = m.(*user.User)
		case *Provider:
			r.Provider = m.(*Provider)
		}
	}
}

// SetPrimary implements the database.Model interface.
func (r *Repo) SetPrimary(id int64) { r.ID = id }

// Primary implements the database.Model interface.
func (r *Repo) Primary() (string, int64) { return "id", r.ID }

// IsZero implements the database.Model interface.
func (r *Repo) IsZero() bool {
	return r == nil || r.ID == 0 &&
		r.UserID == 0 &&
		r.ProviderID == 0 &&
		!r.HookID.Valid &&
		r.RepoID == 0 &&
		!r.Enabled
}

// JSON implements the database.Model interface. This is a stub method and
// returns an empty map.
func (*Repo) JSON(_ string) map[string]interface{} { return map[string]interface{}{} }

// Endpoint returns the endpoint to the current Repo, and appends any of the
// given URI parts.
func (r *Repo) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/repos/" + strconv.FormatInt(r.ID, 10) + "/" + strings.Join(uri, "/")
	}
	return "/repos/" + strconv.FormatInt(r.ID, 10)
}

// Values implements the database.Model interface. This will return a map with
// the following values, user_id, provider_id, hook_id, repo_id, and enabled.
func (r *Repo) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":       r.UserID,
		"provider_id":   r.ProviderID,
		"hook_id":       r.HookID,
		"repo_id":       r.RepoID,
		"provider_name": r.ProviderName,
		"enabled":       r.Enabled,
		"name":          r.Name,
		"href":          r.Href,
	}
}

// New returns a new Repo binding any non-nil models to it from the current
// RepoStore.
func (s *RepoStore) New() *Repo {
	r := &Repo{
		User:     s.User,
		Provider: s.Provider,
	}

	if s.User != nil {
		r.UserID = s.User.ID
	}

	if s.Provider != nil {
		r.ProviderID = s.Provider.ID
	}
	return r
}

// Touch will create the given Repo if it does not exist, otherwise it will
// update it.
func (s *RepoStore) Touch(r *Repo) error {
	if r.ID == 0 {
		r1 := s.New()

		r.UserID = r1.UserID
		r.ProviderID = r1.ProviderID

		return errors.Err(s.Store.Create(repoTable, r))
	}
	return errors.Err(s.Store.Update(repoTable, r))
}

// Update updates the given Repo models in the providers table.
func (s *RepoStore) Update(rr ...*Repo) error {
	mm := database.ModelSlice(len(rr), RepoModel(rr))
	return errors.Err(s.Store.Update(repoTable, mm...))
}

// Delete deletes the repos from the database with the given ids.
func (s *RepoStore) Delete(ids ...int64) error {
	vals := make([]interface{}, 0, len(ids))

	for _, id := range ids {
		vals = append(vals, id)
	}

	q := query.Delete(repoTable, query.Where("id", "IN", query.List(vals...)))

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Bind implements the database.Binder interface. This will only bind the model
// if it is a pointer to either a user.User model or Provider model.
func (s *RepoStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		case *Provider:
			s.Provider = m.(*Provider)
		}
	}
}

// Get returns a single Repo database, applying each query.Option that is given.
// The database.Where option is applied to the *user.User and *Provider
// bound models.
func (s *RepoStore) Get(opts ...query.Option) (*Repo, error) {
	r := &Repo{
		User:     s.User,
		Provider: s.Provider,
	}

	opts = append([]query.Option{
		database.Where(s.User, "user_id"),
		database.Where(s.Provider, "provider_id"),
	}, opts...)

	err := s.Store.Get(r, repoTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return r, errors.Err(err)
}

// All returns a slice Repo models, applying each query.Option that is given.
// The database.Where option is applied to the *user.User and *Provider
// bound models.
func (s *RepoStore) All(opts ...query.Option) ([]*Repo, error) {
	rr := make([]*Repo, 0)

	opts = append([]query.Option{
		database.Where(s.User, "user_id"),
		database.Where(s.Provider, "provider_id"),
	}, opts...)

	err := s.Store.All(&rr, repoTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return rr, errors.Err(err)
}
