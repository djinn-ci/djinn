package namespace

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"djinn-ci.com/auth"
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

var reName = regexp.MustCompile("^[a-zA-Z0-9]+$")

// Resolve returns the owner and namespace for the namespace path. The given
// is used for the namespace owner if the current path has no Owner set. This
// returns *PathError on any errors. If the namespace does not exist then it
// is created.
func (p *Path) Resolve(ctx context.Context, pool *database.Pool, u *auth.User) (*auth.User, *Namespace, error) {
	if p.Owner != "" {
		owner, ok, err := user.NewStore(pool).Get(ctx, user.WhereUsername(p.Owner))

		if err != nil {
			return nil, nil, &PathError{Path: *p, Err: errors.Err(err)}
		}

		if !ok {
			return nil, nil, &PathError{Path: *p, Err: ErrOwner}
		}
		u = owner
	}

	n, _, err := NewStore(pool).Get(
		ctx,
		query.Where("user_id", "=", query.Arg(u.ID)),
		query.Where("path", "=", query.Arg(p.Path)),
	)

	if err != nil {
		return nil, nil, &PathError{Path: *p, Err: errors.Err(err)}
	}

	if err != nil {
		return nil, nil, err
	}

	if n != nil {
		return u, n, nil
	}

	parts := strings.Split(p.Path, "/")

	var (
		parent *Namespace
		level  int64
	)

	namespaces := Store{Store: NewStore(pool)}

	for i, name := range parts {
		if level+1 > MaxDepth {
			break
		}

		if !reName.Match([]byte(name)) {
			if i == 0 && len(parts) == 1 {
				return nil, nil, ErrName
			}
			break
		}

		params := Params{
			User: &auth.User{
				ID: u.ID,
			},
			Name:       name,
			Visibility: Private,
		}

		if parent != nil {
			params.Parent = parent.Path
		}

		n, err = namespaces.Create(ctx, &params)

		if err != nil {
			return nil, nil, &PathError{Path: *p, Err: errors.Err(err)}
		}
		parent = n
	}
	return u, n, nil
}
