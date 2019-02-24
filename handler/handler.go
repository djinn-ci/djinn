package handler

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
	sc    *securecookie.SecureCookie
	store  sessions.Store
}

func html(w http.ResponseWriter, content string, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(content))
}

func New(sc *securecookie.SecureCookie, store sessions.Store) Handler {
	return Handler{sc: sc, store: store}
}

func HTMLError(w http.ResponseWriter, message string, status int) {
	p := &template.Error{
		Code:    status,
		Message: message,
	}

	html(w, template.Render(p), status)
}

func (h *Handler) handleRequestData(f form.Form, w http.ResponseWriter, r *http.Request) error {
	if err := form.UnmarshalAndValidate(f, r); err != nil {
		e, ok := err.(form.Errors)

		if !ok {
			log.Error.Println(errors.Err(err))
			HTMLError(w, "Something went wrong", http.StatusInternalServerError)

			return errors.Err(errors.New("failed to handle request data"))
		}

		h.flashErrors(w, r, e)
		h.flashForm(w, r, f)

		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)

		return errors.Err(errors.New("request data failed validation"))
	}

	return nil
}

func (h *Handler) errors(w http.ResponseWriter, r *http.Request) form.Errors {
	sess, _ := h.store.Get(r, sessionName)

	e, ok := sess.Values["errors"]

	if ok {
		delete(sess.Values, "errors")

		if err := sess.Save(r, w); err != nil {
			log.Error.Println(errors.Err(err))
		}

		return e.(form.Errors)
	}

	return form.NewErrors()
}

func (h *Handler) form(w http.ResponseWriter, r *http.Request) form.Form {
	sess, _ := h.store.Get(r, sessionName)

	f, ok := sess.Values["form"]

	if ok {
		delete(sess.Values, "form")

		if err := sess.Save(r, w); err != nil {
			log.Error.Println(errors.Err(err))
		}

		return f.(form.Form)
	}

	return form.Empty()
}

func (h *Handler) flashErrors(w http.ResponseWriter, r *http.Request, e form.Errors) {
	sess, _ := h.store.Get(r, sessionName)

	sess.Values["errors"] = e

	if err := sess.Save(r, w); err != nil {
		log.Error.Println(errors.Err(err))
	}
}

func (h *Handler) flashForm(w http.ResponseWriter, r *http.Request, f form.Form) {
	sess, _ := h.store.Get(r, sessionName)

	sess.Values["form"] = f

	if err := sess.Save(r, w); err != nil {
		log.Error.Println(errors.Err(err))
	}
}

func (h *Handler) UserFromRequest(r *http.Request) (*model.User, error) {
	cookie, err := r.Cookie("user")

	if err != nil {
		if err == http.ErrNoCookie {
			return &model.User{}, nil
		}

		return &model.User{}, errors.Err(err)
	}

	var str string

	if err := h.sc.Decode("user", cookie.Value, &str); err != nil {
		return &model.User{}, errors.Err(err)
	}

	id, err := strconv.ParseInt(str, 10, 64)

	if err != nil {
		return &model.User{}, nil
	}

	u, err := model.FindUser(id)

	if err != nil {
		return &model.User{}, errors.Err(err)
	}

	return u, err
}
