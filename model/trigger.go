package model

import (
	"database/sql"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/types"

	"github.com/andrewpillar/query"
)

type Trigger struct {
	Model

	BuildID int64             `db:"build_id"`
	Type    types.Trigger     `db:"type"`
	Comment string            `db:"comment"`
	Data    types.TriggerData `db:"data"`

	Build *Build
}

type TriggerStore struct {
	Store

	Build *Build
}

func triggerToInterface(tt []*Trigger) func(i int) Interface {
	return func(i int) Interface {
		return tt[i]
	}
}

func (t Trigger) CommentTitle() string {
	if len(t.Comment) < 72 {
		return t.Comment
	}

	title := t.Comment[:72]

	if strings.Index(t.Comment, "\n") > 72 {
		return title + "..."
	}

	return title
}

func (t Trigger) CommentBody() string {
	if len(t.Comment) < 72 {
		return ""
	}

	body := t.Comment[72:]

	if strings.Index(t.Comment, "\n") > 72 {
		return "..." + body
	}

	return body
}

func (t Trigger) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id": t.BuildID,
		"type":     t.Type,
		"comment":  t.Comment,
		"data":     t.Data,
	}
}

func (s TriggerStore) All(opts ...query.Option) ([]*Trigger, error) {
	tt := make([]*Trigger, 0)

	err := s.Store.All(&tt, TriggerTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, t := range tt {
		t.DB = s.DB
		t.Build = t.Build
	}

	return tt, errors.Err(err)
}

func (s TriggerStore) Create(tt ...*Trigger) error {
	models := interfaceSlice(len(tt), triggerToInterface(tt))

	return errors.Err(s.Store.Create(TriggerTable, models...))
}

func (s TriggerStore) First() (*Trigger, error) {
	t := &Trigger{
		Model: Model{
			DB: s.DB,
		},
		Build: s.Build,
	}

	q := query.Select(
		query.Columns("*"),
		query.From(TriggerTable),
		ForBuild(s.Build),
	)

	err := s.Store.Get(t, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return t, errors.Err(err)
}

func (s TriggerStore) New() *Trigger {
	t := &Trigger{
		Model: Model{
			DB: s.DB,
		},
		Build: s.Build,
	}

	if s.Build != nil {
		t.BuildID = s.Build.ID
	}

	return t
}
