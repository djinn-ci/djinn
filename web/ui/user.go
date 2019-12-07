package ui

import (
	"net/http"
	"path/filepath"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/user"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	web.Handler

	Invites   model.InviteStore
	Providers map[string]oauth2.Provider
}

func (h User) Settings(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	base := filepath.Base(r.URL.Path)

	var p template.Dashboard

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}

	sp := user.SettingsPage{
		BasePage: bp,
		Form:     template.Form{
			Errors: h.Errors(w, r),
			Fields: h.Form(w, r),
		},
		CSRF:     string(csrf.TemplateField(r)),
	}

	switch base {
	case "invites":
		ii, err := h.Invites.Index(query.Where("invitee_id", "=", u.ID))

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p = &user.ShowInvites{
			SettingsPage: sp,
			CSRF:         string(csrf.TemplateField(r)),
			Invites:      ii,
		}

		break
	default:
		set := make(map[string]struct{})

		pp, err := u.ProviderStore().All(query.OrderAsc("name"))

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		for _, p := range pp {
			set[p.Name] = struct{}{}
		}

		for name, p := range h.Providers {
			if _, ok := set[name]; !ok {
				pp = append(pp, &model.Provider{
					Name:    name,
					AuthURL: p.AuthURL(),
				})
			}
		}

		sp.Providers = pp

		p = &sp

		break
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h User) Email(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	f := &form.Email{
		User:  u,
		Users: h.Users,
	}

	if err := h.ValidateForm(f, w, r); err != nil {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	u.Email = f.Email

	if err := h.Users.Update(u); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to update account: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Email has been updated"))

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h User) Password(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	f := &form.Password{
		User:  u,
		Users: h.Users,
	}

	if err := h.ValidateForm(f, w, r); err != nil {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	password, err := bcrypt.GenerateFromPassword([]byte(f.NewPassword), bcrypt.DefaultCost)

	if err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to update account: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	u.Password = password

	if err := h.Users.Update(u); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to update account: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Password has been updated"))

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
