package web

import (
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

var sessionName = "session"

type Handler struct {
	store sessions.Store

	SecureCookie *securecookie.SecureCookie
	Users        model.UserStore
}

func New(sc *securecookie.SecureCookie, store sessions.Store, users model.UserStore) Handler {
	return Handler{
		store:        store,
		SecureCookie: sc,
		Users:        users,
	}
}

func (h *Handler) FlashAlert(w http.ResponseWriter, r *http.Request, a template.Alert) {
	sess, _ := h.store.Get(r, sessionName)

	sess.Values["alert"] = a

	if err := sess.Save(r, w); err != nil {
		log.Error.Println(errors.Err(err))
	}
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

	sess.Values["form"] = f.Fields()

	if err := sess.Save(r, w); err != nil {
		log.Error.Println(errors.Err(err))
	}
}

func (h *Handler) Alert(w http.ResponseWriter, r *http.Request) template.Alert {
	sess, _ := h.store.Get(r, sessionName)

	a, ok := sess.Values["alert"]

	if ok {
		delete(sess.Values, "alert")

		if err := sess.Save(r, w); err != nil {
			log.Error.Println(errors.Err(err))
		}

		return a.(template.Alert)
	}

	return template.Alert{}
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

func (h *Handler) Form(w http.ResponseWriter, r *http.Request) map[string]string {
	sess, _ := h.store.Get(r, sessionName)

	f, ok := sess.Values["form"]

	if ok {
		delete(sess.Values, "form")

		if err := sess.Save(r, w); err != nil {
			log.Error.Println(errors.Err(err))
		}

		return f.(map[string]string)
	}

	return make(map[string]string)
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

	if u.DeletedAt.Valid {
		return &model.User{}, nil
	}

	return u, errors.Err(err)
}
