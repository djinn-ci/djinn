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

func NewTagStore(db *sqlx.DB, mm ...model.Model) TagStore {
	s := TagStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func TagModel(tt []*Tag) func(int) model.Model {
	return func(i int) model.Model {
		return tt[i]
	}
}

func (t *Tag) Bind(mm ...model.Model) {
	if t == nil {
		return
	}

	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			t.User = m.(*user.User)
		case *Build:
			t.Build = m.(*Build)
		}
	}
}

func (*Tag) Kind() string { return "build_tag" }

func (t *Tag) Primary() (string, int64) {
	if t == nil {
		return "id", 0
	}
	return "id", t.ID
}

func (t *Tag) SetPrimary(i int64) {
	if t == nil {
		return
	}
	t.ID = i
}

func (t *Tag) Endpoint(uri ...string) string {
	if t == nil {
		return ""
	}

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
	if t == nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"user_id":  t.UserID,
		"build_id": t.BuildID,
		"name":     t.Name,
	}
}

func (s TagStore) Create(tt ...*Tag) error {
	models := model.Slice(len(tt), TagModel(tt))
	return errors.Err(s.Store.Create(tagTable, models...))
}

func (s TagStore) Delete(tt ...*Tag) error {
	models := model.Slice(len(tt), TagModel(tt))
	return errors.Err(s.Store.Delete(tagTable, models...))
}

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
