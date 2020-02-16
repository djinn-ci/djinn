package web

import (
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"

	"github.com/andrewpillar/query"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

var sessionName = "session"

type Handler struct {
	Store        sessions.Store
	SecureCookie *securecookie.SecureCookie
	Users        model.UserStore
}

func (h *Handler) FlashPut(w http.ResponseWriter, r *http.Request, key, val interface{}) {
	sess, _ := h.Store.Get(r, sessionName)
	sess.Values[key] = val

	if err := sess.Save(r, w); err != nil {
		log.Error.Println("failed to save session: " + errors.Err(err).Error())
	}
}

func (h *Handler) FlashGet(w http.ResponseWriter, r *http.Request, key interface{}) interface{} {
	sess, _ := h.Store.Get(r, sessionName)

	val, ok := sess.Values[key]

	if ok {
		delete(sess.Values, key)
		if err := sess.Save(r, w); err != nil {
			log.Error.Println("failed to save session: " + errors.Err(err).Error())
		}
	}
	return val
}

func (h *Handler) FlashAlert(w http.ResponseWriter, r *http.Request, a template.Alert) {
	h.FlashPut(w, r, "alert", a)
}

func (h *Handler) FlashErrors(w http.ResponseWriter, r *http.Request, e form.Errors) {
	h.FlashPut(w, r, "form_errors", e)
}

func (h *Handler) FlashForm(w http.ResponseWriter, r *http.Request, f form.Form) {
	h.FlashPut(w, r, "form_fields", f.Fields())
}

func (h *Handler) Alert(w http.ResponseWriter, r *http.Request) template.Alert {
	val := h.FlashGet(w, r, "alert")

	if val != nil {
		return val.(template.Alert)
	}
	return template.Alert{}
}

func (h *Handler) Errors(w http.ResponseWriter, r *http.Request) form.Errors {
	val := h.FlashGet(w, r, "form_errors")

	if val != nil {
		return val.(form.Errors)
	}
	return form.NewErrors()
}

func (h *Handler) Form(w http.ResponseWriter, r *http.Request) map[string]string {
	val := h.FlashGet(w, r, "form_fields")

	if val != nil {
		return val.(map[string]string)
	}
	return map[string]string{}
}

// Validate the given form. If form validation fails with form.Errors then the
// form and error will be flashed to the session. If any other error occurs
// then that will be flashed as an alert. This will return the first error
// that occurs, if an error is returned it is expected for the caller to
// redirect back.
func (h *Handler) ValidateForm(f form.Form, w http.ResponseWriter, r *http.Request) error {
	if err := form.Unmarshal(f, r); err != nil {
		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to unmarshal form: " + cause.Error()))

		return errors.Err(err)
	}

	if err := f.Validate(); err != nil {
		if ferr, ok := err.(form.Errors); ok {
			h.FlashErrors(w, r, ferr)
			h.FlashForm(w, r, f)
			return ferr
		}
		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to validate form: " + cause.Error()))
		return errors.Err(err)
	}

	return nil
}

func (h Handler) User(r *http.Request) *model.User {
	val := r.Context().Value("user")

	u, _ := val.(*model.User)

	if u != nil {
		return u
	}

	u, _ = h.UserCookie(r)

	return u
}

func (h Handler) UserCookie(r *http.Request) (*model.User, error) {
	c, err := r.Cookie("user")

	if err != nil {
		if err == http.ErrNoCookie {
			return &model.User{}, nil
		}

		return &model.User{}, errors.Err(err)
	}

	var s string

	if err := h.SecureCookie.Decode("user", c.Value, &s); err != nil {
		return &model.User{}, errors.Err(err)
	}

	id, err := strconv.ParseInt(s, 10, 64)

	if err != nil {
		return &model.User{}, nil
	}

	u, err := h.Users.Get(query.Where("id", "=", id))

	if u.DeletedAt.Valid {
		return &model.User{}, nil
	}

	return u, errors.Err(err)
}

func (h Handler) UserToken(r *http.Request) (*model.User, error) {
	id := r.Header.Get("Authorization")

	u, err := h.Users.Get(query.Where("id", "=", id))

	if u.DeletedAt.Valid {
		return &model.User{}, nil
	}

	return u, errors.Err(err)
}
