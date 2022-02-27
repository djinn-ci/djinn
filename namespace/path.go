package namespace

import (
	"database/sql"
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type PathError struct {
	Path Path
	Err  error
}

func (e *PathError) Unwrap() error { return e.Err }

func (e *PathError) Error() string {
	return e.Path.String() + ": " + e.Err.Error()
}

// Path represents a path to a namespace. Owner is the username of the namespace
// owner, and Path is the full path of the namespace.
type Path struct {
	Owner string
	Path  string
	Valid bool
}

var ErrInvalidPath = errors.New("invalid namespace path")

func isPathChar(r rune) bool {
	return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || r == '/'
}

// ParsePath parses a raw namespace path in the format of path[@owner] into a
// Path structure. This returns *PathError on any errors that may occur.
func ParsePath(s string) (Path, error) {
	var p Path

	if s == "" {
		return p, nil
	}

	buf := make([]rune, 0, len(s))
	pos := -1

	for i, r := range s {
		if !isPathChar(r) {
			if r == '@' {
				pos = i
				break
			}
			return p, &PathError{
				Path: Path{Path: s},
				Err:  ErrName,
			}
		}
		buf = append(buf, r)
	}

	if len(buf) == 0 {
		return p, &PathError{
			Path: Path{Path: s},
			Err:  ErrInvalidPath,
		}
	}

	if pos > 0 {
		p.Owner = s[pos+1:]
	}

	p.Path = string(buf)
	p.Valid = true

	return p, nil
}

func (p *Path) String() string {
	s := p.Path

	if p.Owner != "" {
		s += "@" + p.Owner
	}
	return s
}

func (p *Path) UnmarshalJSON(b []byte) error {
	var (
		s   string
		err error
	)

	err = json.Unmarshal(b, &s)

	if err != nil {
		return errors.Err(err)
	}

	(*p), err = ParsePath(s)

	if err != nil {
		return err
	}
	return nil
}

// UnmarshalText unmarshals a byte slice of a raw namespace path into the
// current Path structure. Under the hood, this simply calls ParsePath.
func (p *Path) UnmarshalText(b []byte) error {
	var err error

	(*p), err = ParsePath(string(b))

	if err != nil {
		return err
	}
	return nil
}

// Resolve returns the owner and namespace for the path. The fallback ID is
// used if the current path has no owner. This returns *PathError on any errors
// that may occur.
func (p *Path) Resolve(db database.Pool, fallback int64) (*user.User, *Namespace, error) {
	if !p.Valid {
		return nil, nil, &PathError{
			Path: *p,
			Err:  ErrInvalidPath,
		}
	}

	users := user.Store{Pool: db}

	opt := user.WhereUsername(p.Owner)

	if p.Owner == "" {
		opt = user.WhereID(fallback)
	}

	u, ok, err := users.Get(opt)

	if err != nil {
		return nil, nil, &PathError{
			Path: *p,
			Err:  errors.Err(err),
		}
	}

	if !ok {
		return nil, nil, &PathError{
			Path: *p,
			Err:  ErrOwner,
		}
	}

	namespaces := Store{Pool: db}

	n, _, err := namespaces.Get(
		query.Where("user_id", "=", query.Arg(u.ID)),
		query.Where("path", "=", query.Arg(p.Path)),
	)

	if err != nil {
		return nil, nil, &PathError{
			Path: *p,
			Err:  errors.Err(err),
		}
	}
	return u, n, nil
}

var rename = regexp.MustCompile("^[a-zA-Z0-9]+$")

// ResolveOrCreate will attempt to return the owner and namespace for the path.
// If no namespace can be resolved, then one is created, then subsequently
// returned.
func (p *Path) ResolveOrCreate(db database.Pool, userId int64) (*user.User, *Namespace, error) {
	u, n, err := p.Resolve(db, userId)

	if err != nil {
		return nil, nil, err
	}

	if n != nil {
		return u, n, nil
	}

	users := user.Store{Pool: db}

	u, ok, err := users.Get(user.WhereID(userId))

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	if !ok {
		return nil, nil, database.ErrNotFound
	}

	parts := strings.Split(p.Path, "/")

	var (
		parent *Namespace
		level  int64
	)

	for i, name := range parts {
		if level+1 > MaxDepth {
			break
		}

		if !rename.Match([]byte(name)) {
			if i == 0 && len(parts) == 1 {
				return nil, nil, ErrName
			}
			break
		}

		n = &Namespace{
			UserID:     userId,
			Name:       name,
			Path:       name,
			Level:      level + 1,
			Visibility: Private,
		}

		now := time.Now()

		if parent != nil {
			n.RootID = parent.RootID
			n.ParentID = sql.NullInt64{
				Int64: parent.ID,
				Valid: true,
			}
			n.Level = parent.Level + 1
			n.Path = strings.Join([]string{parent.Path, n.Name}, "/")
		}

		q := query.Insert(
			table,
			query.Columns("user_id", "root_id", "parent_id", "name", "path", "description", "level", "visibility", "created_at"),
			query.Values(n.UserID, n.RootID, n.ParentID, n.Name, n.Path, n.Description, n.Level, n.Visibility, now),
			query.Returning("id"),
		)

		if err := db.QueryRow(q.Build(), q.Args()...).Scan(&n.ID); err != nil {
			return nil, nil, errors.Err(err)
		}

		level++

		if parent == nil {
			n.RootID = sql.NullInt64{
				Int64: n.ID,
				Valid: true,
			}

			q = query.Update(
				table,
				query.Set("root_id", query.Arg(n.RootID)),
				query.Where("id", "=", query.Arg(n.ID)),
			)

			if _, err := db.Exec(q.Build(), q.Args()...); err != nil {
				return nil, nil, errors.Err(err)
			}
		}
		parent = n
	}
	return u, n, nil
}
