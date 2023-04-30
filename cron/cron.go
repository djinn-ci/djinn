// Package cron implements the database.Model interface for the Cron entity.
// The Cron entity allows for build's to be submitted on a defined schedule.
package cron

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/manifest"
	"djinn-ci.com/namespace"
	"djinn-ci.com/queue"

	"github.com/andrewpillar/query"
)

type Schedule uint

//go:generate stringer -type Schedule -linecomment
const (
	Daily   Schedule = iota // daily
	Weekly                  // weekly
	Monthly                 // monthly
)

var schedules = map[string]Schedule{
	"daily":   Daily,
	"weekly":  Weekly,
	"monthly": Monthly,
}

// Next returns the next time a schedule will occur. If Daily the start of the
// next day is returned. If Weekly the start of the next week is returned. If
// Monthly the start of the next month is returned.
func (s Schedule) Next() time.Time {
	now := time.Now()

	switch s {
	case Daily:
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	case Weekly:
		offset := int(now.Weekday()-time.Sunday) + 1

		return time.Date(now.Year(), now.Month(), now.Day()+offset, 0, 0, 0, 0, time.UTC)
	case Monthly:
		return time.Date(now.Year(), now.Month()+1, now.Day(), 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
	}
}

var ErrInvalidSchedule = errors.New("invalid schedule")

func (s Schedule) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(s.String())

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

// UnmarshalJSON attempts to unmarshal the given byte slice as a JSON string,
// and checks to see if it is a valid schedule of either daily, weekly, or
// monthly.
func (s *Schedule) UnmarshalJSON(b []byte) error {
	var str string

	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	var ok bool

	(*s), ok = schedules[str]

	if !ok {
		return ErrInvalidSchedule
	}
	return nil
}

// UnmarshalText takes the given byte slice, and attempts to map it to a known
// Schedule. If it is a known Schedule, then that the current Schedule is
// set to that, otherwise ErrInvalidSchedule is returned.
func (s *Schedule) UnmarshalText(b []byte) error {
	var ok bool

	(*s), ok = schedules[string(b)]

	if !ok {
		return ErrInvalidSchedule
	}
	return nil
}

// Value returns the string value of the current Schedule so it can be inserted
// into the database.
func (s Schedule) Value() (driver.Value, error) { return driver.Value(s.String()), nil }

// Scan scans the given interface value into a byte slice and will attempt to
// turn it into the correct Schedule value. If it success then it set's it on
// the current Schedule, otherwise an error is returned.
func (s *Schedule) Scan(val interface{}) error {
	v, err := driver.String.ConvertValue(val)

	if err != nil {
		return errors.Err(err)
	}

	str, ok := v.(string)

	if !ok {
		return errors.New("cron: could not type assert Schedule to string")
	}

	if err := s.UnmarshalText([]byte(str)); err != nil {
		return errors.Err(err)
	}
	return nil
}

type Cron struct {
	loaded []string

	ID          int64
	UserID      int64
	AuthorID    int64
	NamespaceID database.Null[int64]
	Name        string
	Schedule    Schedule
	Manifest    manifest.Manifest
	PrevRun     database.Null[time.Time]
	NextRun     time.Time
	CreatedAt   time.Time

	Author    *auth.User
	User      *auth.User
	Namespace *namespace.Namespace
}

var _ database.Model = (*Cron)(nil)

func (c *Cron) Primary() (string, any) { return "id", c.ID }

func (c *Cron) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":           &c.ID,
		"user_id":      &c.UserID,
		"author_id":    &c.AuthorID,
		"namespace_id": &c.NamespaceID,
		"name":         &c.Name,
		"schedule":     &c.Schedule,
		"manifest":     &c.Manifest,
		"prev_run":     &c.PrevRun,
		"next_run":     &c.NextRun,
		"created_at":   &c.CreatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}

	c.loaded = r.Columns
	return nil
}

func (c *Cron) Params() database.Params {
	params := database.Params{
		"id":           database.ImmutableParam(c.ID),
		"user_id":      database.CreateUpdateParam(c.UserID),
		"author_id":    database.CreateOnlyParam(c.AuthorID),
		"namespace_id": database.CreateUpdateParam(c.NamespaceID),
		"name":         database.CreateUpdateParam(c.Name),
		"schedule":     database.CreateUpdateParam(c.Schedule),
		"manifest":     database.CreateUpdateParam(c.Manifest),
		"prev_run":     database.UpdateOnlyParam(c.PrevRun),
		"next_run":     database.CreateUpdateParam(c.NextRun),
		"created_at":   database.CreateOnlyParam(c.CreatedAt),
	}

	if len(c.loaded) > 0 {
		params.Only(c.loaded...)
	}
	return params
}

func (c *Cron) Bind(m database.Model) {
	switch v := m.(type) {
	case *auth.User:
		c.Author = v

		if c.UserID == v.ID {
			c.User = v
		}
	case *namespace.Namespace:
		if c.NamespaceID.Elem == v.ID {
			c.Namespace = v
		}
	}
}

func (c *Cron) MarshalJSON() ([]byte, error) {
	if c == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"id":           c.ID,
		"author_id":    c.AuthorID,
		"user_id":      c.UserID,
		"namespace_id": c.NamespaceID,
		"name":         c.Name,
		"schedule":     c.Schedule,
		"manifest":     c.Manifest.String(),
		"prev_run":     c.PrevRun,
		"next_run":     c.NextRun,
		"created_at":   c.CreatedAt,
		"url":          env.DJINN_API_SERVER + c.Endpoint(),
		"author":       c.Author,
		"user":         c.User,
		"namespace":    c.Namespace,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (c *Cron) Endpoint(elems ...string) string {
	if len(elems) > 0 {
		return "/cron/" + strconv.FormatInt(c.ID, 10) + "/" + strings.Join(elems, "/")
	}
	return "/cron/" + strconv.FormatInt(c.ID, 10)
}

type Event struct {
	dis event.Dispatcher

	Cron   *Cron
	Action string
}

var _ queue.Job = (*Event)(nil)

func InitEvent(dis event.Dispatcher) queue.InitFunc {
	return func(j queue.Job) {
		if e, ok := j.(*Event); ok {
			e.dis = dis
		}
	}
}

func (*Event) Name() string { return "event:" + event.Cron.String() }

func (e *Event) Perform() error {
	if e.dis == nil {
		return event.ErrNilDispatcher
	}

	ev := event.New(e.Cron.NamespaceID, event.Cron, map[string]any{
		"cron":   e.Cron,
		"action": e.Action,
	})

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

const (
	table      = "cron"
	buildTable = "cron_builds"
)

type Store struct {
	*database.Store[*Cron]
}

func NewStore(pool *database.Pool) *database.Store[*Cron] {
	return database.NewStore[*Cron](pool, table, func() *Cron {
		return &Cron{}
	})
}

func SelectBuild(expr query.Expr, opts ...query.Option) query.Query {
	return query.Select(expr, append([]query.Option{query.From(buildTable)}, opts...)...)
}

type Params struct {
	User     *auth.User
	Name     string
	Schedule Schedule
	Manifest manifest.Manifest
}

func (s Store) Create(ctx context.Context, p *Params) (*Cron, error) {
	c := Cron{
		UserID:    p.User.ID,
		AuthorID:  p.User.ID,
		Name:      p.Name,
		Schedule:  p.Schedule,
		Manifest:  p.Manifest,
		NextRun:   p.Schedule.Next(),
		CreatedAt: time.Now(),
		User:      p.User,
		Author:    p.User,
	}

	if p.Manifest.Namespace != "" {
		path, err := namespace.ParsePath(p.Manifest.Namespace)

		if err != nil {
			return nil, errors.Err(err)
		}

		u, n, err := path.Resolve(ctx, s.Pool, p.User)

		if err != nil {
			return nil, errors.Err(err)
		}

		c.UserID = u.ID
		c.User = u
		c.NamespaceID = database.Null[int64]{
			Elem:  n.ID,
			Valid: true,
		}
		c.Namespace = n
	}

	if err := s.Store.Create(ctx, &c); err != nil {
		return nil, errors.Err(err)
	}
	return &c, nil
}

func (s Store) Update(ctx context.Context, c *Cron) error {
	loaded := c.loaded
	c.loaded = []string{"name", "schedule", "manifest", "prev_run", "next_run"}

	if c.Manifest.Namespace != "" {
		path, err := namespace.ParsePath(c.Manifest.Namespace)

		if err != nil {
			return errors.Err(err)
		}

		u, n, err := path.Resolve(ctx, s.Pool, c.User)

		if err != nil {
			return errors.Err(err)
		}

		c.User = u
		c.UserID = u.ID
		c.NamespaceID = database.Null[int64]{
			Elem:  n.ID,
			Valid: n.ID > 0,
		}
		c.Namespace = n
	}

	if err := s.Store.Update(ctx, c); err != nil {
		return errors.Err(err)
	}

	c.loaded = loaded
	return nil
}

// Invoke will create a new build for the given Cron if the NextRun time is
// after the current time. This will add a tag to the created build detailing
// the name of the Cron, and it's schedule.
func (s Store) Invoke(ctx context.Context, c *Cron) (*build.Build, error) {
	if time.Now().Before(c.NextRun) {
		return nil, nil
	}

	p := build.Params{
		User: &auth.User{
			ID: c.UserID,
		},
		Manifest: c.Manifest,
		Trigger: &build.Trigger{
			Type:    build.Schedule,
			Comment: c.Name + ": Scheduled build, next run " + c.NextRun.Format("Mon Jan 2 15:04:05 2006"),
			Data: map[string]string{
				"email":    c.User.Email,
				"username": c.User.Username,
			},
		},
		Tags: []string{
			"cron:" + strings.Replace(c.Name, " ", "-", -1),
		},
	}

	builds := build.Store{Store: build.NewStore(s.Pool)}

	b, err := builds.Create(ctx, &p)

	if err != nil {
		return nil, errors.Err(err)
	}

	c.PrevRun = database.Null[time.Time]{
		Elem:  time.Now(),
		Valid: true,
	}
	c.NextRun = c.Schedule.Next()

	if err := s.Update(ctx, c); err != nil {
		return nil, errors.Err(err)
	}

	q := query.Insert(
		buildTable,
		query.Columns("cron_id", "build_id"),
		query.Values(c.ID, b.ID),
	)

	if _, err := s.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (s Store) Index(ctx context.Context, vals url.Values, opts ...query.Option) (*database.Paginator[*Cron], error) {
	page, _ := strconv.Atoi(vals.Get("page"))

	opts = append([]query.Option{
		database.Search("name", vals.Get("search")),
	}, opts...)

	p, err := s.Paginate(ctx, page, database.PageLimit, opts...)

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := p.Load(ctx, s.Store, append(opts, query.OrderAsc("name"))...); err != nil {
		return nil, errors.Err(err)
	}
	return p, nil
}
