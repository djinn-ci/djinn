package model

import (
	"database/sql"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
)

type Collaborator struct {
	Model

	NamespaceID	int64 `db:"namespace_id"`
	UserID      int64 `db:"user_id"`

	Namespace *Namespace
	User      *User
}

type CollaboratorStore struct {
	Store

	Namespace *Namespace
	User      *User
}

func (s CollaboratorStore) All(opts ...query.Option) ([]*Collaborator, error) {
	cc := make([]*Collaborator, 0)

	opts = append(opts, ForNamespace(s.Namespace), ForUser(s.User))

	err := s.Store.All(&cc, CollaboratorTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, c := range cc {
		c.DB = s.DB
		c.Namespace = s.Namespace
		c.User = s.User
	}

	return cc, errors.Err(err)
}
