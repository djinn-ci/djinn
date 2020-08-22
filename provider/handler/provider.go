package handler

import (
	"crypto/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type Provider struct {
	web.Handler

	Block    *crypto.Block
	Registry *provider.Registry
}

func (h Provider) Auth(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	name := mux.Vars(r)["provider"]

	cfg, _, err := h.Registry.Get(name)

	if err != nil {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	u, _ := user.FromContext(r.Context())

	q := r.URL.Query()

	if q.Get("state") != cfg.Secret {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	access, refresh, user1, err := cfg.Auth(r.Context(), q.Get("code"))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to authenticate to " + name), "alert")
		h.Redirect(w, r, "/settings")
		return
	}

	encAccess, err := h.Block.Encrypt([]byte(access))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to authenticate to " + name), "alert")
		h.Redirect(w, r, "/settings")
		return
	}

	encRefresh, err := h.Block.Encrypt([]byte(refresh))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to authenticate to " + name), "alert")
		h.Redirect(w, r, "/settings")
		return
	}

	if u.IsZero() {
		u, err = h.Users.Get(
			query.WhereQuery("id", "=", provider.Select(
				"user_id",
				query.Where("provider_user_id", "=", user1.ID),
				query.Where("name", "=", name),
			)),
			query.OrWhere("email", "=", user1.Email),
		)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to authenticate to " + name), "alert")
			h.Redirect(w, r, "/settings")
			return
		}

		if u.IsZero() {
			password := make([]byte, 16)

			if _, err := rand.Read(password); err != nil {
				h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
				sess.AddFlash(template.Danger("Failed to authenticate to " + name), "alert")
				h.Redirect(w, r, "/settings")
				return
			}

			username := user1.Username

			if username == "" {
				username = user1.Login
			}

			u, err = h.Users.Create(user1.Email, username, password)

			if err != nil {
				h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
				sess.AddFlash(template.Danger("Failed to authenticate to " + name), "alert")
				h.Redirect(w, r, "/settings")
				return
			}
		}

		encoded, err := h.SecureCookie.Encode("user", strconv.FormatInt(u.ID, 10))

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to authenticate to " + name), "alert")
			h.Redirect(w, r, "/settings")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "user",
			HttpOnly: true,
			MaxAge:   user.MaxAge,
			Expires:  time.Now().Add(time.Duration(user.MaxAge) * time.Second),
			Value:    encoded,
			Path:     "/",
		})
	}

	providers := provider.NewStore(h.DB, u)

	p, err := providers.Get(query.Where("name", "=", name))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to authenticate to " + name), "alert")
		h.Redirect(w, r, "/settings")
		return
	}

	if p.IsZero() {
		p, err = providers.Create(u.ID, name, encAccess, encRefresh, true)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to authenticate to " + name), "alert")
			h.Redirect(w, r, "/settings")
			return
		}
	} else {
		if err := providers.Update(p.ID, user1.ID, name, encAccess, encRefresh, true); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to authenticate to " + name), "alert")
			h.Redirect(w, r, "/settings")
			return
		}
	}
	sess.AddFlash(template.Success("Successfully connected to " + name), "alert")
	h.Redirect(w, r, "/settings")
}

func (h Provider) Revoke(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	name := mux.Vars(r)["provider"]

	if _, _, err := h.Registry.Get(name); err != nil {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	providers := provider.NewStore(h.DB, u)

	p, err := providers.Get(query.Where("name", "=", name))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to disconnect from provider"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if p.IsZero() {
		h.RedirectBack(w, r)
		return
	}

	if err := providers.Update(p.ID, 0, p.Name, nil, nil, false); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to disconnect from provider"), "alert")
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Success("Successfully disconnected from provider"), "alert")
	h.RedirectBack(w, r)
}
