package model

import (
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/types"
)

type Code struct {
	Model

	UserID    int64       `db:"user_id"`
	Code      []byte      `db:"code"`
	Scope     types.Scope `db:"scope"`
	ExpiresAt time.Time   `db:"expires_at"`

	User *User `db:"-"`
}

type CodeStore struct {
	Store

	User *User
}

func codeToInterface(cc []*Code) func(i int) Interface {
	return func(i int) Interface {
		return cc[i]
	}
}

func (c Code) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":    c.UserID,
		"code":       c.Code,
		"scope":      c.Scope,
		"expires_at": c.ExpiresAt,
		"created_at": c.CreatedAt,
		"updated_at": c.UpdatedAt,
	}
}

func (s CodeStore) Create(cc ...*Code) error {
	models := interfaceSlice(len(cc), codeToInterface(cc))

	return errors.Err(s.Store.Create(CodeTable, models...))
}

func (s CodeStore) New() *Code {
	c := &Code{
		Model: Model{
			DB: s.DB,
		},
		User: s.User,
	}

	if s.User != nil {
		c.UserID = s.User.ID
	}

	return c
}
