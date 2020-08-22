package handler

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/user"
	usertemplate "github.com/andrewpillar/thrall/user/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
)

type User struct {
	web.Handler

	Prefix   string
	Registry *provider.Registry
}

func (h User) Register(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		p := &usertemplate.Register{
			Form: template.Form{
				CSRF:   string(csrf.TemplateField(r)),
				Errors: web.FormErrors(sess),
				Fields: web.FormFields(sess),
			},
		}
		save(r, w)
		web.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &user.RegisterForm{Users: h.Users}

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		h.RedirectBack(w, r)
		return
	}

	u, err := h.Users.Create(f.Email, f.Username, []byte(f.Password))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	providers := provider.NewStore(h.DB, u)

	cfgs, _ := h.Registry.All()

	for name := range cfgs {
		if _, err := providers.Create(0, name, nil, nil, false); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}
	h.Redirect(w, r, "/login")
}

func (h User) Login(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if r.Method == "GET" {
		cfgs, _ := h.Registry.All()

		order := make([]string, 0, len(cfgs))

		for name := range cfgs {
			order = append(order, name)
		}

		sort.Strings(order)

		pp := make([]*provider.Provider, 0, len(cfgs))

		for _, name := range order {
			cfg := cfgs[name]

			pp = append(pp, &provider.Provider{
				Name:    name,
				AuthURL: cfg.AuthCodeURL(cfg.Secret),
			})
		}

		p := &usertemplate.Login{
			Form: template.Form{
				CSRF:   string(csrf.TemplateField(r)),
				Errors: web.FormErrors(sess),
				Fields: web.FormFields(sess),
			},
			Providers: pp,
		}
		save(r, w)
		web.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &user.LoginForm{}

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		h.RedirectBack(w, r)
		return
	}

	u, err := h.Users.Auth(f.Handle, f.Password)

	if err != nil {
		cause := errors.Cause(err)

		if cause != user.ErrAuth {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
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
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		MaxAge:   user.MaxAge,
		Expires:  time.Now().Add(time.Duration(user.MaxAge) * time.Second),
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

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	m := make(map[string]*provider.Provider)

	pp, err := provider.NewStore(h.DB, u).All(query.OrderAsc("name"))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	for _, p := range pp {
		m[p.Name] = p
	}

	cfgs, _ := h.Registry.All()

	order := make([]string, 0, len(cfgs))

	for name, cfg := range cfgs {
		order = append(order, name)

		if _, ok := m[name]; !ok {
			m[name] = &provider.Provider{
				Name:    name,
				AuthURL: cfg.AuthCodeURL(cfg.Secret),
			}
		} else {
			m[name].AuthURL = cfg.AuthCodeURL(cfg.Secret)
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
			Errors: web.FormErrors(sess),
			Fields: web.FormFields(sess),
		},
		Providers: make([]*provider.Provider, 0, len(cfgs)),
	}

	for _, name := range order {
		p.Providers = append(p.Providers, m[name])
	}

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h User) Email(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	f := &user.EmailForm{
		User:  u,
		Users: h.Users,
	}

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update email"), "alert")
		h.RedirectBack(w, r)
		return
	}

	u.Email = f.Email
	u.UpdatedAt = time.Now()

	if err := h.Users.Update(u.ID, f.Email, []byte(f.VerifyPassword)); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update account"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Email has been updated"), "alert")
	h.RedirectBack(w, r)
}

func (h User) Password(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	f := &user.PasswordForm{
		User:  u,
		Users: h.Users,
	}

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update password"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Users.Update(u.ID, u.Email, []byte(f.NewPassword)); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update password"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Password has been updated"), "alert")
	h.RedirectBack(w, r)
}
