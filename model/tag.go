package model

import (
	"database/sql"
	"fmt"

	"github.com/andrewpillar/thrall/errors"

	"github.com/andrewpillar/query"
)

type Tag struct {
	Model

	UserID  int64  `db:"user_id"`
	BuildID int64  `db:"build_id"`
	Name    string `db:"name"`

	User  *User
	Build *Build
}

type TagStore struct {
	Store

	User  *User
	Build *Build
}

func tagToInterface(tt []*Tag) func(i int) Interface {
	return func(i int) Interface {
		return tt[i]
	}
}

func (t Tag) UIEndpoint(uri ...string) string {
	if t.Build == nil || t.Build.IsZero() {
		return ""
	}

	uri = append([]string{"tags", fmt.Sprintf("%v", t.ID)}, uri...)

	return t.Build.UIEndpoint(uri...)
}

func (t Tag) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":  t.UserID,
		"build_id": t.BuildID,
		"name":     t.Name,
	}
}

func (s TagStore) All(opts ...query.Option) ([]*Tag, error) {
	tt := make([]*Tag, 0)

	opts = append(opts, ForBuild(s.Build))

	err := s.Store.All(&tt, TagTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, t := range tt {
		t.DB = s.DB

		if s.Build != nil {
			t.Build = s.Build
		}
	}

	return tt, errors.Err(err)
}

func (s TagStore) Create(tt ...*Tag) error {
	models := interfaceSlice(len(tt), tagToInterface(tt))

	return errors.Err(s.Store.Create(TagTable, models...))
}

func (s TagStore) Delete(tt ...*Tag) error {
	models := interfaceSlice(len(tt), tagToInterface(tt))

	return errors.Err(s.Store.Delete(TagTable, models...))
}

func (s TagStore) Find(id int64) (*Tag, error) {
	t := &Tag{
		Model: Model{
			DB: s.DB,
		},
		Build: s.Build,
		User:  s.User,
	}

	err := s.FindBy(t, TagTable, "id", id)

	if err == sql.ErrNoRows {
		err = nil
	}

	return t, errors.Err(err)
}

func (s TagStore) Index(opts ...query.Option) ([]*Tag, error) {
	tt, err := s.All(opts...)

	if err != nil {
		return tt, errors.Err(err)
	}

	err = s.LoadUsers(tt)

	return tt, errors.Err(err)
}

func (s TagStore) loadUser(tt []*Tag) func(i int, u *User) {
	return func(i int, u *User) {
		t := tt[i]

		if t.UserID == u.ID {
			t.User = u
		}
	}
}

func (s TagStore) LoadUsers(tt []*Tag) error {
	if len(tt) == 0 {
		return nil
	}

	models := interfaceSlice(len(tt), tagToInterface(tt))

	users := UserStore{
		Store: s.Store,
	}

	err := users.Load(mapKey("user_id", models), s.loadUser(tt))

	return errors.Err(err)
}

func (s TagStore) New() *Tag {
	t := &Tag{
		Model: Model{
			DB: s.DB,
		},
		User:  s.User,
		Build: s.Build,
	}

	if s.Build != nil {
		t.BuildID = s.Build.ID
	}

	if s.User != nil {
		t.UserID = s.User.ID
	}

	return t
}
