package http

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"sync"
	"time"

	"djinn-ci.com/alert"
	"djinn-ci.com/errors"
	"djinn-ci.com/provider"
	providertemplate "djinn-ci.com/provider/template"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type UI struct {
	*Handler

	User *userhttp.Handler
}

func (h UI) getUserOrCreate(name string, providerUserId int64, email, username string) (*user.User, error) {
	u, ok, err := h.Users.Get(query.Where("id", "=", provider.Select(
		"user_id",
		query.Where("provider_user_id", "=", query.Arg(providerUserId)),
		query.Where("name", "=", query.Arg(name)),
		query.Where("main_account", "=", query.Arg(true)),
	)))

	if err != nil {
		return nil, errors.Err(err)
	}

	if ok {
		return u, nil
	}

	_, ok, err = h.Users.Get(query.Where("email", "=", query.Arg(email)))

	if err != nil {
		return nil, errors.Err(err)
	}

	if ok {
		return nil, user.ErrExists
	}

	password := make([]byte, 16)

	if _, err := rand.Read(password); err != nil {
		return nil, errors.Err(err)
	}

	u, tok, err := h.Users.Create(user.Params{
		Email:    email,
		Username: username,
		Password: hex.EncodeToString(password),
	})

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := h.Users.Verify(tok); err != nil {
		return nil, errors.Err(err)
	}
	return u, nil
}

func (h UI) Auth(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u, ok, err := h.User.UserFromRequest(r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	name := mux.Vars(r)["provider"]

	cli, err := h.Server.Providers.Get(name)

	if err != nil {
		h.NotFound(w, r)
		return
	}

	back := "/settings"

	if !ok {
		back = "/login"
	}

	access, refresh, oauthuser, err := cli.Auth(r.Context(), r.URL.Query())

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to authenticate to  "+name)
		h.Redirect(w, r, back)
		return
	}

	// No user, so create one and set the cookie.
	if !ok {
		username := oauthuser.Username

		if username == "" {
			username = oauthuser.Login
		}

		u, err = h.getUserOrCreate(name, oauthuser.ID, oauthuser.Email, username)

		if err != nil {
			if errors.Is(errors.Cause(err), user.ErrExists) {
				msg := "User already exists with username " + username

				if oauthuser.Email != "" {
					msg += " and email " + oauthuser.Email
				}

				alert.Flash(sess, alert.Danger, msg)
				h.RedirectBack(w, r)
				return
			}

			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to authenticate to  "+name)
			h.Redirect(w, r, back)
			return
		}

		encoded, err := h.SecureCookie.Encode("user", strconv.FormatInt(u.ID, 10))

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to authenticate to  "+name)
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

	p, ok, err := h.Providers.Get(
		query.Where("user_id", "=", query.Arg(u.ID)),
		query.Where("name", "=", query.Arg(name)),
		query.Where("main_account", "=", query.Arg(true)),
	)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to authenticate to  "+name)
		h.Redirect(w, r, back)
		return
	}

	params := provider.Params{
		UserID:         u.ID,
		ProviderUserID: oauthuser.ID,
		Name:           name,
		AccessToken:    access,
		RefreshToken:   refresh,
		Connected:      true,
		MainAccount:    true,
	}

	if !ok {
		if _, err := h.Providers.Create(params); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to authenticate to  "+name)
			h.Redirect(w, r, back)
			return
		}
	} else {
		if err := h.Providers.Update(p.ID, params); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to authenticate to  "+name)
			h.Redirect(w, r, back)
			return
		}
	}

	groups, err := h.Providers.All(
		query.Where("user_id", "=", query.Arg(u.ID)),
		query.Where("name", "=", query.Arg(name)),
		query.Where("main_account", "=", query.Arg(false)),
	)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to authenticate to  "+name)
		h.Redirect(w, r, back)
		return
	}

	set := make(map[int64]struct{})

	for _, grp := range groups {
		set[grp.ProviderUserID.Int64] = struct{}{}
	}

	groupIds, err := cli.Groups(access)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to authenticate to  "+name)
		h.Redirect(w, r, back)
		return
	}

	for _, id := range groupIds {
		// Ignore any groups we already have on our side as a provider.
		if _, ok := set[id]; ok {
			continue
		}

		_, err := h.Providers.Create(provider.Params{
			UserID:         u.ID,
			ProviderUserID: id,
			Name:           name,
			AccessToken:    access,
			RefreshToken:   refresh,
			Connected:      true,
			MainAccount:    false,
		})

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to authenticate to  "+name)
			h.Redirect(w, r, back)
			return
		}
	}
	alert.Flash(sess, alert.Success, "Successfully connected to "+name)
	h.Redirect(w, r, "/")
}

func (h UI) Revoke(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	name := mux.Vars(r)["provider"]

	if _, err := h.Server.Providers.Get(name); err != nil {
		h.NotFound(w, r)
		return
	}

	p, ok, err := h.Providers.Get(
		query.Where("user_id", "=", query.Arg(u.ID)),
		query.Where("name", "=", query.Arg(name)),
	)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to disconnect from provider")
		h.RedirectBack(w, r)
		return
	}

	if !ok {
		h.RedirectBack(w, r)
		return
	}

	rr, err := h.Repos.All(
		query.Where("provider_id", "=", query.Arg(p.ID)),
		query.Where("enabled", "=", query.Arg(true)),
	)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to disconnect from provider")
		h.RedirectBack(w, r)
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(rr))

	ch := make(chan error)

	for _, r := range rr {
		go func(r *provider.Repo) {
			defer wg.Done()

			if err := p.ToggleRepo(r); err != nil {
				ch <- err
			}
		}(r)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	errs := make([]string, 0, len(rr))

	for err := range ch {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
		alert.Flash(sess, alert.Danger, "Failed to disconnect from provider")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Providers.Cache.Purge(p); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to disconnect from provider")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Providers.Delete(r.Context(), p.Name, u.ID); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to disconnect from provider")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Successfully disconnected from provider")
	h.RedirectBack(w, r)
}

func (h UI) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	q := r.URL.Query()

	page, err := strconv.ParseInt(q.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	name := q.Get("provider")

	if name == "" {
		if h.Server.Providers.Len() == 0 {
			pg := &providertemplate.RepoIndex{
				BasePage: template.BasePage{
					URL:  r.URL,
					User: u,
				},
			}
			d := template.NewDashboard(pg, r.URL, u, alert.First(sess), csrf.TemplateField(r))
			save(r, w)
			webutil.HTML(w, template.Render(d), http.StatusOK)
			return
		}
	}

	p, rr, paginator, err := h.Providers.LoadRepos(u.ID, name, page)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	names := h.Server.Providers.Names()

	if p == nil {
		if name == "" {
			// No provider for the main account, so stub in the first one we
			// have from the configuration.
			name = names[0]
		}

		p = &provider.Provider{
			Name: name,
		}
	}

	pp := make([]*provider.Provider, 0, len(names))

	for _, name := range names {
		if name == p.Name {
			pp = append(pp, p)
			continue
		}
		pp = append(pp, &provider.Provider{Name: name})
	}

	csrf := csrf.TemplateField(r)
	pg := &providertemplate.RepoIndex{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrf,
		Paginator: paginator,
		Repos:     rr,
		Provider:  p,
		Providers: pp,
	}
	d := template.NewDashboard(pg, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Update(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	p, ok, err := h.Providers.Get(
		query.Where("user_id", "=", query.Arg(u.ID)),
		query.Where("name", "=", query.Arg(r.URL.Query().Get("provider"))),
		query.Where("main_account", "=", query.Arg(true)),
	)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to refresh repository cache")
		h.RedirectBack(w, r)
		return
	}

	if !ok {
		alert.Flash(sess, alert.Danger, "Failed to refresh repository cache: no such provider")
		h.RedirectBack(w, r)
		return
	}

	rr, paginator, err := p.Repos(page)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to refresh repository cache")
		h.RedirectBack(w, r)
		return
	}

	if len(rr) > 0 {
		if err := h.Providers.Cache.Put(p, rr, paginator); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to refresh repository cache")
			h.RedirectBack(w, r)
			return
		}
	}

	alert.Flash(sess, alert.Success, "Successfully reloaded repository cache")
	h.RedirectBack(w, r)
}

func (h UI) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	var f RepoForm

	if err := webutil.UnmarshalForm(&f, r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to enable repository hooks")
		h.RedirectBack(w, r)
		return
	}

	p, ok, err := h.Providers.Get(
		query.Where("id", "=", query.Arg(f.ProviderID)),
		query.Where("user_id", "=", query.Arg(u.ID)),
	)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to enable repository hooks")
		h.RedirectBack(w, r)
		return
	}

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to enable repository hooks: no such provider")
		h.RedirectBack(w, r)
		return
	}

	repo, ok, err := h.Repos.Get(
		query.Where("user_id", "=", query.Arg(u.ID)),
		query.Where("provider_id", "=", query.Arg(p.ID)),
		query.Where("repo_id", "=", query.Arg(f.RepoID)),
	)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to enable repository hooks")
		h.RedirectBack(w, r)
		return
	}

	if !ok {
		repo = &provider.Repo{
			UserID:       u.ID,
			ProviderID:   f.ProviderID,
			RepoID:       f.RepoID,
			ProviderName: p.Name,
			Name:         f.Name,
			Href:         f.Href,
		}
	}

	if err := p.ToggleRepo(repo); err != nil {
		if errors.Is(err, provider.ErrLocalhost) {
			alert.Flash(sess, alert.Danger, "Failed to enable repository hooks: "+errors.Unwrap(err).Error())
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to enable repository hooks")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Repos.Touch(repo); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to enable repository hooks")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Repository hooks enabled")
	h.RedirectBack(w, r)
}

func (h UI) Destroy(u *user.User, repo *provider.Repo, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if !repo.Enabled {
		alert.Flash(sess, alert.Success, "Repository hooks disabled")
		h.RedirectBack(w, r)
		return
	}

	p, ok, err := h.Providers.Get(query.Where("id", "=", query.Arg(repo.ProviderID)))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to disable repository hooks")
		h.RedirectBack(w, r)
		return
	}

	if !ok {
		alert.Flash(sess, alert.Danger, "Failed to disable repository hooks: no such provider")
		h.RedirectBack(w, r)
		return
	}

	if err := p.ToggleRepo(repo); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to disable repository hooks")
		h.RedirectBack(w, r)
		return
	}

	if err := h.Repos.Touch(repo); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to disable repository hooks")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Repository hooks disabled")
	h.RedirectBack(w, r)
}

func RegisterUI(srv *server.Server) {
	user := userhttp.NewHandler(srv)

	ui := UI{
		Handler: NewHandler(srv),
		User:    user,
	}

	auth := srv.Router.PathPrefix("/oauth").Subrouter()
	auth.HandleFunc("/{provider}", ui.Auth).Methods("GET")
	auth.HandleFunc("/{provider}", user.WithUser(ui.Revoke)).Methods("DELETE")
	auth.Use(srv.CSRF)

	sr := srv.Router.PathPrefix("/repos").Subrouter()
	sr.HandleFunc("", user.WithUser(ui.Index)).Methods("GET")
	sr.HandleFunc("/reload", user.WithUser(ui.Update)).Methods("PATCH")
	sr.HandleFunc("/enable", user.WithUser(ui.Store)).Methods("POST")
	sr.HandleFunc("/disable/{repo:[0-9]+}", user.WithUser(ui.WithRepo(ui.Destroy))).Methods("DELETE")
	sr.Use(srv.CSRF)
}
