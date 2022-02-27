// Package cron implements the database.Model interface for the Cron entity.
// The Cron entity allows for build's to be submitted on a defined schedule.
package cron

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/manifest"
	"djinn-ci.com/namespace"
	"djinn-ci.com/queue"
	"djinn-ci.com/user"

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
	ID          int64
	UserID      int64
	AuthorID    int64
	NamespaceID sql.NullInt64
	Name        string
	Schedule    Schedule
	Manifest    manifest.Manifest
	PrevRun     sql.NullTime
	NextRun     time.Time
	CreatedAt   time.Time

	Author    *user.User
	User      *user.User
	Namespace *namespace.Namespace
}

var _ database.Model = (*Cron)(nil)

func LoadNamespaces(db database.Pool, cc ...*Cron) error {
	mm := make([]database.Model, 0, len(cc))

	for _, c := range cc {
		mm = append(mm, c)
	}

	if err := namespace.Load(db, mm...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func LoadRelations(db database.Pool, cc ...*Cron) error {
	mm := make([]database.Model, 0, len(cc))

	for _, c := range cc {
		mm = append(mm, c)
	}

	if err := database.LoadRelations(mm, namespace.ResourceRelations(db)...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (c *Cron) Dest() []interface{} {
	return []interface{}{
		&c.ID,
		&c.UserID,
		&c.AuthorID,
		&c.NamespaceID,
		&c.Name,
		&c.Schedule,
		&c.Manifest,
		&c.PrevRun,
		&c.NextRun,
		&c.CreatedAt,
	}
}

func (c *Cron) Bind(m database.Model) {
	switch v := m.(type) {
	case *user.User:
		c.Author = v

		if c.UserID == v.ID {
			c.User = v
		}
	case *namespace.Namespace:
		if c.NamespaceID.Int64 == v.ID {
			c.Namespace = v
		}
	}
}

func (c *Cron) JSON(addr string) map[string]interface{} {
	if c == nil {
		return nil
	}

	json := map[string]interface{}{
		"id":           c.ID,
		"author_id":    c.AuthorID,
		"user_id":      c.UserID,
		"namespace_id": nil,
		"name":         c.Name,
		"schedule":     c.Schedule.String(),
		"manifest":     c.Manifest.String(),
		"next_run":     c.NextRun.Format(time.RFC3339),
		"created_at":   c.CreatedAt.Format(time.RFC3339),
		"url":          addr + c.Endpoint(),
	}

	if c.Author != nil {
		json["author"] = c.Author.JSON(addr)
	}

	if c.User != nil {
		json["user"] = c.User.JSON(addr)
	}

	if c.NamespaceID.Valid {
		json["namespace_id"] = c.NamespaceID.Int64

		if c.Namespace != nil {
			json["namespace"] = c.Namespace.JSON(addr)
		}
	}
	return json
}

func (c *Cron) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/cron/" + strconv.FormatInt(c.ID, 10) + "/" + strings.Join(uri, "/")
	}
	return "/cron/" + strconv.FormatInt(c.ID, 10)
}

func (c *Cron) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":           c.ID,
		"user_id":      c.UserID,
		"author_id":    c.AuthorID,
		"namespace_id": c.NamespaceID,
		"name":         c.Name,
		"schedule":     c.Schedule,
		"manifest":     c.Manifest,
		"prev_run":     c.PrevRun,
		"next_run":     c.NextRun,
		"created_at":   c.CreatedAt,
	}
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

	ev := event.New(e.Cron.NamespaceID, event.Cron, map[string]interface{}{
		"cron":   e.Cron.JSON(env.DJINN_API_SERVER),
		"action": e.Action,
	})

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

type Store struct {
	database.Pool
}

var (
	_ database.Loader = (*Store)(nil)

	table      = "cron"
	buildTable = "cron_builds"
)

// SelectBuildIDs returns a query that will select the build_id column from the
// cron_builds table where the cron_id matches the given id.
func SelectBuildIDs(id int64) query.Query {
	return query.Select(
		query.Columns("build_id"),
		query.From(buildTable),
		query.Where("cron_id", "=", query.Arg(id)),
	)
}

type Params struct {
	UserID   int64
	Name     string
	Schedule Schedule
	Manifest manifest.Manifest
}

func (s Store) Create(p Params) (*Cron, error) {
	// userId is changed to the namespace userId if the cron if being submitted
	// to a namespace.
	userId := p.UserID

	var (
		n           *namespace.Namespace
		namespaceId sql.NullInt64
	)

	if p.Manifest.Namespace != "" {
		path, err := namespace.ParsePath(p.Manifest.Namespace)

		if err != nil {
			return nil, errors.Err(err)
		}

		u, n0, err := path.ResolveOrCreate(s.Pool, p.UserID)

		if err != nil {
			return nil, errors.Err(err)
		}

		if err := n0.IsCollaborator(s.Pool, p.UserID); err != nil {
			return nil, errors.Err(err)
		}

		userId = u.ID

		n = n0
		n.User = u

		namespaceId = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}
	}

	now := time.Now()

	q := query.Insert(
		table,
		query.Columns("user_id", "author_id", "namespace_id", "name", "schedule", "manifest", "next_run", "created_at"),
		query.Values(userId, p.UserID, namespaceId, p.Name, p.Schedule, p.Manifest, p.Schedule.Next(), now),
		query.Returning("id"),
	)

	var id int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}

	return &Cron{
		ID:          id,
		UserID:      userId,
		AuthorID:    p.UserID,
		NamespaceID: namespaceId,
		Name:        p.Name,
		Schedule:    p.Schedule,
		Manifest:    p.Manifest,
		NextRun:     p.Schedule.Next(),
		CreatedAt:   now,
		Namespace:   n,
	}, nil
}

func (s Store) Update(id int64, p Params) error {
	opts := []query.Option{
		query.Set("name", query.Arg(p.Name)),
		query.Set("schedule", query.Arg(p.Schedule)),
		query.Set("manifest", query.Arg(p.Manifest)),
		query.Set("next_run", query.Arg(p.Schedule.Next())),
	}

	if p.Manifest.Namespace != "" {
		path, err := namespace.ParsePath(p.Manifest.Namespace)

		if err != nil {
			return errors.Err(err)
		}

		u, n, err := path.ResolveOrCreate(s.Pool, p.UserID)

		if err != nil {
			return errors.Err(err)
		}

		opts = append(opts,
			query.Set("user_id", query.Arg(u.ID)),
			query.Set("namespace_id", query.Arg(n.ID)),
		)
	}

	opts = append(opts, query.Where("id", "=", query.Arg(id)))

	q := query.Update(table, opts...)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) Delete(id int64) error {
	q := query.Delete(table, query.Where("id", "=", query.Arg(id)))

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

// Invoke will create a new build for the given Cron if the NextRun time is
// after the current time. This will add a tag to the created build detailing
// the name of the Cron, and it's schedule.
func (s Store) Invoke(c *Cron) (*build.Build, error) {
	if time.Now().Before(c.NextRun) {
		return nil, nil
	}

	c.NextRun = c.Schedule.Next()

	t := &build.Trigger{
		Type:    build.Schedule,
		Comment: c.Name + ": Scheduled build, next run " + c.NextRun.Format("Mon Jan 2 15:04:05 2006"),
		Data: map[string]string{
			"email":    c.User.Email,
			"username": c.User.Username,
		},
	}

	tag := "cron:" + strings.Replace(c.Name, " ", "-", -1)

	builds := build.Store{Pool: s.Pool}

	b, err := builds.Create(build.Params{
		UserID:   c.UserID,
		Manifest: c.Manifest,
		Trigger:  t,
		Tags:     []string{tag},
	})

	if err != nil {
		return nil, errors.Err(err)
	}

	q := query.Update(
		table,
		query.Set("prev_run", query.Arg(time.Now())),
		query.Set("next_run", query.Arg(c.NextRun)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return nil, errors.Err(err)
	}

	q = query.Insert(
		buildTable,
		query.Columns("cron_id", "build_id"),
		query.Values(c.ID, b.ID),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (s Store) Get(opts ...query.Option) (*Cron, bool, error) {
	var c Cron

	ok, err := s.Pool.Get(table, &c, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &c, ok, nil
}

func (s Store) All(opts ...query.Option) ([]*Cron, error) {
	cc := make([]*Cron, 0)

	new := func() database.Model {
		c := &Cron{}
		cc = append(cc, c)
		return c
	}

	if err := s.Pool.All(table, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return cc, nil
}

func (s Store) Paginate(page, limit int64, opts ...query.Option) (database.Paginator, error) {
	paginator, err := s.Pool.Paginate(table, page, limit, opts...)

	if err != nil {
		return paginator, errors.Err(err)
	}
	return paginator, nil
}

func (s Store) Index(vals url.Values, opts ...query.Option) ([]*Cron, database.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		database.Search("name", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, database.PageLimit, opts...)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	cc, err := s.All(append(
		opts,
		query.OrderAsc("name"),
		query.Limit(paginator.Limit),
		query.Offset(paginator.Offset),
	)...)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}
	return cc, paginator, nil
}

func (s Store) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	uu, err := s.All(query.Where(pk, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	for _, u := range uu {
		for _, m := range mm {
			m.Bind(u)
		}
	}
	return nil
}
