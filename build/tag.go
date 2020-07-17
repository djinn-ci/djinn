package build

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Tag is the type that represents a tag on a build.
type Tag struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	BuildID   int64     `db:"build_id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`

	User  *user.User `db:"-"`
	Build *Build     `db:"-"`
}

// TagStore is the type for creating and modifying Tag models in the database.
type TagStore struct {
	database.Store

	User  *user.User
	Build *Build
}

var (
	_ database.Model  = (*Tag)(nil)
	_ database.Binder = (*TagStore)(nil)
	_ database.Loader = (*TagStore)(nil)

	tagTable = "build_tags"
)

// NewTagStore returns a new TagStore for querying the build_tags table.
// Each database passed to this function will be bound to the returned TagStore.
func NewTagStore(db *sqlx.DB, mm ...database.Model) *TagStore {
	s := &TagStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// TagModel is called along with database.ModelSlice to convert the given slice of
// Tag models to a slice of database.Model interfaces.
func TagModel(tt []*Tag) func(int) database.Model {
	return func(i int) database.Model {
		return tt[i]
	}
}

// Bind implements the database.Binder interface. This will only bind the models
// if they are pointers to either Build or user.User models.
func (t *Tag) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			t.User = m.(*user.User)
		case *Build:
			t.Build = m.(*Build)
		}
	}
}

// Primary implements the database.Model interface.
func (t Tag) Primary() (string, int64) { return "id", t.ID }

// SetPrimary implements the database.Model interface.
func (t *Tag) SetPrimary(i int64) { t.ID = i }

// Endpoint implements the database.Model interface. If the current Tag has a
// nil or zero value Tag bound model then an emtpy string is returned,
// otherwise the full Build endpoint is returned, suffixed with the Tag
// endpoint, for example,
//
//   /b/l.belardo/10/tags/7
func (t Tag) Endpoint(uri ...string) string {
	if t.Build == nil || t.Build.IsZero() {
		return ""
	}
	return t.Build.Endpoint("tags", strconv.FormatInt(t.ID, 10))
}

// IsZero implements the database.Model interface.
func (t *Tag) IsZero() bool {
	return t == nil || t.ID == 0 &&
		t.UserID == 0 &&
		t.BuildID == 0 &&
		t.Name == "" &&
		t.CreatedAt == time.Time{}
}

// JSON implements the database.Model interface. This will return reutrn a map
// with the current Tag's values under each key. If the User or Build bound
// models are not zero, then the JSON representation of each will be in the
// returned map under the user and build keys respectively.
func (t *Tag) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":         t.ID,
		"user_id":    t.UserID,
		"build_id":   t.BuildID,
		"name":       t.Name,
		"created_at": t.CreatedAt.Format(time.RFC3339),
		"url":        addr + t.Endpoint(),
	}

	for name, m := range map[string]database.Model{
		"user":  t.User,
		"build": t.Build,
	} {
		if !m.IsZero() {
			json[name] = m.JSON(addr)
		}
	}
	return json
}

// Values implements the database.Model interface. This will return a map with
// the following values, user_id, build_id, and name.
func (t *Tag) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":  t.UserID,
		"build_id": t.BuildID,
		"name":     t.Name,
	}
}

// Create takes the given names, and creates a Tag for each for the given build
// ID, and user ID. A slice of the created Tag models are returned.
func (s *TagStore) Create(userId int64, names ...string) ([]*Tag, error) {
	if len(names) == 0 {
		return []*Tag{}, nil
	}

	tt := make([]*Tag, 0, len(names))

	for _, name := range names {
		t := s.New()
		t.UserID = userId
		t.Name = name

		tt = append(tt, t)
	}

	err := s.Store.Create(tagTable, database.ModelSlice(len(tt), TagModel(tt))...)
	return tt, errors.Err(err)
}

func (s *TagStore) Delete(ids ...int64) error {
	vals := make([]interface{}, 0, len(ids))

	for _, id := range ids {
		vals = append(vals, id)
	}

	q := query.Delete(query.From(tagTable), query.Where("id", "IN", vals...))

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// New returns a new Tag binding any non-nil models to it from the current
// TagStore.
func (s TagStore) New() *Tag {
	t := &Tag{
		User:  s.User,
		Build: s.Build,
	}

	if s.Build != nil {
		_, id := s.Build.Primary()
		t.BuildID = id
	}

	if s.User != nil {
		_, id := s.User.Primary()
		t.UserID = id
	}
	return t
}

// Bind implements the database.Binder interface. This will only bind the models
// if they are pointers to either Build or user.User models.
func (s *TagStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		case *Build:
			s.Build = m.(*Build)
		}
	}
}

// All returns a slice of Tag models, applying each query.Option that is given.
func (s TagStore) All(opts ...query.Option) ([]*Tag, error) {
	tt := make([]*Tag, 0)

	opts = append([]query.Option{
		database.Where(s.Build, "build_id"),
	}, opts...)

	err := s.Store.All(&tt, tagTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, t := range tt {
		t.Build = s.Build
	}
	return tt, errors.Err(err)
}

// Get returns a single Tag database, applying each query.Option that is given.
func (s TagStore) Get(opts ...query.Option) (*Tag, error) {
	t := &Tag{
		Build: s.Build,
		User:  s.User,
	}

	opts = append([]query.Option{
		database.Where(s.Build, "build_id"),
		database.Where(s.User, "user_id"),
	}, opts...)

	err := s.Store.Get(t, tagTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return t, errors.Err(err)
}

// Load loads in a slice of Job models where the given key is in the list of
// given vals. Each database is loaded individually via a call to the given load
// callback. This method calls JobStore.All under the hood, so any bound models
// will impact the models being loaded.
func (s TagStore) Load(key string, vals []interface{}, load database.LoaderFunc) error {
	tt, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, t := range tt {
			load(i, t)
		}
	}
	return nil
}
