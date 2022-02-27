package build

import (
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/queue"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type Tag struct {
	ID        int64
	UserID    int64
	BuildID   int64
	Name      string
	CreatedAt time.Time

	User  *user.User
	Build *Build
}

var _ database.Model = (*Tag)(nil)

func (t *Tag) Dest() []interface{} {
	return []interface{}{
		&t.ID,
		&t.UserID,
		&t.BuildID,
		&t.Name,
		&t.CreatedAt,
	}
}

func (t *Tag) Bind(m database.Model) {
	switch v := m.(type) {
	case *user.User:
		if t.UserID == v.ID {
			t.User = v
		}
	case *Build:
		if t.BuildID == v.ID {
			t.Build = v
		}
	}
}

func (t *Tag) Endpoint(uri ...string) string {
	if t.Build == nil {
		return ""
	}
	return t.Build.Endpoint("tags", t.Name)
}

func (t *Tag) JSON(addr string) map[string]interface{} {
	if t == nil {
		return nil
	}

	json := map[string]interface{}{
		"id":         t.ID,
		"user_id":    t.UserID,
		"build_id":   t.BuildID,
		"name":       t.Name,
		"created_at": t.CreatedAt.Format(time.RFC3339),
		"url":        addr + t.Endpoint(),
	}

	if t.User != nil {
		json["user"] = t.User.JSON(addr)
	}
	if t.Build != nil {
		json["build"] = t.Build.JSON(addr)
	}
	return json
}

func (t *Tag) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":         t.ID,
		"user_id":    t.UserID,
		"build_id":   t.BuildID,
		"name":       t.Name,
		"created_at": t.CreatedAt,
	}
}

type TagEvent struct {
	dis event.Dispatcher

	Build *Build
	User  *user.User
	Tags  []*Tag
}

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

	tt := make([]map[string]interface{}, 0, len(e.Tags))

	for _, t := range e.Tags {
		e.Build.Tags = append(e.Build.Tags, t)

		tt = append(tt, map[string]interface{}{
			"name": t.Name,
			"url":  env.DJINN_API_SERVER + t.Endpoint(),
		})
	}

	payload := map[string]interface{}{
		"url":   e.Build.Endpoint("tags"),
		"build": e.Build.JSON(env.DJINN_API_SERVER),
		"user":  e.User.JSON(env.DJINN_API_SERVER),
		"tags":  tt,
	}

	ev := event.New(e.Build.NamespaceID, event.BuildTagged, payload)

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

type TagStore struct {
	database.Pool
}

type TagParams struct {
	UserID  int64
	BuildID int64
	Tags    []string
}

var (
	_ database.Loader = (*TagStore)(nil)

	_ queue.Job = (*Event)(nil)

	tagTable = "build_tags"
)

func (s TagStore) Create(p TagParams) ([]*Tag, error) {
	if len(p.Tags) == 0 {
		return nil, nil
	}

	set := make(map[string]struct{})
	vals := make([]interface{}, 0, len(p.Tags))

	for _, tag := range p.Tags {
		set[tag] = struct{}{}
	}

	p.Tags = p.Tags[0:0]

	for tag := range set {
		p.Tags = append(p.Tags, tag)
		vals = append(vals, tag)
		delete(set, tag)
	}

	q := query.Select(
		query.Columns("name"),
		query.From(tagTable),
		query.Where("build_id", "=", query.Arg(p.BuildID)),
		query.Where("name", "IN", query.List(vals...)),
	)

	rows, err := s.Query(q.Build(), q.Args()...)

	if err != nil {
		return nil, errors.Err(err)
	}

	for rows.Next() {
		var s string

		if err := rows.Scan(&s); err != nil {
			return nil, errors.Err(err)
		}
		set[s] = struct{}{}
	}

	tt := make([]*Tag, 0, len(p.Tags))

	now := time.Now()

	for _, tag := range p.Tags {
		if _, ok := set[tag]; ok {
			continue
		}

		if tag == "" {
			continue
		}

		q = query.Insert(
			tagTable,
			query.Columns("user_id", "build_id", "name", "created_at"),
			query.Values(p.UserID, p.BuildID, tag, now),
			query.Returning("id"),
		)

		var id int64

		if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
			return nil, errors.Err(err)
		}
		tt = append(tt, &Tag{
			ID:        id,
			UserID:    p.UserID,
			BuildID:   p.BuildID,
			Name:      tag,
			CreatedAt: now,
		})
	}
	return tt, nil
}

func (s TagStore) Delete(id int64) error {
	q := query.Delete(tagTable, query.Where("id", "=", query.Arg(id)))

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s TagStore) Get(opts ...query.Option) (*Tag, bool, error) {
	var t Tag

	ok, err := s.Pool.Get(tagTable, &t, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &t, ok, nil
}

func (s TagStore) All(opts ...query.Option) ([]*Tag, error) {
	tt := make([]*Tag, 0)

	new := func() database.Model {
		t := &Tag{}
		tt = append(tt, t)
		return t
	}

	if err := s.Pool.All(tagTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return tt, nil
}

func (s TagStore) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	tt, err := s.All(query.Where(pk, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	for _, t := range tt {
		for _, m := range mm {
			m.Bind(t)
		}
	}
	return nil
}
