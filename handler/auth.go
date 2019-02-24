package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/auth"
	"github.com/andrewpillar/thrall/webutil"

	"golang.org/x/crypto/bcrypt"
)

var (
	maxAge = 5 * 365 * 86400
)

type Auth struct {
	Handler
}

func NewAuth(h Handler) Auth {
	return Auth{Handler: h}
}

func (h Auth) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		p := &auth.RegisterPage{
			Errors: h.errors(w, r),
			Form:   h.form(w, r),
		}

		webutil.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &form.Register{}

	if err := h.handleRequestData(f, w, r); err != nil {
		return
	}

	password, err := bcrypt.GenerateFromPassword([]byte(f.Password), bcrypt.DefaultCost)

	if err != nil {
		log.Error.Println(errors.Err(err))
		webutil.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	u := model.User{
		Email:    f.Email,
		Username: f.Username,
		Password: password,
	}

	if err := u.Create(); err != nil {
		log.Error.Println(errors.Err(err))
		webutil.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h Auth) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		p := &auth.LoginPage{
			Errors: h.errors(w, r),
			Form:   h.form(w, r),
		}

		webutil.HTML(w, template.Render(p), http.StatusOK)
		return
	}

	f := &form.Login{}

	if err := h.handleRequestData(f, w, r); err != nil {
		return
	}

	u, err := model.FindUserByHandle(f.Handle)

	if err != nil {
		log.Error.Println(errors.Err(err))
		webutil.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword(u.Password, []byte(f.Password)); err != nil {
		errs := form.NewErrors()
		errs.Put("handle", errors.New("Invalid login credentials"))
		errs.Put("password", errors.New("Invalid login credentials"))

		h.flashErrors(w, r, errs)
		h.flashForm(w, r, f)

		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	cookie := &http.Cookie{
		Name:     "user",
		HttpOnly: true,
	}

	if f.RememberMe {
		cookie.MaxAge = maxAge
		cookie.Expires = time.Now().Add(time.Duration(maxAge) * time.Second)
	}

	id := strconv.FormatInt(u.ID, 10)

	encoded, err := h.sc.Encode("user", id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		webutil.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	cookie.Value = encoded

	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
