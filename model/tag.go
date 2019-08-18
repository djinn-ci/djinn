package model

import (
	"database/sql"
	"fmt"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
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
	return errors.Err(s.Store.Create(TagTable, s.interfaceSlice(tt...)...))
}

func (s TagStore) Delete(tt ...*Tag) error {
	return errors.Err(s.Store.Delete(TagTable, s.interfaceSlice(tt...)...))
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

func (s TagStore) interfaceSlice(tt ...*Tag) []Interface {
	ii := make([]Interface, len(tt), len(tt))

	for i, t := range tt {
		ii[i] = t
	}

	return ii
}

func (s TagStore) LoadUsers(tt []*Tag) error {
	if len(tt) == 0 {
		return nil
	}

	ids := make([]interface{}, len(tt), len(tt))

	for i, t := range tt {
		ids[i] = t.UserID
	}

	users := UserStore{
		Store: s.Store,
	}

	uu, err := users.All(query.WhereIn("id", ids...))

	if err != nil {
		return errors.Err(err)
	}

	for _, t := range tt {
		for _, u := range uu {
			if t.UserID == u.ID {
				t.User = u
			}
		}
	}

	return nil
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
