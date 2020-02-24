package ui

import (
	"net/http"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/auth"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"

	"golang.org/x/crypto/bcrypt"
)

var maxAge = 5 * 365 * 86400

type Auth struct {
	web.Handler

	Providers map[string]oauth2.Provider
}

func (h Auth) Register(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)
	defer save(r, w)

	if r.Method == "GET" {
		p := &auth.RegisterPage{
			Form: template.Form{
				CSRF:   string(csrf.TemplateField(r)),
				Errors: h.FormErrors(sess),
				Fields: h.FormFields(sess),
			},
		}
		web.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &form.Register{
		Users: h.Users,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); !ok {
			log.Error.Println(errors.Err(err))
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	password, err := bcrypt.GenerateFromPassword([]byte(f.Password), bcrypt.DefaultCost)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	u := h.Users.New()
	u.Email = f.Email
	u.Username = f.Username
	u.Password = password

	if err := h.Users.Create(u); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h Auth) Login(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)
	defer save(r, w)

	if r.Method == "GET" {
		p := &auth.LoginPage{
			Form: template.Form{
				CSRF:   string(csrf.TemplateField(r)),
				Errors: h.FormErrors(sess),
				Fields: h.FormFields(sess),
			},
		}
		web.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &form.Login{}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); !ok {
			log.Error.Println(errors.Err(err))
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	u, err := h.Users.Auth(f.Handle, f.Password)

	if err != nil {
		if errors.Cause(err) != model.ErrAuth {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		errs := form.NewErrors()
		errs.Put("handle", model.ErrAuth)
		errs.Put("password", model.ErrAuth)

		sess.AddFlash(errs, "form_errors")
		sess.AddFlash(f.Fields(), "form_fields")

		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	cookie := &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge)*time.Second),
	}

	id := strconv.FormatInt(u.ID, 10)

	encoded, err := h.SecureCookie.Encode("user", id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	cookie.Value = encoded

	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h Auth) Logout(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Unix(0, 0),
	}

	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
