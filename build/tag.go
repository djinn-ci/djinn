package build

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Tag struct {
	ID        int64     `db:"id"`
	UserID    int64     `db:"user_id"`
	BuildID   int64     `db:"build_id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`

	User  *user.User `db:"-"`
	Build *Build     `db:"-"`
}

type TagStore struct {
	model.Store

	User  *user.User
	Build *Build
}

var (
	_ model.Model  = (*Tag)(nil)
	_ model.Binder = (*TagStore)(nil)
	_ model.Loader = (*TagStore)(nil)

	tagTable = "build_tags"
)

// NewTagStore returns a new TagStore for querying the build_tags table.
// Each model passed to this function will be bound to the returned TagStore.
func NewTagStore(db *sqlx.DB, mm ...model.Model) *TagStore {
	s := &TagStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// TagModel is called along with model.Slice to convert the given slice of
// Tag models to a slice of model.Model interfaces.
func TagModel(tt []*Tag) func(int) model.Model {
	return func(i int) model.Model {
		return tt[i]
	}
}

// Bind the given models to the current Tag. This will only bind the model if
// they are one of the following,
//
// - *user.User
// - *Build
func (t *Tag) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			t.User = m.(*user.User)
		case *Build:
			t.Build = m.(*Build)
		}
	}
}

func (t Tag) Primary() (string, int64) {
	return "id", t.ID
}

func (t *Tag) SetPrimary(i int64) {
	t.ID = i
}

// Endpoint returns the endpoint for the current Tag. If nil, or if missing
// a bound Build model, then an empty string is returned.
func (t Tag) Endpoint(uri ...string) string {
	if t.Build == nil || t.Build.IsZero() {
		return ""
	}

	uri = append([]string{"tags", fmt.Sprintf("%v", t.ID)}, uri...)
	return t.Build.Endpoint(uri...)
}

func (t *Tag) IsZero() bool {
	return t == nil || t.ID == 0 &&
		t.UserID == 0 &&
		t.BuildID == 0 &&
		t.Name == "" &&
		t.CreatedAt == time.Time{}
}

func (t *Tag) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":  t.UserID,
		"build_id": t.BuildID,
		"name":     t.Name,
	}
}

// Create inserts the given Tag models into the build_tags table.
func (s TagStore) Create(tt ...*Tag) error {
	models := model.Slice(len(tt), TagModel(tt))
	return errors.Err(s.Store.Create(tagTable, models...))
}

// Delete removes the given Tag models from the build_tags table.
func (s TagStore) Delete(tt ...*Tag) error {
	models := model.Slice(len(tt), TagModel(tt))
	return errors.Err(s.Store.Delete(tagTable, models...))
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

// Bind the given models to the current Tag. This will only bind the model if
// they are one of the following,
//
// - *user.User
// - *Build
func (s *TagStore) Bind(mm ...model.Model) {
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
// The model.Where option is used on the Build bound model to limit the query
// to those relations.
func (s TagStore) All(opts ...query.Option) ([]*Tag, error) {
	tt := make([]*Tag, 0)

	opts = append([]query.Option{
		model.Where(s.Build, "build_id"),
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

// Get returns a single Tag model, applying each query.Option that is given.
// The model.Where option is used on the Build bound model to limit the query
// to those relations.
func (s TagStore) Get(opts ...query.Option) (*Tag, error) {
	t := &Tag{
		Build: s.Build,
		User:  s.User,
	}

	opts = append([]query.Option{
		model.Where(s.Build, "build_id"),
		model.Where(s.User, "user_id"),
	}, opts...)

	err := s.Store.Get(t, tagTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return t, errors.Err(err)
}

// Load loads in a slice of Job models where the given key is in the list of
// given vals. Each model is loaded individually via a call to the given load
// callback. This method calls JobStore.All under the hood, so any bound models
// will impact the models being loaded.
func (s TagStore) Load(key string, vals []interface{}, load model.LoaderFunc) error {
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
