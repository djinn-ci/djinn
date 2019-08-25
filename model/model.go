package model

import (
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model/query"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

type Interface interface {
	SetPrimary(i int64)

	Primary() (string, int64)

	Values() map[string]interface{}
}

type Model struct {
	*sqlx.DB `db:"-"`

	ID        int64     `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Resource interface {

}

type Store struct {
	*sqlx.DB
}

var (
	sourceFmt = "host=%s port=%s dbname=%s user=%s password=%s sslmode=disable"

	ArtifactTable      = "artifacts"
	BuildTable         = "builds"
	BuildObjectTable   = "build_objects"
	BuildVariableTable = "build_variables"
	CollaboratorTable  = "collaborators"
	DriverTable        = "drivers"
	InviteTable        = "invites"
	JobTable           = "jobs"
	JobDependencyTable = "job_dependencies"
	KeyTable           = "keys"
	NamespaceTable     = "namespaces"
	ObjectTable        = "objects"
	StageTable         = "stages"
	TagTable           = "tags"
	TriggerTable       = "triggers"
	UserTable          = "users"
	VariableTable      = "variables"
)

// Convert a slice of models of length l, into a slice of model.Interface. The
// given callback takes the current index of the new model.Interface slice as
// its only argument. It is expected for this index to be used to return the
// original type that implements model.Interface from a source slice.
func interfaceSlice(l int, get func(i int) Interface) []Interface {
	models := make([]Interface, l, l)

	for i := range models {
		models[i] = get(i)
	}

	return models
}

// Return a slice of values for the given key from the given slice of models.
func mapKey(key string, models []Interface) []interface{} {
	vals := make([]interface{}, 0, len(models))

	for _, m := range models {
		val, ok := m.Values()[key]

		if !ok {
			continue
		}

		vals = append(vals, val)
	}

	return vals
}

func Connect(addr, dbname, username, password string) (*sqlx.DB, error) {
	host, port, err := net.SplitHostPort(addr)

	if err != nil {
		return nil, errors.Err(err)
	}

	source := fmt.Sprintf(sourceFmt, host, port, dbname, username, password)

	log.Debug.Printf("opening postgresql connection with '%s'\n", source)

	db, err := sqlx.Open("postgres", source)

	if err != nil {
		return nil, errors.Err(err)
	}

	log.Debug.Println("testing connection to database")

	return db, errors.Err(db.Ping())
}

func ForBuild(b *Build) query.Option {
	return func(q query.Query) query.Query {
		if b == nil || b.IsZero() {
			return q
		}

		return query.WhereEq("build_id", b.ID)(q)
	}
}

func ForJob(j *Job) query.Option {
	return func(q query.Query) query.Query {
		if j == nil || j.IsZero() {
			return q
		}

		return query.WhereEq("job_id", j.ID)(q)
	}
}

func ForNamespace(n *Namespace) query.Option {
	return func(q query.Query) query.Query {
		if n == nil || n.IsZero() {
			return q
		}

		return query.WhereEq("namespace_id", n.ID)(q)
	}
}

func ForRootNamespace(n *Namespace) query.Option {
	return func(q query.Query) query.Query {
		if n == nil || n.IsZero() {
			return q
		}

		return query.WhereEq("namespace_id", n.RootID)(q)
	}
}

func ForObject(o *Object) query.Option {
	return func(q query.Query) query.Query {
		if o == nil || o.IsZero() {
			return q
		}

		return query.WhereEq("object_id", o.ID)(q)
	}
}

func ForParent(n *Namespace) query.Option {
	return func(q query.Query) query.Query {
		if n == nil || n.IsZero() {
			return q
		}

		return query.WhereEq("parent_id", n.ID)(q)
	}
}

func ForStage(s *Stage) query.Option {
	return func(q query.Query) query.Query {
		if s == nil || s.IsZero() {
			return q
		}

		return query.WhereEq("stage_id", s.ID)(q)
	}
}

func ForUser(u *User) query.Option {
	return func(q query.Query) query.Query {
		if u == nil || u.IsZero() {
			return q
		}

		return query.WhereEq("user_id", u.ID)(q)
	}
}

func Search(col, search string) query.Option {
	return func(q query.Query) query.Query {
		if search == "" {
			return q
		}

		return query.WhereLike(col, "%" + search + "%")(q)
	}
}

func (m Model) Primary() (string, int64) {
	return "id", m.ID
}

func (m Model) IsZero() bool {
	return m.ID == 0 && m.CreatedAt == time.Time{} && m.UpdatedAt == time.Time{}
}

func (m *Model) SetPrimary(i int64) {
	m.ID = i
}

func (s Store) All(i interface{}, table string, opts ...query.Option) error {
	opts = append(opts, query.Columns("*"), query.Table(table))

	q := query.Select(opts...)

	err := s.Select(i, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return errors.Err(err)
}

func (s Store) Create(table string, ii ...Interface) error {
	for _, i := range ii {
		m := i.Values()

		cols := make([]string, 0, len(m))
		vals := make([]interface{}, 0, len(m))

		for k, v := range m {
			cols = append(cols, k)
			vals = append(vals, v)
		}

		q := query.Insert(
			query.Table(table),
			query.Columns(cols...),
			query.Values(vals...),
			query.Returning("id"),
		)

		stmt, err := s.Prepare(q.Build())

		if err != nil {
			return errors.Err(err)
		}

		defer stmt.Close()

		row := stmt.QueryRow(q.Args()...)

		var id int64

		if err := row.Scan(&id); err != nil {
			return errors.Err(err)
		}

		i.SetPrimary(id)
	}

	return nil
}

func (s Store) Delete(table string, ii ...Interface) error {
	if len(ii) == 0 {
		return nil
	}

	peek := ii[0]
	col, val := peek.Primary()

	ids := make([]interface{}, len(ii), len(ii))

	for i, model := range ii {
		_, val = model.Primary()

		ids[i] = val
	}

	q := query.Delete(
		query.Table(table),
		query.WhereIn(col, ids...),
	)

	qs := q.Build()

	stmt, err := s.Prepare(qs)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(q.Args()...)

	return errors.Err(err)
}

func (s Store) FindBy(i Interface, table, col string, val interface{}) error {
	q := query.Select(
		query.Columns("*"),
		query.Table(table),
		query.WhereEq(col, val),
	)

	err := s.Get(i, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return errors.Err(err)
}

func (s Store) Update(table string, ii ...Interface) error {
	for _, i := range ii {
		m := i.Values()
		col, val := i.Primary()

		opts := []query.Option{
			query.Table(table),
			query.SetRaw("updated_at", "NOW()"),
		}

		for k, v := range m {
			if k == col {
				continue
			}

			opts = append(opts, query.Set(k, v))
		}

		opts = append(opts, query.WhereEq(col, val))

		q := query.Update(opts...)

		stmt, err := s.Prepare(q.Build())

		if err != nil {
			return errors.Err(err)
		}

		defer stmt.Close()

		_, err = stmt.Exec(q.Args()...)

		return errors.Err(err)
	}

	return nil
}
