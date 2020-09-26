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
	"github.com/andrewpillar/djinn/form"
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

// Cron is the type that represents a cron job that has been created by the user.
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

// Store is the type for creating, modifying, and deleting Cron models in the
// database.
type Store struct {
	database.Store

	// User is the bound user.User model. If not nil this will bind the
	// user.User model to any Cron models that are created. If not nil this
	// will append a WHERE clause on the user_id column for all SELECT queries
	// performed.
	User *user.User

	// Namespace is the bound namespace.Namespace model. If not nil this will
	// bind the namespace.Namespace model to any Variable models that are
	// created. If not nil this will append a WHERE clause on the namespace_id
	// column for all SELECT queries performed.
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

// NewStore returns a new Store for querying the cron table. Each of the given
// models is bound to the returned Store.
func NewStore(db *sqlx.DB, mm ...database.Model) *Store {
	s := &Store{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// FromContext returns the Cron model from the given context, if any.
func FromContext(ctx context.Context) (*Cron, bool) {
	c, ok := ctx.Value("cron").(*Cron)
	return c, ok
}

// Model is called along with database.ModelSlice to convert the given slice of
// Cron models to a slice of database.Model interfaces.
func Model(cc []*Cron) func(int) database.Model {
	return func(i int) database.Model {
		return cc[i]
	}
}

// LoadRelations loads all of the available relations for the given Cron models
// using the given loaders available.
func LoadRelations(loaders *database.Loaders, cc ...*Cron) error {
	mm := database.ModelSlice(len(cc), Model(cc))
	return errors.Err(database.LoadRelations(relations, loaders, mm...))
}

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User or namespace.Namespace.
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

// SetPrimary implements the database.Model interface.
func (c *Cron) SetPrimary(id int64) { c.ID = id }

// Primary implements the database.Model interface.
func (c *Cron) Primary() (string, int64) { return "id", c.ID }

// IsZero implements the database.Model interface.
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

// JSON implements the database.Model interface. This will return a map with
// the current Cron values under each key. If any of the User, or Namespace
// bound models exist on the Cron, then the JSON representation of these models
// will be returned in the map, under the user, and namespace keys respectively.
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

// Endpoint returns the endpoint to the current Variable database, with the given
// URI parts appended to it.
func (c *Cron) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/cron/" + strconv.FormatInt(c.ID, 10) + "/" + strings.Join(uri, "/")
	}
	return "/cron/" + strconv.FormatInt(c.ID, 10)
}

// Values implements the database.Model interface. This will return a map with
// the following values, user_id, namespace_id, name, schedule, manifest, and
// next_run.
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

// Bind implements the database.Binder interface. This will only bind the model
// if they are pointers to either user.User or namespace.Namespace.
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

// New returns a new Cron binding any non-nil models to it from the current
// Store.
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

// All returns a single Cron model, applying each query.Option that is
// given. The namespace.WhereCollaborator option is applied to the *user.User
// bound database, and the database.Where option is applied to the
// *namespace.Namespace bound database.
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

// All returns a slice of Variable models, applying each query.Option that is
// given. The namespace.WhereCollaborator option is applied to the *user.User
// bound database, and the database.Where option is applied to the
// *namespace.Namespace bound database.
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

// Paginate returns the database.Paginator for the cron table for the given page.
// This applies the namespace.WhereCollaborator option to the *user.User bound
// database, and the database.Where option to the *namespace.Namespace bound
// database.
func (s *Store) Paginate(page int64, opts ...query.Option) (database.Paginator, error) {
	paginator, err := s.Store.Paginate(table, page, opts...)
	return paginator, errors.Err(err)
}

// Index returns the paginated results from the cron table depending on the
// values that are present in url.Values. Detailed below are the values that
// are used from the given url.Values,
//
// name - This applies the database.Search query.Option using the value of key
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

// Load loads in a slice of Cron models where the given key is in the list of
// given vals. Each database is loaded individually via a call to the given
// load callback. This method calls Store.All under the hood, so any bound
// models will impact the models being loaded.
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

// UnmarshalText takes the given byte slice, and attempts to map it to a known
// Schedule. If it is a known Schedule, then that the current Schedule is
// set to that, otherwise form.UnmarshalError is returned.
func (s *Schedule) UnmarshalText(b []byte) error {
	var ok bool

	(*s), ok = schedules[string(b)]

	if !ok {
		return form.UnmarshalError{
			Field: "schedule",
			Err:   errors.New("unknown schedule: " + string(b)),
		}
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
	b, err := database.Scan(val)

	if err != nil {
		return errors.Err(err)
	}
	return errors.Err(s.UnmarshalText(b))
}
