package web

import (
	"net/http"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/log"
	"github.com/andrewpillar/djinn/user"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"

	"github.com/jmoiron/sqlx"
)

const sessionName = "session"

// Handler is the main type for embedding into request handlers.
type Handler struct {
	// DB is the current connection to the database.
	DB *sqlx.DB

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
