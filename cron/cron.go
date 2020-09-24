// Package cron implements the database.Model interface for the Cron entity.
// The Cron entity allows for build's to be submitted on a defined schedule.
package cron

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

// Schedule represents the schedule of the Cron. This will either be Daily,
// Weekly, or Monthly. Below is how the Cron schedules are handled,
//
// Daily   - This will trigger a Cron on the start of the next day
// Weekly  - This will trigger a Cron on the start of the next week
// Monthly - This will trigger a Cron on the start of the next month
type Schedule uint

type Cron struct {
	ID          int64           `db:"id"`
	UserID      int64           `db:"user_id"`
	NamespaceID sql.NullInt64   `db:"namespace_id"`
	Name        string          `db:"name"`
	Schedule    Schedule        `db:"schedule"`
	Manifest    config.Manifest `db:"manifest"`
	PrevRun     sql.NullTime    `db:"prev_run"`
	NextRun     time.Time       `db:"next_run"`
	CreatedAt   time.Time       `db:"created_at"`

	User      *user.User           `db:"-"`
	Namespace *namespace.Namespace `db:"-"`
}

type Store struct {
	database.Store

	User      *user.User
	Namespace *namespace.Namespace
}

//go:generate stringer -type Schedule -linecomment
const (
	Daily Schedule = iota // daily
	Weekly                // weekly
	Monthly               // monthly
)

var (
	_ database.Model  = (*Cron)(nil)
	_ database.Binder = (*Store)(nil)
	_ database.Loader = (*Store)(nil)

	table     = "cron"
	relations = map[string]database.RelationFunc{
		"namespace": database.Relation("namespace_id", "id"),
		"user":      database.Relation("user_id", "id"),
	}

	schedules = map[string]Schedule{
		"daily":   Daily,
		"weekly":  Weekly,
		"monthly": Monthly,
	}
)

func NewStore(db *sqlx.DB, mm ...database.Model) *Store {
	s := &Store{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func FromContext(ctx context.Context) (*Cron, bool) {
	c, ok := ctx.Value("cron").(*Cron)
	return c, ok
}

func Model(cc []*Cron) func(int) database.Model {
	return func(i int) database.Model {
		return cc[i]
	}
}

func SelectBuild(col string, opts ...query.Option) query.Query {
	return query.Select(append([]query.Option{
		query.Columns(col),
		query.From(buildTable),
	}, opts...)...)
}

func LoadRelations(loaders *database.Loaders, cc ...*Cron) error {
	mm := database.ModelSlice(len(cc), Model(cc))
	return errors.Err(database.LoadRelations(relations, loaders, mm...))
}

func (c *Cron) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			c.User = m.(*user.User)
		case *namespace.Namespace:
			c.Namespace = m.(*namespace.Namespace)
		}
	}
}

func (c *Cron) SetPrimary(id int64) { c.ID = id }

func (c *Cron) Primary() (string, int64) { return "id", c.ID }

func (c *Cron) IsZero() bool {
	return c == nil || c.ID == 0 &&
		c.UserID == 0 &&
		!c.NamespaceID.Valid &&
		c.Name == "" &&
		c.Schedule == Schedule(0) &&
		c.Manifest.String() == "{}" &&
		!c.PrevRun.Valid &&
		c.NextRun == time.Time{} &&
		c.CreatedAt == time.Time{}
}

func (c *Cron) JSON(addr string) map[string]interface{} {
	json := map[string]interface{}{
		"id":           c.ID,
		"user_id":      c.UserID,
		"namespace_id": nil,
		"name":         c.Name,
		"schedule":     c.Schedule.String(),
		"manifest":     c.Manifest.String(),
		"next_run":     c.NextRun.Format(time.RFC3339),
		"created_at":   c.CreatedAt.Format(time.RFC3339),
		"url":          addr + c.Endpoint(),
	}

	if c.NamespaceID.Valid {
		json["namespace_id"] = c.NamespaceID.Int64
	}

	for name, m := range map[string]database.Model{
		"user":      c.User,
		"namespace": c.Namespace,
	}{
		if m != nil && !m.IsZero() {
			json[name] = m.JSON(addr)
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
		"user_id":      c.UserID,
		"namespace_id": c.NamespaceID,
		"name":         c.Name,
		"schedule":     c.Schedule,
		"manifest":     c.Manifest,
		"next_run":     c.NextRun,
	}
}

func (s *Store) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		case *namespace.Namespace:
			s.Namespace = m.(*namespace.Namespace)
		}
	}
}

func (s *Store) New() *Cron {
	c := &Cron{
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		c.UserID = s.User.ID
	}
	if s.Namespace != nil {
		c.NamespaceID = sql.NullInt64{
			Int64: s.Namespace.ID,
			Valid: true,
		}
	}
	return c
}

// Create will create a new Cron with the given name, schedule, and manifest.
func (s *Store) Create(name string, sched Schedule, m config.Manifest) (*Cron, error) {
	c := s.New()

	c.Name = name
	c.Schedule = sched
	c.Manifest = m
	c.NextRun = sched.Next()

	err := s.Store.Create(table, c)
	return c, errors.Err(err)
}

// Update will update the name, schedule, and manifest for the cron with the
// given id.
func (s *Store) Update(id int64, name string, sched Schedule, m config.Manifest) error {
	q := query.Update(
		query.Table(table),
		query.Set("name", name),
		query.Set("schedule", sched),
		query.Set("manifest", m),
		query.Set("next_run", sched.Next()),
	)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Delete a cron with the given id.
func (s *Store) Delete(id int64) error {
	q := query.Delete(
		query.From(table),
		query.Where("id", "=", id),
	)

	_, err := s.DB.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
}

// Invoke will create a new build for the given Cron if the NextRun time is
// after the current time. This will add a tag to the created build detailing
// the name of the Cron, and it's schedule.
func (s *Store) Invoke(c *Cron) (*build.Build, error) {
	if c.NextRun.Before(time.Now()) {
		return nil, nil
	}

	c.NextRun = c.Schedule.Next()

	t := &build.Trigger{
		Type:    build.Schedule,
		Comment: c.Name + ": Scheduled build, next run " + c.NextRun.Format("Mon Jan 2 15:04:05 2006"),
		Data:    map[string]string{
			"email":    c.User.Email,
			"username": c.User.Username,
		},
	}

	tag := "cron:" + c.Schedule.String() + " " + c.Name

	b, err := build.NewStore(s.DB, c.User, c.Namespace).Create(c.Manifest, t, tag)

	if err != nil {
		return nil, errors.Err(err)
	}

	q := query.Update(
		query.Table(table),
		query.Set("next_run", c.NextRun),
	)

	if _, err := s.DB.Exec(q.Build(), q.Args()...); err != nil {
		return nil, errors.Err(err)
	}

	cb := &Build{
		CronID:  c.ID,
		BuildID: b.ID,
	}

	if err := NewBuildStore(s.DB).Create(buildTable, cb); err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (s *Store) Get(opts ...query.Option) (*Cron, error) {
	c := &Cron{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(c, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return c, errors.Err(err)
}

func (s *Store) All(opts ...query.Option) ([]*Cron, error) {
	cc := make([]*Cron, 0)

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		database.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.All(&cc, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, c := range cc {
		c.User = s.User
		c.Namespace = s.Namespace
	}
	return cc, errors.Err(err)
}

func (s *Store) Paginate(page int64, opts ...query.Option) (database.Paginator, error) {
	paginator, err := s.Store.Paginate(table, page, opts...)
	return paginator, errors.Err(err)
}

func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Cron, database.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		database.Search("name", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	cc, err := s.All(append(
		opts,
		query.OrderAsc("name"),
		query.Limit(database.PageLimit),
		query.Offset(paginator.Offset),
	)...)
	return cc, paginator, errors.Err(err)
}

func (s *Store) Load(key string, vals []interface{}, load database.LoaderFunc) error {
	cc, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, c := range cc {
			load(i, c)
		}
	}
	return nil
}

// Next returns the next time a schedule will occur. If Daily the start of the
// next day is returned. If Weekly the start of the next week is returned. If
// Monthly the start of the next month is returned.
func (s *Schedule) Next() time.Time {
	now := time.Now()

	switch *s {
	case Daily:
		return time.Date(now.Year(), now.Month(), now.Day() + 1, 0, 0, 0, 0, time.UTC)
	case Weekly:
		offset := int(now.Weekday() - time.Sunday) + 1

		return time.Date(now.Year(), now.Month(), now.Day() + offset, 0, 0, 0, 0, time.UTC)
	case Monthly:
		return time.Date(now.Year(), now.Month() + 1, now.Day(), 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(now.Year(), now.Month(), now.Day() + 1, 0, 0, 0, 0, time.UTC)
	}
}

func (s *Schedule) UnmarshalText(b []byte) error {
	var ok bool

	(*s), ok = schedules[string(b)]

	if !ok {
		return errors.New("unknown schedule: " + string(b))
	}
	return nil
}

func (s Schedule) Value() (driver.Value, error) { return driver.Value(s.String()), nil }

func (s *Schedule) Scan(val interface{}) error {
	b, err := database.Scan(val)

	if err != nil {
		return errors.Err(err)
	}
	return errors.Err(s.UnmarshalText(b))
}
