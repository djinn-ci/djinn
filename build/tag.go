package build

import (
	"encoding/json"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/queue"
)

type Tag struct {
	ID        int64
	UserID    int64
	BuildID   int64
	Name      string
	CreatedAt time.Time

	User  *auth.User
	Build *Build
}

var _ database.Model = (*Tag)(nil)

func (t *Tag) Primary() (string, any) { return "id", t.ID }

func (t *Tag) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":         &t.ID,
		"user_id":    &t.UserID,
		"build_id":   &t.BuildID,
		"name":       &t.Name,
		"created_at": &t.CreatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (t *Tag) Params() database.Params {
	return database.Params{
		"id":         database.ImmutableParam(t.ID),
		"user_id":    database.CreateOnlyParam(t.UserID),
		"build_id":   database.CreateOnlyParam(t.BuildID),
		"name":       database.CreateOnlyParam(t.Name),
		"created_at": database.CreateOnlyParam(t.CreatedAt),
	}
}

func (t *Tag) Bind(m database.Model) {
	switch v := m.(type) {
	case *auth.User:
		if t.UserID == v.ID {
			t.User = v
		}
	case *Build:
		if t.BuildID == v.ID {
			t.Build = v
		}
	}
}

func (t *Tag) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"user_id":    t.UserID,
		"build_id":   t.BuildID,
		"name":       t.Name,
		"created_at": t.CreatedAt,
		"url":        env.DJINN_API_SERVER + t.Endpoint(),
		"user":       t.User,
		"build":      t.Build,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (t *Tag) Endpoint(...string) string {
	if t.Build == nil {
		return ""
	}
	return t.Build.Endpoint("tags", t.Name)
}

type TagEvent struct {
	dis event.Dispatcher

	Build *Build
	User  *auth.User
	Tags  []*Tag
}

var _ queue.Job = (*Event)(nil)

func InitTagEvent(dis event.Dispatcher) queue.InitFunc {
	return func(j queue.Job) {
		if ev, ok := j.(*TagEvent); ok {
			ev.dis = dis
		}
	}
}

func (*TagEvent) Name() string { return "event:" + event.BuildTagged.String() }

func (e *TagEvent) Perform() error {
	if e.dis == nil {
		return event.ErrNilDispatcher
	}

	tt := make([]map[string]any, 0, len(e.Tags))

	for _, t := range e.Tags {
		e.Build.Tags = append(e.Build.Tags, t)

		tt = append(tt, map[string]any{
			"name": t.Name,
			"url":  env.DJINN_API_SERVER + t.Endpoint(),
		})
	}

	payload := map[string]any{
		"url":   e.Build,
		"build": e.Build,
		"user":  e.User,
		"tags":  tt,
	}

	ev := event.New(e.Build.NamespaceID, event.BuildTagged, payload)

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

const tagTable = "build_tags"

func NewTagStore(pool *database.Pool) *database.Store[*Tag] {
	return database.NewStore[*Tag](pool, tagTable, func() *Tag {
		return &Tag{}
	})
}
