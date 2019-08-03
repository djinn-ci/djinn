package model

import (
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model/query"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

var sourceFmt = "host=%s port=%s dbname=%s user=%s password=%s sslmode=disable"

type Action uint8

const (
	Create Action = iota
	Edit
	Show
	Delete
)

type Model struct {
	*sqlx.DB `db:"-"`

	ID        int64     `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Resource interface {
	IsZero() bool

	AccessibleBy(u *User, a Action) bool
}

type ResourceFinder func(name string, vars map[string]string) []query.Option

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

func (m Model) IsZero() bool {
	return m.ID == 0 && m.CreatedAt == time.Time{} && m.UpdatedAt == time.Time{}
}

func (rs *ResourceStore) Register(name string, t Type) {
	if rs.types == nil {
		rs.types = make(map[string]Type)
	}

	if t.HandleFind == nil {
		t.HandleFind = func(name string, vars map[string]string) []query.Option {
			id, _ := strconv.ParseInt(vars[name], 10, 64)

			return []query.Option{
				query.Columns("*"),
				query.Table(t.Table),
				query.WhereEq("id", id),
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

	q := query.Select(t.HandleFind(name, vars)...)

	err := rs.Get(t.Resource, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return t.Resource, errors.Err(err)
}
