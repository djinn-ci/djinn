package http

import (
	"net/http"
	"strconv"
	"sync"

	"djinn-ci.com/alert"
	"djinn-ci.com/auth"
	"djinn-ci.com/auth/oauth2"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/provider"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"

	"github.com/gorilla/mux"
)

type UI struct {
	*Handler
}

func (h UI) Connect(u *auth.User, w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to parse request"))
		return
	}

	mech := r.PostForm.Get("auth_mech")

	a, err := h.Auths.Get(mech)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to authenticate request"))
		return
	}

	cli, ok := a.(*oauth2.Authenticator)

	if !ok {
		h.Error(w, r, errors.Benign("Not a valid OAuth2 client"))
		return
	}

	url := cli.AuthURL()

	h.Log.Debug.Println(r.Method, r.URL, "authenticating via oauth2 provider", mech)
	h.Log.Debug.Println(r.Method, r.URL, "auth_url =", url)

	http.Redirect(w, r, url, http.StatusSeeOther)
	return
}

func (h UI) Auth(w http.ResponseWriter, r *http.Request) {
	provider := mux.Vars(r)["provider"]

	h.Log.Debug.Println(r.Method, r.URL, "authenticating against", provider)

	a, err := h.Auths.Get("oauth2." + provider)

	if err != nil {
		h.NotFound(w, r)
		return
	}

	// Wrap the OAuth2 authenticator to get the currently authenticated user,
	// if any, and attach them to the raw data of the user we get from the
	// provider.
	wrapped := auth.AuthenticatorFunc(func(r *http.Request) (*auth.User, error) {
		u, err := a.Auth(r)

		if err != nil {
			return nil, errors.Err(err)
		}

		cookieUser, err := user.CookieAuth(h.DB, h.SecureCookie).Auth(r)

		if err != nil {
			if !errors.Is(err, auth.ErrAuth) {
				return nil, errors.Err(err)
			}
			return u, nil
		}

		u.RawData[user.InternalProvider] = cookieUser
		return u, nil
	})

	sess, _ := h.Session(r)

	u, err := auth.Persist(wrapped, h.Providers).Auth(r)

	if err != nil {
		if errors.Is(err, database.ErrExists) {
			alert.Flash(sess, alert.Danger, "User already exists")
			h.Redirect(w, r, "/login")
			return
		}

		h.Error(w, r, errors.Wrap(err, "Failed to authenticated via "+provider))
		return
	}

	if _, ok := u.RawData["token"]; ok {
		h.Queues.Produce(r.Context(), "email", user.VerifyMail(h.SMTP.From, webutil.BaseAddress(r), u))
	}

	cookie, err := user.Cookie(u, h.SecureCookie)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to authenticate via "+provider))
		return
	}

	http.SetCookie(w, cookie)
	alert.Flash(sess, alert.Success, "Successfully connected to "+provider)
	h.Redirect(w, r, "/")
}

func (h UI) Revoke(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	name := mux.Vars(r)["provider"]

	if _, err := h.Server.Providers.Get("", name); err != nil {
		h.NotFound(w, r)
		return
	}

	ctx := r.Context()

	p, ok, err := h.Providers.Get(
		ctx,
		query.Where("user_id", "=", query.Arg(u.ID)),
		query.Where("name", "=", query.Arg(name)),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to disconnect from provider"))
		return
	}

	if !ok {
		h.RedirectBack(w, r)
		return
	}

	rr, err := h.Repos.All(
		ctx,
		query.Where("provider_id", "=", query.Arg(p.ID)),
		query.Where("enabled", "=", query.Arg(true)),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to disconnect from provider"))
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(rr))

	ch := make(chan error)

	for _, r := range rr {
		go func(r *provider.Repo) {
			defer wg.Done()

			if err := p.Client().ToggleWebhook(r); err != nil {
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
		h.Error(w, r, errors.Wrap(errors.Slice(errs), "Failed to disconnect from provider"))
		return
	}

	if err := h.Repos.Purge(p); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to disconnect from provider"))
		return
	}

	if err := h.Providers.Delete(ctx, p); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to disconnect from provider"))
		return
	}

	alert.Flash(sess, alert.Success, "Successfully disconnected from provider")
	h.RedirectBack(w, r)
}

func (h UI) Index(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	q := r.URL.Query()

	page, err := strconv.Atoi(q.Get("page"))

	if err != nil {
		page = 1
	}

	name := q.Get("provider")

	opts := []query.Option{
		query.Where("name", "=", query.Arg(name)),
	}

	if name == "" {
		opts = []query.Option{
			query.Where("connected", "=", query.Arg(true)),
			query.OrderAsc("name"),
		}
	}

	ctx := r.Context()

	p, ok, err := h.Providers.Get(ctx, opts...)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to find provider"))
		return
	}

	pp := make([]*provider.Provider, 0)

	for _, name := range h.Server.Providers.Names() {
		pp = append(pp, &provider.Provider{Name: name})
	}

	if !ok {
		p = &provider.Provider{Name: name}

		// No name given, so use first provider we have from what was
		// configured.
		if p.Name == "" {
			p.Name = pp[0].Name
		}

		tmpl := template.NewDashboard(u, sess, r)
		tmpl.Partial = &template.RepoIndex{
			Paginator: template.NewPaginator[*provider.Repo](tmpl.Page, &database.Paginator[*provider.Repo]{}),
			Provider:  p,
			Providers: pp,
		}
		h.Template(w, r, tmpl, http.StatusOK)
		return
	}

	repos, err := h.Repos.Load(ctx, p, page)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed load repos"))
		return
	}

	if repos == nil {
		repos = &database.Paginator[*provider.Repo]{}
	}

	tmpl := template.NewDashboard(u, sess, r)
	tmpl.Partial = &template.RepoIndex{
		Paginator: template.NewPaginator[*provider.Repo](tmpl.Page, repos),
		Repos:     repos.Items,
		Provider:  p,
		Providers: pp,
	}
	h.Template(w, r, tmpl, http.StatusOK)
}

func (h UI) Update(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	page, err := strconv.Atoi(r.URL.Query().Get("page"))

	if err != nil {
		page = 1
	}

	ctx := r.Context()

	p, ok, err := h.Providers.Get(
		ctx,
		query.Where("user_id", "=", query.Arg(u.ID)),
		query.Where("name", "=", query.Arg(r.URL.Query().Get("provider"))),
		query.Where("main_account", "=", query.Arg(true)),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to refresh repository cache"))
		return
	}

	if !ok {
		alert.Flash(sess, alert.Danger, "Failed to refresh repository cache: no such provider")
		h.RedirectBack(w, r)
		return
	}

	if _, err := h.Repos.Reload(ctx, p, page); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to refresh repository cache"))
		return
	}

	alert.Flash(sess, alert.Success, "Successfully reloaded repository cache")
	h.RedirectBack(w, r)
}

func (h UI) Store(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	var f RepoForm

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		h.FormError(w, r, &f, errors.Wrap(err, "Failed to enable repo webhook"))
		return
	}

	ctx := r.Context()

	p, ok, err := h.Providers.Get(
		ctx,
		query.Where("id", "=", query.Arg(f.ProviderID)),
		query.Where("user_id", "=", query.Arg(u.ID)),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to enable repo webhook"))
		return
	}

	if !ok {
		h.Error(w, r, errors.Wrap(err, "Failed to enable repo webhook"))
		return
	}

	repo, ok, err := h.Repos.Get(
		ctx,
		query.Where("user_id", "=", query.Arg(u.ID)),
		query.Where("provider_id", "=", query.Arg(p.ID)),
		query.Where("repo_id", "=", query.Arg(f.RepoID)),
	)

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to enable repo webhook"))
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

	if err := p.Client().ToggleWebhook(repo); err != nil {
		if errors.Is(err, provider.ErrLocalhost) {
			h.Error(w, r, errors.Benign("Failed to enable repo webhook: "+err.Error()))
			return
		}
		h.Error(w, r, errors.Wrap(err, "Failed to enable repo webhook"))
		return
	}

	if err := h.Repos.Touch(ctx, repo); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to enable repo webhook"))
		return
	}

	alert.Flash(sess, alert.Success, "Repository hooks enabled")
	h.RedirectBack(w, r)
}

func (h UI) Destroy(u *auth.User, repo *provider.Repo, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if !repo.Enabled {
		alert.Flash(sess, alert.Success, "Repo webhook disabled")
		h.RedirectBack(w, r)
		return
	}

	ctx := r.Context()

	p, ok, err := h.Providers.Get(ctx, query.Where("id", "=", query.Arg(repo.ProviderID)))

	if err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to disable repo webhook"))
		return
	}

	if !ok {
		h.Error(w, r, errors.Benign("Failed to disable repo webhook: no such provider"))
		return
	}

	if err := p.Client().ToggleWebhook(repo); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to disable repo webhook"))
		return
	}

	if err := h.Repos.Touch(ctx, repo); err != nil {
		h.Error(w, r, errors.Wrap(err, "Failed to disable repo webhook"))
		return
	}

	alert.Flash(sess, alert.Success, "Repo webhook disabled")
	h.RedirectBack(w, r)
}

func RegisterUI(a auth.Authenticator, srv *server.Server) {
	ui := UI{
		Handler: NewHandler(srv),
	}

	auth := srv.Router.PathPrefix("/oauth").Subrouter()
	auth.HandleFunc("", ui.Restrict(a, nil, ui.Connect)).Methods("POST")
	auth.HandleFunc("/{provider}", ui.Auth).Methods("GET")
	auth.HandleFunc("/{provider}", ui.Restrict(a, nil, ui.Revoke)).Methods("DELETE")
	auth.Use(srv.CSRF)

	index := ui.Restrict(a, nil, ui.Index)
	store := ui.Restrict(a, nil, ui.Store)
	update := ui.Restrict(a, nil, ui.Update)
	destroy := ui.Restrict(a, nil, ui.Repo(ui.Destroy))

	sr := srv.Router.PathPrefix("/repos").Subrouter()
	sr.HandleFunc("", index).Methods("GET")
	sr.HandleFunc("/reload", update).Methods("PATCH")
	sr.HandleFunc("/enable", store).Methods("POST")
	sr.HandleFunc("/disable/{repo:[0-9]+}", destroy).Methods("DELETE")
	sr.Use(srv.CSRF)
}
