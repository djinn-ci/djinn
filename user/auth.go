package user

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/mail"
	"djinn-ci.com/oauth2"

	"github.com/andrewpillar/query"

	"github.com/gorilla/securecookie"
)

const maxAge = 5 * 365 * 86400

func Cookie(u *auth.User, cookie *securecookie.SecureCookie) (*http.Cookie, error) {
	enc, err := cookie.Encode("user", strconv.FormatInt(u.ID, 10))

	if err != nil {
		return nil, errors.Err(err)
	}
	return &http.Cookie{
		Name:     "user",
		HttpOnly: true,
		MaxAge:   maxAge,
		Expires:  time.Now().Add(time.Duration(maxAge) * time.Second),
		Value:    enc,
		Path:     "/",
	}, nil
}

const verifyMail = `To secure your account please verify your email. Click the link below to
verify your account's email address,

    %s/settings/verify?token=%s`

func VerifyMail(from, addr string, u *auth.User) *mail.Mail {
	tok := u.RawData["account_token"].(string)

	return &mail.Mail{
		From:    from,
		To:      []string{u.Email},
		Subject: "Djinn CI - Verify email",
		Body:    fmt.Sprintf(verifyMail, addr, tok),
	}
}

func TokenAuth(db *database.Pool) auth.Authenticator {
	const tokenPrefix = "Bearer "

	users := NewStore(db)
	tokens := oauth2.NewTokenStore(db)

	return auth.AuthenticatorFunc(func(r *http.Request) (*auth.User, error) {
		tok := r.Header.Get("Authorization")

		if tok == "" {
			return nil, auth.ErrAuth
		}

		if !strings.HasPrefix(tok, tokenPrefix) {
			return nil, auth.ErrAuth
		}

		ctx := r.Context()

		tok = tok[len(tokenPrefix):]

		t, ok, err := tokens.Get(ctx, query.Where("token", "=", query.Arg(tok)))

		if err != nil {
			return nil, errors.Err(err)
		}

		if !ok {
			return nil, auth.ErrAuth
		}

		u, ok, err := users.Get(ctx, WhereID(t.UserID))

		if err != nil {
			return nil, errors.Err(err)
		}

		if !ok {
			return nil, auth.ErrAuth
		}

		for perm := range t.Permissions() {
			u.Grant(perm)
		}
		return u, nil
	})
}

func CookieAuth(db *database.Pool, cookie *securecookie.SecureCookie) auth.Authenticator {
	users := NewStore(db)

	perms := make(map[string]struct{})

	for _, res := range oauth2.Resources {
		perms[res.String()+":read"] = struct{}{}
		perms[res.String()+":write"] = struct{}{}
		perms[res.String()+":delete"] = struct{}{}
	}

	return auth.AuthenticatorFunc(func(r *http.Request) (*auth.User, error) {
		c, err := r.Cookie("user")

		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				return nil, auth.ErrAuth
			}
			return nil, errors.Err(err)
		}

		var s string

		if err := cookie.Decode("user", c.Value, &s); err != nil {
			if strings.Contains(err.Error(), "expired timestamp") {
				return nil, auth.ErrAuth
			}
			return nil, errors.Err(err)
		}

		id, _ := strconv.ParseInt(s, 10, 64)

		u, ok, err := users.Get(r.Context(), WhereID(id))

		if err != nil {
			return nil, errors.Err(err)
		}

		if !ok {
			return nil, auth.ErrAuth
		}

		for perm := range perms {
			u.Grant(perm)
		}
		return u, nil
	})
}

func FormAuth(db *database.Pool) auth.Authenticator {
	users := Store{
		Store: NewStore(db),
	}

	perms := make(map[string]struct{})

	for _, res := range oauth2.Resources {
		perms[res.String()+":read"] = struct{}{}
		perms[res.String()+":write"] = struct{}{}
		perms[res.String()+":delete"] = struct{}{}
	}

	return auth.AuthenticatorFunc(func(r *http.Request) (*auth.User, error) {
		if r.Method != "POST" {
			return nil, auth.ErrAuth
		}

		if r.URL.Path != "/login" && r.URL.Path != "/sudo" {
			return nil, auth.ErrAuth
		}

		if err := r.ParseForm(); err != nil {
			return nil, errors.Err(err)
		}

		u, err := users.Auth(r.Context(), r.PostForm.Get("handle"), r.PostForm.Get("password"))

		if err != nil {
			if !errors.Is(err, auth.ErrAuth) {
				return nil, errors.Err(err)
			}
			return nil, err
		}

		for perm := range perms {
			u.Grant(perm)
		}
		return u, nil
	})
}
