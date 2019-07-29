package web

import (
	"net/http"
	"strconv"
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
	Objects   model.ObjectStore
	Users     model.UserStore
	Variables model.VariableStore
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

func (h Middleware) AuthBuild(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := h.auth(w, r)

		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		vars := mux.Vars(r)

		if _, ok := vars["build"]; !ok {
			next.ServeHTTP(w, r)
			return
		}

		id, _ := strconv.ParseInt(vars["build"], 10, 64)

		b, err := h.Builds.Find(id)

		if err != nil {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if b.IsZero() || u.ID != b.UserID {
			HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h Middleware) AuthNamespace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := h.auth(w, r)

		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		vars := mux.Vars(r)

		owner, err := h.Users.FindByUsername(vars["username"])

		if err != nil {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if owner.IsZero() {
			HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		n, err := owner.NamespaceStore().FindByPath(strings.TrimSuffix(vars["namespace"], "/"))

		if err != nil {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if n.IsZero() {
			HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		switch n.Visibility {
			case model.Private:
				if n.UserID != u.ID {
					HTMLError(w, "Not found", http.StatusNotFound)
					return
				}

				break
			case model.Internal:
				if u.IsZero() {
					HTMLError(w, "Not found", http.StatusNotFound)
					return
				}

				break
			case model.Public:
				break
		}

		next.ServeHTTP(w, r)
	})
}

func (h Middleware) AuthObject(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := h.auth(w, r)

		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		vars := mux.Vars(r)

		if _, ok := vars["object"]; !ok {
			next.ServeHTTP(w, r)
			return
		}

		id, _ := strconv.ParseInt(vars["object"], 10, 64)

		o, err := h.Objects.Find(id)

		if err != nil {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if o.IsZero() || u.ID != o.UserID || o.DeletedAt.Valid {
			HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h Middleware) AuthVariable(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := h.auth(w, r)

		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		vars := mux.Vars(r)

		if _, ok := vars["variable"]; !ok {
			next.ServeHTTP(w, r)
			return
		}

		id, _ := strconv.ParseInt(vars["variable"], 10, 64)

		v, err := h.Variables.Find(id)

		if err != nil {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if v.IsZero() || u.ID != v.UserID {
			HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
