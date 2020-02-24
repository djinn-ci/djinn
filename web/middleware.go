package web

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"

	"github.com/gorilla/mux"
)

type Middleware struct {
	Handler
}

// Gate serves as a stripped down middleware handler function that will be
// passed the current user in the request, if any, along with the request
// itself. This will determine whether the given user can access whatever is on
// the other end of the current endpoint, hence the bool return value.
type Gate func(u *model.User, r *http.Request) (*http.Request, bool)

// Get the currently authenticated user from the request. Check for token
// auth first, then fallback to cookie.
func (h Middleware) auth(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	if _, ok := r.Header["Authorization"]; ok {
		u, err := h.UserToken(r)

		if err != nil {
			log.Error.Println(errors.Err(err))
			return u, false
		}
		return u, !u.IsZero()
	}

	u, err := h.UserCookie(r)

	if err != nil {
		cause := errors.Cause(err)

		if strings.Contains(cause.Error(), "expired timestamp") {
			c := &http.Cookie{
				Name:     "user",
				HttpOnly: true,
				Path:     "/",
				Expires:  time.Unix(0, 0),
			}

			http.SetCookie(w, c)
		}
		return u, false
	}

	return u, !u.IsZero()
}

func (h Middleware) Guest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := h.auth(w, r); ok {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h Middleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := h.auth(w, r)

		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), "user", u))

		next.ServeHTTP(w, r)
	})
}

func (h Middleware) Gate(gates ...Gate) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, _ := h.auth(w, r)

			var ok bool

			r = r.WithContext(context.WithValue(r.Context(), "user", u))

			for _, g := range gates {
				r, ok = g(u, r)

				if !ok {
					HTMLError(w, "Not found", http.StatusNotFound)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
