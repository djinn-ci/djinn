package web

import (
	"net/http"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
)

type gate func(u *model.User) (string, bool)

type Middleware struct {
	Handler
}

func NewMiddleware(h Handler) Middleware {
	return Middleware{
		Handler: h,
	}
}

func (h Middleware) gate(next http.HandlerFunc, g gate) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}

			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if url, ok := g(u); !ok {
			http.Redirect(w, r, url, http.StatusSeeOther)
			return
		}

		next(w, r)
	})
}

func (h Middleware) Auth(next http.HandlerFunc) http.HandlerFunc {
	return h.gate(next, func(u *model.User) (string, bool) {
		return "/login", !u.IsZero()
	})
}

func (h Middleware) Guest(next http.HandlerFunc) http.HandlerFunc {
	return h.gate(next, func(u *model.User) (string, bool) {
		return "/", u.IsZero()
	})
}
