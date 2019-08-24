package ui

import (
	"net/http"
	"path/filepath"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/model/query"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/user"

	"github.com/gorilla/csrf"
)

type User struct {
	web.Handler

	Invites model.InviteStore
}

func (h User) Settings(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	base := filepath.Base(r.URL.Path)

	var p template.Dashboard

	bp := template.BasePage{
		URI:  r.URL.Path,
		User: u,
	}

	sp := user.SettingsPage{
		BasePage: bp,
	}

	switch base {
	case "invites":
		ii, err := h.Invites.Index(query.WhereEq("invitee_id", u.ID))

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
		p = &sp

		break
	}

	d := template.NewDashboard(p, r.URL.Path, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}
