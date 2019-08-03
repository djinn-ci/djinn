package model

import (
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

var sourceFmt = "host=%s port=%s dbname=%s user=%s password=%s sslmode=disable"

type Model struct {
	*sqlx.DB `db:"-"`

	ID        int64     `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Resource interface {
	IsZero() bool

	AccessibleBy(u *User) bool
}

type ResourceFinder func(name string, vars map[string]string) []Option

type Type struct {
	Table      string
	Resource   Resource
	HandleFind ResourceFinder
}

type ResourceStore struct {
	*sqlx.DB

	types map[string]Type
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

func ForBuild(b *Build) Option {
	return func(q Query) Query {
		if b == nil || b.IsZero() {
			return q
		}

		return WhereEq("build_id", b.ID)(q)
	}
}

func ForJob(j *Job) Option {
	return func(q Query) Query {
		if j == nil || j.IsZero() {
			return q
		}

		return WhereEq("job_id", j.ID)(q)
	}
}

func ForNamespace(n *Namespace) Option {
	return func(q Query) Query {
		if n == nil || n.IsZero() {
			return q
		}

		return WhereEq("namespace_id", n.ID)(q)
	}
}

func ForObject(o *Object) Option {
	return func(q Query) Query {
		if o == nil || o.IsZero() {
			return q
		}

		return WhereEq("object_id", o.ID)(q)
	}
}

func ForParent(n *Namespace) Option {
	return func(q Query) Query {
		if n == nil || n.IsZero() {
			return q
		}

		return WhereEq("parent_id", n.ID)(q)
	}
}

func ForStage(s *Stage) Option {
	return func(q Query) Query {
		if s == nil || s.IsZero() {
			return q
		}

		return WhereEq("stage_id", s.ID)(q)
	}
}

func ForUser(u *User) Option {
	return func(q Query) Query {
		if u == nil || u.IsZero() {
			return q
		}

		return WhereEq("user_id", u.ID)(q)
	}
}

func Search(col, search string) Option {
	return func(q Query) Query {
		if search == "" {
			return q
		}

		return WhereLike(col, "%" + search + "%")(q)
	}
}

func (m Model) IsZero() bool {
	return m.ID == 0 && m.CreatedAt == time.Time{} && m.UpdatedAt == time.Time{}
}

func (rs *ResourceStore) Register(name string, t Type) {
	if rs.types == nil {
		rs.types = make(map[string]Type)
	}

	if t.HandleFind == nil {
		t.HandleFind = func(name string, vars map[string]string) []Option {
			id, _ := strconv.ParseInt(vars[name], 10, 64)

			return []Option{
				Columns("*"),
				Table(t.Table),
				WhereEq("id", id),
			}
		}
	}

	rs.types[name] = t
}

func (rs ResourceStore) Find(name string, vars map[string]string) (Resource, error) {
	t, ok := rs.types[name]

	if !ok {
		return nil, errors.Err(errors.New("unknown resource model " + name))
	}

	q := Select(t.HandleFind(name, vars)...)

	err := rs.Get(t.Resource, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return t.Resource, errors.Err(err)
}
