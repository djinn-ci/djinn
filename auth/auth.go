package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"djinn-ci.com/database"
)

type User struct {
	ID          int64
	Email       string
	Username    string
	Provider    string
	Permissions map[string]struct{}
	RawData     map[string]any
}

func (u *User) Grant(perm string) {
	if u.Permissions == nil {
		u.Permissions = make(map[string]struct{})
	}
	u.Permissions[perm] = struct{}{}
}

func (u *User) Has(perm string) bool {
	_, ok := u.Permissions[perm]
	return ok
}

func (u *User) Primary() (string, any) { return "id", u.ID }

func (u *User) Scan(r *database.Row) error {
	u.RawData = make(map[string]any, len(r.Columns))

	vals := make([]any, 0, len(r.Columns))

	for range r.Columns {
		vals = append(vals, &database.Any{})
	}

	if err := r.Scan(vals...); err != nil {
		return err
	}

	var ok bool

	for i, col := range r.Columns {
		a := vals[i].(*database.Any)

		u.RawData[col] = a.Value

		switch col {
		case "id":
			if u.ID, ok = a.Value.(int64); !ok {
				return fmt.Errorf("auth: *User.Scan: could not type assert type %T to %T", a.Value, u.ID)
			}
		case "email":
			if u.Email, ok = a.Value.(string); !ok {
				return fmt.Errorf("auth: *User.Scan:  could not type assert type %T to %T", a.Value, u.Email)
			}
		case "username":
			if u.Username, ok = a.Value.(string); !ok {
				return fmt.Errorf("auth: *User.Scan:  could not type assert type %T to %T", a.Value, u.Username)
			}
		}
	}
	return nil
}

func (u *User) Params() database.Params {
	immutable := map[string]struct{}{
		"id": {},
	}

	params := make(database.Params)

	for col, val := range u.RawData {
		param := database.CreateUpdateParam(val)

		if _, ok := immutable[col]; ok {
			param = database.ImmutableParam(val)
		}
		params[col] = param
	}
	return params
}

func (*User) Bind(database.Model)       {}
func (*User) Endpoint(...string) string { return "" }

func (u *User) MarshalJSON() ([]byte, error) {
	if u == nil {
		return []byte("null"), nil
	}

	raw := map[string]any{
		"email":    u.Email,
		"username": u.Username,
	}

	if v, ok := u.RawData["created_at"]; ok {
		if t, ok := v.(time.Time); ok {
			raw["created_at"] = t
		}
	}
	return json.Marshal(raw)
}

var (
	ErrAuth       = errors.New("auth: authentication failed")
	ErrPermission = errors.New("auth: permission denied")
)

type Authenticator interface {
	// Auth authenticates the given request, and upon success should return the
	// user from the given request. If authentication fails, then ErrAuth should
	// be returned as the error, if there is no user in the request. Otherwise,
	// ErrPermission should be returned if there is a user, but they lack the
	// necessary permissions.
	Auth(r *http.Request) (*User, error)
}

type AuthenticatorFunc func(r *http.Request) (*User, error)

func (fn AuthenticatorFunc) Auth(r *http.Request) (*User, error) {
	return fn(r)
}

type Store interface {
	Put(ctx context.Context, u *User) (*User, error)
}

func Persist(a Authenticator, s Store) Authenticator {
	return AuthenticatorFunc(func(r *http.Request) (*User, error) {
		u, err := a.Auth(r)

		if err != nil {
			return nil, err
		}
		return s.Put(r.Context(), u)
	})
}

// Fallback returns an authenticator that will fallback through the given
// authenticators should any of them fail with ErrAuth. If all of them fail with
// ErrAuth, then ErrAuth is returned as the error.
func Fallback(auths ...Authenticator) Authenticator {
	return AuthenticatorFunc(func(r *http.Request) (*User, error) {
		var (
			u   *User
			err error
		)

		for _, a := range auths {
			u, err = a.Auth(r)

			if err == nil {
				break
			}

			if !errors.Is(err, ErrAuth) {
				return nil, err
			}
		}

		if err != nil {
			return nil, err
		}
		return u, nil
	})
}

type HandlerFunc func(u *User, w http.ResponseWriter, r *http.Request)

type Registry struct {
	mu    sync.RWMutex
	field string
	auths map[string]Authenticator
}

func NewRegistry(field string) *Registry {
	return &Registry{
		field: field,
		auths: make(map[string]Authenticator),
	}
}

func (r *Registry) Register(name string, auth Authenticator) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.auths[name]; ok {
		panic("auth: authenticator already registered: " + name)
	}
	r.auths[name] = auth
}

func (r *Registry) Get(name string) (Authenticator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	auth, ok := r.auths[name]

	if !ok {
		return nil, errors.New("auth: unknown authenticator: " + name)
	}
	return auth, nil
}

func (r *Registry) Auth(req *http.Request) (*User, error) {
	if err := req.ParseForm(); err != nil {
		return nil, err
	}

	auth, err := r.Get(req.Form.Get(r.field))

	if err != nil {
		return nil, err
	}
	return auth.Auth(req)
}
