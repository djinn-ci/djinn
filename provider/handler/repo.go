package handler

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/provider"
	providertemplate "github.com/andrewpillar/thrall/provider/template"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"

	"github.com/go-redis/redis"
)

type Repo struct {
	web.Handler

	Redis    *redis.Client
	Block    *crypto.Block
	Registry *provider.Registry
	Repos    *provider.RepoStore
}

var cacheKey = "repos-%s-%v-%v"

func (h Repo) cachePut(name string, id int64, rr []*provider.Repo, paginator database.Paginator) error {
	var err error

	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)

	rr1 := make([]*provider.Repo, 0, len(rr))

	for _, r := range rr {
		r1 := (*r)
		r1.User = nil
		r1.Provider = nil

		rr1 = append(rr1, &r1)
	}

	if err := enc.Encode(rr1); err != nil {
		return errors.Err(err)
	}

	if err := enc.Encode(paginator); err != nil {
		return errors.Err(err)
	}

	_, err = h.Redis.Set(fmt.Sprintf(cacheKey, name, id, paginator.Page), buf.String(), time.Hour).Result()
	return errors.Err(err)
}

func (h Repo) cacheGet(name string, id, page int64) ([]*provider.Repo, database.Paginator, error) {
	var err error

	rr := make([]*provider.Repo, 0)
	paginator := database.Paginator{}

	s, err := h.Redis.Get(fmt.Sprintf(cacheKey, name, id, page)).Result()

	if err != nil {
		if err == redis.Nil {
			return rr, paginator, nil
		}
		return rr, paginator, errors.Err(err)
	}

	r := strings.NewReader(s)
	dec := gob.NewDecoder(r)

	if err := dec.Decode(&rr); err != nil {
		return rr, paginator, errors.Err(err)
	}

	err = dec.Decode(&paginator)
	return rr, paginator, errors.Err(err)
}

func (h Repo) loadRepos(p *provider.Provider, page int64) ([]*provider.Repo, database.Paginator, error) {
	if !p.Connected {
		return []*provider.Repo{}, database.Paginator{}, nil
	}

	rr, paginator, err := p.Repos(h.Block, h.Registry, page)
	return rr, paginator, errors.Err(err)
}

func (h Repo) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	opt := query.OrderAsc("name")

	if name := r.URL.Query().Get("provider"); name != "" {
		opt = query.Where("name", "=", name)
	}

	providers := provider.NewStore(h.DB, u)

	p, err := providers.Get(opt)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	rr, paginator, err := h.cacheGet(p.Name, u.ID, page)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if len(rr) == 0 {
		rr, paginator, err = h.loadRepos(p, page)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if err := h.cachePut(p.Name, u.ID, rr, paginator); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	enabled, err := provider.NewRepoStore(h.DB, u, p).All(query.Where("enabled", "=", true))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	pp, err := providers.All(query.OrderAsc("name"))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	providersLookup := make(map[int64]*provider.Provider)

	for _, p := range pp {
		providersLookup[p.ID] = p
	}

	enabledLookup := make(map[int64]int64)

	for _, r := range enabled {
		enabledLookup[r.RepoID] = r.ID
	}

	for _, r := range rr {
		if id, ok := enabledLookup[r.RepoID]; ok {
			r.ID = id
			r.Enabled = true
		}
		r.Provider = providersLookup[r.ProviderID]
	}

	csrfField := string(csrf.TemplateField(r))

	pg := &providertemplate.RepoIndex{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrfField,
		Paginator: paginator,
		Repos:     rr,
		Provider:  p,
		Providers: pp,
	}
	d := template.NewDashboard(pg, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Repo) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	p, err := provider.NewStore(h.DB, u).Get(query.Where("name", "=", r.URL.Query().Get("provider")))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to refresh repository cache"), "alert")
		h.RedirectBack(w, r)
		return
	}

	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	rr, paginator, err := h.loadRepos(p, page)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to refresh repository cache"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := h.cachePut(p.Name, u.ID, rr, paginator); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to refresh repository cache"), "alert")
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Success("Successfully reloaded repository cache"), "alert")
	h.RedirectBack(w, r)
}

func (h Repo) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	f := &provider.RepoForm{}

	if err := form.Unmarshal(f, r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to enable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}

	p, err := provider.NewStore(h.DB, u).Get(query.Where("name", "=", f.Provider))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to enable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}

	repos := provider.NewRepoStore(h.DB, u, p)

	repo, err := repos.Get(query.Where("id", "=", f.RepoID))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to enable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if repo.IsZero() {
		repo.RepoID = f.RepoID
		repo.Name = f.Name
	}

	if err := p.ToggleRepo(h.Block, h.Registry, repo); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to enable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if _, err := repos.Create(f.RepoID, f.Name, f.Href, repo.HookID.Int64); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to enable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Success("Repository hooks enabled"), "alert")
	h.RedirectBack(w, r)
}

func (h Repo) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	repo, ok := provider.RepoFromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get repo from request context")
	}

	if !repo.Enabled {
		sess.AddFlash(template.Success("Repository hooks disabled"), "alert")
		h.RedirectBack(w, r)
		return
	}

	p, err := provider.NewStore(h.DB).Get(query.Where("id", "=", repo.ProviderID))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to disable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := p.ToggleRepo(h.Block, h.Registry, repo); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to disable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}

	repo.Enabled = false

	if err := h.Repos.Update(repo); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to disable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Success("Repository hooks disabled"), "alert")
	h.RedirectBack(w, r)
}
