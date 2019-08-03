package web

import (
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

	Builds    model.BuildStore
	Resources model.ResourceStore
	Users     model.UserStore
}

func (h Middleware) auth(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	u, err := h.User(r)

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
		if _, ok := h.auth(w, r); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h Middleware) AuthResource(name string) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := h.auth(w, r)

			if !ok {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			vars := mux.Vars(r)

			if _, ok := vars[name]; !ok {
				next.ServeHTTP(w, r)
				return
			}

			res, err := h.Resources.Find(name, vars)

			if err != nil {
				log.Error.Println(errors.Err(err))
				HTMLError(w, "Something went wrong", http.StatusInternalServerError)
				return
			}

			if res.IsZero() || !res.AccessibleBy(u) {
				HTMLError(w, "Not found", http.StatusNotFound)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
