package web

import (
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"

	"github.com/jmoiron/sqlx"
)

var sessionName = "session"

type Handler struct {
	DB *sqlx.DB

	Store        sessions.Store
	SecureCookie *securecookie.SecureCookie
	Users        user.Store
	Tokens       oauth2.TokenStore
}

func (h *Handler) Alert(sess *sessions.Session) template.Alert {
	val := sess.Flashes("alert")

	if val == nil {
		return template.Alert{}
	}
	return val[0].(template.Alert)
}

func (h *Handler) FormErrors(sess *sessions.Session) form.Errors {
	val := sess.Flashes("form_errors")

	if val == nil {
		return form.NewErrors()
	}
	return val[0].(form.Errors)
}

func (h *Handler) FormFields(sess *sessions.Session) map[string]string {
	val := sess.Flashes("form_fields")

	if val == nil {
		return map[string]string{}
	}
	return val[0].(map[string]string)
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request, url string) {
	_, save := h.Session(r)
	save(r, w)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func (h *Handler) RedirectBack(w http.ResponseWriter, r *http.Request) {
	h.Redirect(w, r, r.Header.Get("Referer"))
}

func (h *Handler) Session(r *http.Request) (*sessions.Session, func(*http.Request, http.ResponseWriter)) {
	sess, _ := h.Store.Get(r, sessionName)

	return sess, func(r *http.Request, w http.ResponseWriter) {
		if err := sess.Save(r, w); err != nil {
			log.Error.Println(r.Method, r.URL.Path, "failed to save session", errors.Err(err))
		}
	}
}

func (h Handler) User(r *http.Request) *user.User {
	val := r.Context().Value("user")

	u, _ := val.(*user.User)

	if u == nil {
		u, _ = h.UserCookie(r)
	}
	return u
}

func (h Handler) UserCookie(r *http.Request) (*user.User, error) {
	c, err := r.Cookie("user")

	if err != nil {
		if err == http.ErrNoCookie {
			return &user.User{}, nil
		}
		return &user.User{}, errors.Err(err)
	}

	var s string

	if err := h.SecureCookie.Decode("user", c.Value, &s); err != nil {
		return &user.User{}, errors.Err(err)
	}

	id, err := strconv.ParseInt(s, 10, 64)

	if err != nil {
		return &user.User{}, nil
	}

	u, err := h.Users.Get(query.Where("id", "=", id))

	if u.DeletedAt.Valid {
		return &user.User{}, nil
	}
	return u, errors.Err(err)
}

func (h Handler) UserToken(r *http.Request) (*user.User, error) {
	prefix := "Bearer "
	tok := r.Header.Get("Authorization")

	if !strings.HasPrefix(tok, prefix) {
		return &user.User{}, nil
	}

	b, err := hex.DecodeString(tok[len(prefix):])

	if err != nil {
		return &user.User{}, errors.Err(err)
	}

	t, err := h.Tokens.Get(query.Where("token", "=", b))

	if err != nil {
		return &user.User{}, errors.Err(err)
	}

	if t.IsZero() {
		return &user.User{}, nil
	}

	u, err := h.Users.Get(query.Where("id", "=", t.UserID))

	if u.DeletedAt.Valid {
		return &user.User{}, nil
	}
	return u, errors.Err(err)
}

func (h *Handler) ValidateForm(f form.Form, r *http.Request, sess *sessions.Session) error {
	if err := form.Unmarshal(f, r); err != nil {
		if sess != nil {
			cause := errors.Cause(err)
			sess.AddFlash(template.Danger("Failed to unmarshal form: " + cause.Error()), "alert")
		}
		return errors.Err(err)
	}

	if err := f.Validate(); err != nil {
		if ferr, ok := err.(form.Errors); ok {
			if sess != nil {
				sess.AddFlash(ferr, "form_errors")
				sess.AddFlash(f.Fields(), "form_fields")
			}
			return ferr
		}

		if sess != nil {
			cause := errors.Cause(err)
			sess.AddFlash(template.Danger("Failed to validate form: " + cause.Error()), "alert")
		}
		return errors.Err(err)
	}
	return nil
}
