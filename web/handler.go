package web

import (
	"encoding/hex"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/log"
	"github.com/andrewpillar/djinn/oauth2"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"

	"github.com/jmoiron/sqlx"
)

const sessionName = "session"

// Handler is the main type for embedding into request handlers.
type Handler struct {
	// DB is the current connection to the database.
	DB *sqlx.DB

	SMTP struct {
		Client *smtp.Client
		From   string
	}

	// Log is the log file that the HTTP server is writing to.
	Log *log.Logger

	// Store is the current store being used for storing session data, such
	// as form fields, form errors, and general error messages.
	Store sessions.Store

	// Users is a pointer to the user.Store. This would typically be used for
	// getting the currently authenticated user from the database.
	Users *user.Store

	// SecureCookie is what is used to encrypt the data we store inside the
	// request cookies.
	SecureCookie *securecookie.SecureCookie
}

// Redirect redirects to the given URL, and saves the session in the process.
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request, url string) {
	_, save := h.Session(r)
	save(r, w)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

// RedirectBack redirects to the Referer of the given request.
func (h *Handler) RedirectBack(w http.ResponseWriter, r *http.Request) {
	h.Redirect(w, r, r.Header.Get("Referer"))
}

// SaveMiddleware will save any data put in the session before serving the next
// request.
func (h *Handler) SaveMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, save := h.Session(r)
		save(r, w)
		handler.ServeHTTP(w, r)
	})
}

// Session returns the session for the current request, and a callback for
// saving the returned session.
func (h *Handler) Session(r *http.Request) (*sessions.Session, func(*http.Request, http.ResponseWriter)) {
	sess, _ := h.Store.Get(r, sessionName)

	return sess, func(r *http.Request, w http.ResponseWriter) {
		if err := sess.Save(r, w); err != nil {
			h.Log.Error.Println(r.Method, r.URL, "failed to save session", errors.Err(err))
		}
	}
}

func (h Handler) UserFromCookie(r *http.Request) (*user.User, bool, error) {
	c, err := r.Cookie("user")

	if err != nil {
		if err == http.ErrNoCookie {
			return nil, false, nil
		}
		return nil, false, errors.Err(err)
	}

	var s string

	if err := h.SecureCookie.Decode("user", c.Value, &s); err != nil {
		return nil, false, errors.Err(err)
	}

	id, err := strconv.ParseInt(s, 10, 64)

	if err != nil {
		return nil, false, nil
	}

	u, err := h.Users.Get(query.Where("id", "=", query.Arg(id)))

	if u.DeletedAt.Valid {
		return nil, false, nil
	}

	for _, res := range oauth2.Resources {
		u.SetPermission(res.String() + ":read")
		u.SetPermission(res.String() + ":write")
		u.SetPermission(res.String() + ":delete")
	}
	return u, !u.IsZero(), errors.Err(err)
}

func (h Middleware) UserFromToken(r *http.Request) (*user.User, bool, error) {
	prefix := "Bearer "
	tok := r.Header.Get("Authorization")

	if !strings.HasPrefix(tok, prefix) {
		return nil, false, nil
	}

	b, err := hex.DecodeString(tok[len(prefix):])

	if err != nil {
		return nil, false, errors.Err(err)
	}

	t, err := h.Tokens.Get(query.Where("token", "=", query.Arg(b)))

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if t.IsZero() {
		return nil, false, nil
	}

	u, err := h.Users.Get(query.Where("id", "=", query.Arg(t.UserID)))

	if u.DeletedAt.Valid {
		return nil, false, nil
	}

	u.Permissions = t.Permissions()
	return u, !u.IsZero(), errors.Err(err)
}
