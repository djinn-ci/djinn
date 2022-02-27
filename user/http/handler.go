package http

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/errors"
	"djinn-ci.com/oauth2"
	"djinn-ci.com/provider"
	"djinn-ci.com/server"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

type HandlerFunc func(*user.User, http.ResponseWriter, *http.Request)

type Handler struct {
	*server.Server

	Users     user.Store
	Tokens    *oauth2.TokenStore
	Providers *provider.Store
}

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server: srv,
		Users:  user.Store{Pool: srv.DB},
		Tokens: &oauth2.TokenStore{Pool: srv.DB},
		Providers: &provider.Store{
			Pool:   srv.DB,
			AESGCM: srv.AESGCM,
		},
	}
}

var tokenPrefix = "Bearer "

func (h *Handler) UserFromRequest(r *http.Request) (*user.User, bool, error) {
	if _, _, ok := r.BasicAuth(); ok {
		goto cookie
	}

	if tok := r.Header.Get("Authorization"); tok != "" {
		if !strings.HasPrefix(tok, tokenPrefix) {
			return nil, false, nil
		}

		tok, ok, err := h.Tokens.Get(query.Where("token", "=", query.Arg(tok[len(tokenPrefix):])))

		if err != nil {
			return nil, false, errors.Err(err)
		}

		if !ok {
			return nil, false, nil
		}

		u, ok, err := h.Users.Get(user.WhereID(tok.UserID))

		if err != nil {
			return nil, false, errors.Err(err)
		}

		if ok {
			u.Permissions = tok.Permissions()
		}
		return u, ok, nil
	}

cookie:
	c, err := r.Cookie("user")

	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return nil, false, nil
		}
		return nil, false, errors.Err(err)
	}

	var s string

	if err := h.SecureCookie.Decode("user", c.Value, &s); err != nil {
		return nil, false, errors.Err(err)
	}

	id, err := strconv.ParseInt(s, 10, 64)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	u, ok, err := h.Users.Get(user.WhereID(id))

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if ok {
		for _, res := range oauth2.Resources {
			u.SetPermission(res.String() + ":read")
			u.SetPermission(res.String() + ":write")
			u.SetPermission(res.String() + ":delete")
		}
	}
	return u, ok, nil
}

func (h *Handler) WithUser(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok, err := h.UserFromRequest(r)

		if err != nil {
			msg := errors.Cause(err).Error()

			if !strings.Contains(msg, "expired timestamp") && !strings.Contains(msg, "invalid timestamp") {
				h.InternalServerError(w, r, errors.Err(err))
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     "user",
				HttpOnly: true,
				Path:     "/",
				Expires:  time.Unix(0, 0),
			})
		}

		if !ok {
			if server.IsJSON(r) {
				h.NotFound(w, r)
				return
			}

			uri := url.PathEscape(webutil.BaseAddress(r) + r.URL.String())

			h.Redirect(w, r, "/login?redirect_uri="+uri)
			return
		}

		if u.Email == "" && r.URL.Path != "/settings/email" {
			if server.IsJSON(r) {
				webutil.JSON(w, map[string]interface{}{
					"message": "No email set on account, go to your Settings page to set it.",
				}, http.StatusUnauthorized)
				return
			}

			uri := url.PathEscape(webutil.BaseAddress(r) + r.URL.String())
			h.Redirect(w, r, "/settings/email?redirect_uri="+uri)
			return
		}
		fn(u, w, r)
	}
}

func (h *Handler) WithOptionalUser(fn HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, ok, err := h.UserFromRequest(r)

		if err != nil {
			msg := errors.Cause(err).Error()

			if !strings.Contains(msg, "expired timestamp") && !strings.Contains(msg, "invalid timestamp") {
				h.InternalServerError(w, r, errors.Err(err))
				return
			}
		}

		if ok {
			if u.Email == "" {
				if server.IsJSON(r) {
					webutil.JSON(w, map[string]interface{}{
						"message": "No email set on account, go to your Settings page to set it.",
					}, http.StatusUnauthorized)
					return
				}

				uri := url.PathEscape(webutil.BaseAddress(r) + r.URL.String())
				h.Redirect(w, r, "/settings/email?redirect_uri="+uri)
				return
			}
		}

		if !ok {
			u = &user.User{
				Permissions: make(map[string]struct{}),
			}
		}
		fn(u, w, r)
	}
}

func (h *Handler) Guest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok, err := h.UserFromRequest(r)

		if err != nil {
			msg := errors.Cause(err).Error()

			if !strings.Contains(msg, "expired timestamp") && !strings.Contains(msg, "invalid timestamp") {
				h.InternalServerError(w, r, errors.Err(err))
				return
			}
		}

		if ok {
			if server.IsJSON(r) {
				h.NotFound(w, r)
				return
			}
			h.Redirect(w, r, "/")
			return
		}
		next.ServeHTTP(w, r)
	})
}
