package model

import (
	"fmt"
	"net"
	"time"

	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

var sourceFmt = "host=%s port=%s dbname=%s user=%s password=%s sslmode=disable"

type model struct {
	*sqlx.DB

	ID        int64     `db:"id,omitcreate"`
	CreatedAt time.Time `db:"created_at,omitcreate"`
	UpdatedAt time.Time `db:"updated_at"`
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

func (m model) IsZero() bool {
	return m.ID == 0 && m.CreatedAt == time.Time{} && m.UpdatedAt == time.Time{}
}
