package handler

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/template"
	usertemplate "github.com/andrewpillar/thrall/user/template"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	web.Handler

	Providers map[string]oauth2.Provider
}

var maxAge = 5*365*86400

func (h User) Register(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		p := &usertemplate.Register{
			Form: template.Form{
				CSRF:   string(csrf.TemplateField(r)),
				Errors: h.FormErrors(sess),
				Fields: h.FormFields(sess),
			},
		}
		save(r, w)
		web.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &user.RegisterForm{Users: h.Users}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); !ok {
			log.Error.Println(errors.Err(err))
		}
		h.RedirectBack(w, r)
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

	providers := provider.NewStore(h.DB)

	for name := range h.Providers {
		p := providers.New()
		p.Name = name
		p.UserID = u.ID
		p.Connected = false

		if err := providers.Create(p); err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}
	h.Redirect(w, r, "/login")
}

func (h User) Login(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		p := &usertemplate.Login{
			Form: template.Form{
				CSRF:   string(csrf.TemplateField(r)),
				Errors: h.FormErrors(sess),
				Fields: h.FormFields(sess),
			},
		}
		save(r, w)
		web.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &user.LoginForm{}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); !ok {
			log.Error.Println(errors.Err(err))
		}
		h.RedirectBack(w, r)
		return
	}

	u, err := h.Users.Auth(f.Handle, f.Password)

	if err != nil {
		cause := errors.Cause(err)

		if cause != user.ErrAuth {
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to login:"+cause.Error()), "alert")
			h.RedirectBack(w, r)
			return
		}

		errs := form.NewErrors()
		errs.Put("handle", user.ErrAuth)
		errs.Put("password", user.ErrAuth)

		sess.AddFlash(errs, "form_errors")
		sess.AddFlash(f.Fields(), "form_fields")
		h.RedirectBack(w, r)
		return
	}

	id := strconv.FormatInt(u.ID, 10)

	encoded, err := h.SecureCookie.Encode("user", id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge)*time.Second),
		Value:    encoded,
	})
	h.Redirect(w, r, "/")
}

func (h User) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		Path:     "/",
		Expires:  time.Unix(0, 0),
	})
	h.Redirect(w, r, "/")
}

func (h User) Settings(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)
	m := make(map[string]*provider.Provider)

	pp, err := provider.NewStore(h.DB, u).All(query.OrderAsc("name"))

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	for _, p := range pp {
		m[p.Name] = p
	}

	order := make([]string, 0, len(h.Providers))

	for name, p := range h.Providers {
		order = append(order, name)

		if _, ok := m[name]; !ok {
			m[name] = &provider.Provider{
				Name:    name,
				AuthURL: p.AuthURL(),
			}
		} else {
			m[name].AuthURL = p.AuthURL()
		}
	}

	sort.Strings(order)

	p := &usertemplate.Settings{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.FormErrors(sess),
			Fields: h.FormFields(sess),
		},
		Providers: make([]*provider.Provider, 0, len(h.Providers)),
	}

	for _, name := range order {
		p.Providers = append(p.Providers, m[name])
	}

	d := template.NewDashboard(p, r.URL, h.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h User) Email(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u := h.User(r)

	f := &user.EmailForm{
		User:      u,
		Users:     h.Users,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); !ok {
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to update password"), "alert")
		}
		h.RedirectBack(w, r)
		return
	}

	u.Email = f.Email
	u.UpdatedAt = time.Now()

	if err := h.Users.Update(u); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update account"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Email has been updated"), "alert")
	h.RedirectBack(w, r)
}

func (h User) Password(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u := h.User(r)

	f := &user.PasswordForm{
		User:  u,
		Users: h.Users,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); !ok {
			log.Error.Println(errors.Err(err))
			sess.AddFlash(template.Danger("Failed to update password"), "alert")
		}
		h.RedirectBack(w, r)
		return
	}

	password, err := bcrypt.GenerateFromPassword([]byte(f.NewPassword), bcrypt.DefaultCost)

	if err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update password"), "alert")
		h.RedirectBack(w, r)
		return
	}

	u.Password = password
	u.UpdatedAt = time.Now()

	if err := h.Users.Update(u); err != nil {
		log.Error.Println(errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update password"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Password has been updated"), "alert")
	h.RedirectBack(w, r)
}
