package web

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/oauth2"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/jmoiron/sqlx"
)

type Middleware struct {
	Handler

	Tokens *oauth2.TokenStore
}

type databaseFunc func(int64) (database.Model, error)

type errorHandler func(http.ResponseWriter, string, int)

// Gate serves as a stripped down middleware handler function that will be
// passed the current user in the request, if any, along with the request
// itself. This will determine whether the given user can access whatever is on
// the other end of the current endpoint, hence the bool return value.
type Gate func(u *user.User, r *http.Request) (*http.Request, bool, error)

// CanAccessResource returns whether the current user has access to the given
// resource. The resource's ID will be taken from the request based on the
// name, this is passed back to the databaseFunc which will return the underlying
// model for that resource. The name of the resource is also used to check
// against the permissions of that user.
func CanAccessResource(db *sqlx.DB, name string, r *http.Request, get databaseFunc) (bool, error) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		return false, nil
	}

	switch r.Method {
	case "GET":
		_, ok = u.Permissions[name+":read"]
	case "POST", "PATCH":
		_, ok = u.Permissions[name+":write"]
	case "DELETE":
		_, ok = u.Permissions[name+":delete"]
	}

	if !ok {
		return false, nil
	}

	base := BasePath(r.URL.Path)

	if base == "/" || base == "create" || base == name || base == name+"s" {
		return ok, nil
	}

	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars[name], 10, 64)

	m, err := get(id)

	if err != nil {
		return false, errors.Err(err)
	}

	if m.IsZero() {
		return false, nil
	}

	namespaceId, _ := m.Values()["namespace_id"].(sql.NullInt64)

	if !namespaceId.Valid {
		userId, ok := m.Values()["user_id"].(int64)

		if !ok {
			return false, nil
		}
		return u.ID == userId, nil
	}

	root, err := namespace.NewStore(db).Get(
		query.Where("root_id", "=", namespace.SelectRootID(namespaceId.Int64)),
		query.Where("id", "=", namespace.SelectRootID(namespaceId.Int64)),
	)

	if err != nil {
		return false, errors.Err(err)
	}

	cc, err := namespace.NewCollaboratorStore(db, root).All()

	if err != nil {
		return false, errors.Err(err)
	}

	root.LoadCollaborators(cc)
	return root.AccessibleBy(u), nil
}

// Get the currently authenticated user from the request. Check for token
// auth first, then fallback to cookie.
func (h Middleware) userFromRequest(w http.ResponseWriter, r *http.Request) (*user.User, bool, error) {
	if _, ok := r.Header["Authorization"]; ok {
		u, ok, err := h.UserFromToken(r)

		if err != nil {
			return nil, ok, errors.Err(err)
		}
		return u, ok, nil
	}

	u, ok, err := h.UserFromCookie(r)

	if err != nil {
		cause := errors.Cause(err)

		if !strings.Contains(cause.Error(), "expired timestamp") {
			return nil, false, errors.Err(err)
		}

		c := &http.Cookie{
			Name:     "user",
			HttpOnly: true,
			Path:     "/",
			Expires:  time.Unix(0, 0),
		}

		http.SetCookie(w, c)
		return nil, false, nil
	}
	return u, ok, nil
}

// Guest redirects the user back to the homepage if they're already
// authenticated. Otherwise it let's them continue with the request.
func (h Middleware) Guest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errh errorHandler = HTMLError

		json := strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")

		if json {
			errh = JSONError
		}

		_, ok, err := h.userFromRequest(w, r)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			errh(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if ok {
			if json {
				JSONError(w, "Not found", http.StatusNotFound)
				return
			}
			h.Redirect(w, r, "/")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Auth redirects the user back to /login if they're not authenticated, however
// it let's them continue if they are, and set's the user in the request
// context.
func (h Middleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errh errorHandler = HTMLError

		json := strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")

		if json {
			errh = JSONError
		}

		u, ok, err := h.userFromRequest(w, r)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			errh(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if !ok {
			if json {
				JSONError(w, "Not found", http.StatusNotFound)
				return
			}
			h.Redirect(w, r, "/login?redirect_uri="+url.PathEscape(BaseAddress(r)+r.URL.String()))
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), "user", u))
		next.ServeHTTP(w, r)
	})
}

// Gate returns a mux.MiddlewareFunc that when called will iterate over the
// given gates to determine if the user can access the next request.
func (h Middleware) Gate(gates ...Gate) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var errh errorHandler = HTMLError

			json := strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")

			if json {
				errh = JSONError
			}

			u, ok, err := h.userFromRequest(w, r)

			if !ok {
				u = &user.User{}
			}

			if err != nil {
				h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
				errh(w, "Something went wrong", http.StatusInternalServerError)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), "user", u))

			for _, gate := range gates {
				r, ok, err = gate(u, r)

				if err != nil {
					h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
					errh(w, "Something went wrong", http.StatusInternalServerError)
					return
				}

				if !ok {
					errh(w, "Not found", http.StatusNotFound)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
