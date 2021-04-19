package handler

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/errors"
	"djinn-ci.com/provider"
	"djinn-ci.com/template"
	"djinn-ci.com/user"
	"djinn-ci.com/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/go-redis/redis"
)

// Provider is the handler that handles the authentication flow against a
// provider that a user either wants to login with, or connect to.
type Provider struct {
	web.Handler

	redis     *redis.Client
	block     *crypto.Block
	providers *provider.Registry
}

func New(h web.Handler, redis *redis.Client, block *crypto.Block, providers *provider.Registry) Provider {
	return Provider{
		Handler:   h,
		redis:     redis,
		block:     block,
		providers: providers,
	}
}

func (h Provider) cachePurge(name string, id int64) error {
	page := 1

	// Delete every key we can find, this breaks when the result returned from
	// DEL is 0, or if there is an error.
	for {
		key := fmt.Sprintf(cacheKey, name, id, page)

		n, err := h.redis.Del(key).Result()

		if err != nil {
			return errors.Err(err)
		}

		if n == 0 {
			break
		}
		page++
	}
	return nil
}

func (h Provider) disableHooks(p *provider.Provider) error {
	rr, err := provider.NewRepoStore(h.DB, p).All(query.Where("enabled", "=", query.Arg(true)))

	if err != nil {
		return errors.Err(err)
	}

	cherrs := make(chan error)

	wg := &sync.WaitGroup{}
	wg.Add(len(rr))

	for _, r := range rr {
		go func(r *provider.Repo) {
			defer wg.Done()

			if err := p.ToggleRepo(h.block, h.providers, r); err != nil {
				cherrs <- err
			}
		}(r)
	}

	go func() {
		wg.Wait()
		close(cherrs)
	}()

	errs := make([]error, 0)

	for err := range cherrs {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Slice(errs)
	}
	return nil
}

func (h Provider) lookupUser(name string, userId int64, email, username string) (*user.User, error) {
	// Do we have a pre-existing user that is connected.
	u, err := h.Users.Get(
		query.Where("id", "=", provider.Select(
			"user_id",
			query.Where("provider_user_id", "=", query.Arg(userId)),
			query.Where("name", "=", query.Arg(name)),
			query.Where("main_account", "=", query.Arg(true)),
		)),
		query.Where("email", "=", query.Arg(email)),
	)

	if err != nil {
		return nil, errors.Err(err)
	}

	if !u.IsZero() {
		return u, nil
	}

	// No pre-existing user, try and create a user on the fly if the email
	// isn't taken.
	u, err = h.Users.Get(
		query.Where("id", "=", provider.Select(
			"user_id",
			query.Where("name", "=", query.Arg(name)),
			query.Where("main_account", "=", query.Arg(true)),
		)),
		query.Where("email", "=", query.Arg(email)),
	)

	if err != nil {
		return nil, errors.Err(err)
	}

	if !u.IsZero() {
		return nil, user.ErrExists
	}

	password := make([]byte, 16)

	if _, err := rand.Read(password); err != nil {
		return nil, errors.Err(err)
	}

	u, tok, err := h.Users.Create(email, username, password)

	if err != nil {
		return nil, errors.Err(err)
	}
	return u, errors.Err(h.Users.Verify(tok))
}

// Auth will authenticate the user via the provider in the request. If a user
// is logging in for the first time via a provider, then we create an account
// for them and link it to the provider. Otherwise we simply connect the
// provider to the current user's account.
func (h Provider) Auth(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	name := mux.Vars(r)["provider"]

	cli, err := h.providers.Get(name)

	if err != nil {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	back := "/settings"

	u, ok, err := h.UserFromCookie(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if !ok {
		back = "/login"
	}

	access, refresh, user1, err := cli.Auth(r.Context(), r.URL.Query())

	if err != nil {
		if err == provider.ErrStateMismatch {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}
		web.HTMLError(w, r.URL.Query().Get("error_description"), http.StatusBadRequest)
		return
	}

	encAccess, err := h.block.Encrypt([]byte(access))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to authenticate to " + name,
		}, "alert")
		h.Redirect(w, r, back)
		return
	}

	encRefresh, err := h.block.Encrypt([]byte(refresh))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to authenticate to " + name,
		}, "alert")
		h.Redirect(w, r, back)
		return
	}

	if u.IsZero() {
		username := user1.Username

		if username == "" {
			username = user1.Login
		}

		u, err = h.lookupUser(name, user1.ID, user1.Email, username)

		if err != nil {
			cause := errors.Cause(err)

			if cause == user.ErrExists {
				sess.AddFlash(template.Alert{
					Level:   template.Danger,
					Close:   true,
					Message: "User already exists with email " + user1.Email,
				}, "alert")
				h.RedirectBack(w, r)
				return
			}

			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to authenticate to " + name,
			}, "alert")
			h.Redirect(w, r, back)
			return
		}

		encoded, err := h.SecureCookie.Encode("user", strconv.FormatInt(u.ID, 10))

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to authenticate to " + name,
			}, "alert")
			h.Redirect(w, r, back)
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

	p, err := providers.Get(
		query.Where("name", "=", query.Arg(name)),
		query.Where("main_account", "=", query.Arg(true)),
	)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to authenticate to " + name,
		}, "alert")
		h.Redirect(w, r, back)
		return
	}

	if p.IsZero() {
		p, err = providers.Create(user1.ID, name, encAccess, encRefresh, true, true)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to authenticate to " + name,
			}, "alert")
			h.Redirect(w, r, back)
			return
		}
	} else {
		if err := providers.Update(p.ID, user1.ID, name, encAccess, encRefresh, true, true); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to authenticate to " + name,
			}, "alert")
			h.Redirect(w, r, back)
			return
		}
	}

	groups, err := providers.All(
		query.Where("name", "=", query.Arg(name)),
		query.Where("main_account", "=", query.Arg(false)),
	)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to authenticate to " + name,
		}, "alert")
		h.Redirect(w, r, back)
		return
	}

	m := make(map[int64]struct{})

	for _, g := range groups {
		m[g.ProviderUserID.Int64] = struct{}{}
	}

	// Workaround for pushing from org repos.
	groupIds, err := cli.Groups(access)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to authenticate to " + name,
		}, "alert")
		h.Redirect(w, r, back)
		return
	}

	for _, id := range groupIds {
		if _, ok := m[id]; ok {
			continue
		}

		if _, err = providers.Create(id, name, encAccess, encRefresh, false, true); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to authenticate to " + name,
			}, "alert")
			h.Redirect(w, r, back)
			return
		}
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Successfully connected to " + name,
	}, "alert")
	h.Redirect(w, r, "/")
}

// Revoke will disconnect the user from the provider sent in the request. This
// will also disable all of the repository hooks for the given provider if any
// were set.
func (h Provider) Revoke(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	name := mux.Vars(r)["provider"]

	if _, err := h.providers.Get(name); err != nil {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	providers := provider.NewStore(h.DB, u)

	p, err := providers.Get(query.Where("name", "=", query.Arg(name)))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to disconnect from provider",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	if p.IsZero() {
		h.RedirectBack(w, r)
		return
	}

	if err := h.disableHooks(p); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to disconnect from provider",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := h.cachePurge(p.Name, p.UserID); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to disconnect from provider",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := providers.Update(p.ID, 0, p.Name, nil, nil, true, false); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to disconnect from provider",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	pp, err := providers.All(query.Where("main_account", "=", query.Arg(false)))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to disconnect from provider",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := providers.Delete(pp...); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to disconnect from provider",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Successfully disconnected from provider",
	}, "alert")
	h.RedirectBack(w, r)
}
