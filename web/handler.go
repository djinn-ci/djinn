package web

import (
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

var sessionName = "session"

// Zero-value form implementation.
type Form map[string]string

type Handler struct {
	store sessions.Store

	SecureCookie *securecookie.SecureCookie
	Users        *model.UserStore
}

func New(sc *securecookie.SecureCookie, store sessions.Store, users *model.UserStore) Handler {
	return Handler{
		store:        store,
		SecureCookie: sc,
		Users:        users,
	}
}

func (f Form) Get(key string) string {
	return f[key]
}

func (f Form) Validate() error {
	return nil
}

func (h *Handler) FlashErrors(w http.ResponseWriter, r *http.Request, e form.Errors) {
	sess, _ := h.store.Get(r, sessionName)

	sess.Values["Errors"] = e

	if err := sess.Save(r, w); err != nil {
		log.Error.Println(errors.Err(err))
	}
}

func (h *Handler) FlashForm(w http.ResponseWriter, r *http.Request, f form.Form) {
	sess, _ := h.store.Get(r, sessionName)

	sess.Values["form"] = f

	if err := sess.Save(r, w); err != nil {
		log.Error.Println(errors.Err(err))
	}
}

func (h *Handler) Errors(w http.ResponseWriter, r *http.Request) form.Errors {
	sess, _ := h.store.Get(r, sessionName)

	e, ok := sess.Values["Errors"]

	if ok {
		delete(sess.Values, "Errors")

		if err := sess.Save(r, w); err != nil {
			log.Error.Println(errors.Err(err))
		}

		return e.(form.Errors)
	}

	return form.NewErrors()
}

func (h *Handler) Form(w http.ResponseWriter, r *http.Request) form.Form {
	sess, _ := h.store.Get(r, sessionName)

	f, ok := sess.Values["form"]

	if ok {
		delete(sess.Values, "form")

		if err := sess.Save(r, w); err != nil {
			log.Error.Println(errors.Err(err))
		}

		return f.(form.Form)
	}

	return Form(make(map[string]string))
}

func (h *Handler) ValidateForm(f form.Form, w http.ResponseWriter, r *http.Request) error {
	if err := form.Unmarshal(f, r); err != nil {
		return errors.Err(err)
	}

	if err := f.Validate(); err != nil {
		h.FlashErrors(w, r, err.(form.Errors))
		h.FlashForm(w, r, f)

		return err
	}

	return nil
}

func (h Handler) User(r *http.Request) (*model.User, error) {
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

	u, err := h.Users.Find(id)

	if u.Deleted() {
		return &model.User{}, nil
	}

	return u, errors.Err(err)
}
