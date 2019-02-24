package handler

import (
	"net/http"

	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/webutil"
)

type gateHandler func(u *model.User) bool

type Middleware struct {
	Handler
}

func NewMiddleware(h Handler) Middleware {
	return Middleware{Handler: h}
}

func (h Middleware) gate(next http.HandlerFunc, handler gateHandler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := h.UserFromRequest(r)

		if err != nil {
			log.Error.Println(err)
			webutil.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if !handler(u) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		next(w, r)
	})
}

func (h Middleware) Auth(next http.HandlerFunc) http.HandlerFunc {
	return h.gate(next, func(u *model.User) bool {
		return !u.IsZero()
	})
}

func (h Middleware) Guest(next http.HandlerFunc) http.HandlerFunc {
	return h.gate(next, func(u *model.User) bool {
		return u.IsZero()
	})
}
