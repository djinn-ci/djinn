package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/repo"
	repotemplate "github.com/andrewpillar/thrall/repo/template"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"

	"github.com/go-redis/redis"
)

type Repo struct {
	web.Handler

	Redis     *redis.Client
	Repos     *repo.Store
	Providers map[string]oauth2.Provider
}

type repos struct {
	Paginator model.Paginator
	Items     []*repo.Repo
}

var cacheKey = "repos-%s-%v-%v"

func (h Repo) cachePut(name string, id, page int64, repos repos) error {
	buf := &bytes.Buffer{}

	json.NewEncoder(buf).Encode(repos)

	_, err := h.Redis.Set(fmt.Sprintf(cacheKey, name, id, page), buf.String(), time.Hour).Result()
	return errors.Err(err)
}

func (h Repo) cacheGet(name string, id, page int64) (repos, error) {
	repos := repos{}

	s, err := h.Redis.Get(fmt.Sprintf(cacheKey, name, id, page)).Result()

	if err != nil {
		if err == redis.Nil {
			return repos, nil
		}
		return repos, errors.Err(err)
	}

	err = json.NewDecoder(strings.NewReader(s)).Decode(&repos)
	return repos, errors.Err(err)
}

func (h Repo) loadRepos(p *provider.Provider, page int64) (repos, error) {
	repos := repos{
		Paginator: model.Paginator{
			Page: page,
		},
		Items:     make([]*repo.Repo, 0),
	}

	if !p.Connected {
		return repos, nil
	}

	prv, ok := h.Providers[p.Name]

	if !ok {
		return repos, nil
	}

	tok, _ := crypto.Decrypt(p.AccessToken)

	tmp, err := prv.Repos(tok, page)

	if err != nil {
		return repos, errors.Err(err)
	}

	repos.Paginator.Next = tmp.Next
	repos.Paginator.Prev = tmp.Prev
	repos.Paginator.Pages = []int64{tmp.Next, tmp.Prev}

	for _, r := range tmp.Items {
		repos.Items = append(repos.Items, &repo.Repo{
			UserID:     p.UserID,
			ProviderID: p.ID,
			RepoID:     r.ID,
			Name:       r.Name,
			Href:       r.Href,
			Provider:   p,
		})
	}
	return repos, nil
}

func (h Repo) Model(r *http.Request) *repo.Repo {
	val := r.Context().Value("repo")
	rp, _ := val.(*repo.Repo)
	return rp
}

func (h Repo) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	opt := query.OrderAsc("name")

	if name := r.URL.Query().Get("provider"); name != "" {
		opt = query.Where("name", "=", name)
	}

	providers := provider.NewStore(h.DB, u)

	prv, err := providers.Get(opt)

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	repos, err := h.cacheGet(prv.Name, u.ID, page)

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if len(repos.Items) == 0 {
		repos, err = h.loadRepos(prv, page)

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if err := h.cachePut(prv.Name, u.ID, page, repos); err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	enabled, err := repo.NewStore(h.DB, u, prv).All(query.Where("enabled", "=", true))

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	m := make(map[int64]int64)

	for _, repo := range enabled {
		m[repo.RepoID] = repo.ID
	}

	for _, r := range repos.Items {
		if id, ok := m[r.RepoID]; ok {
			r.ID = id
			r.Enabled = true
		}
	}

	pp, err := providers.All(query.OrderAsc("name"))

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrfField := string(csrf.TemplateField(r))

	p := &repotemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrfField,
		Paginator: repos.Paginator,
		Repos:     repos.Items,
		Provider:  prv,
		Providers: pp,
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Repo) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u := h.User(r)

	p, err := provider.NewStore(h.DB, u).Get(query.Where("name", "=", r.URL.Query().Get("provider")))

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to refresh repository cache"), "alert")
		h.RedirectBack(w, r)
		return
	}

	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	repos, err := h.loadRepos(p, page)

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to refresh repository cache"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := h.cachePut(p.Name, u.ID, page, repos); err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to refresh repository cache"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Successfully reloaded repository cache"), "alert")
	h.RedirectBack(w, r)
}

func (h Repo) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u := h.User(r)
	f := &repo.Form{}

	if err := form.Unmarshal(f, r); err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to enable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}

	p, err := provider.NewStore(h.DB, u).Get(query.Where("name", "=", f.Provider))

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to enable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if _, ok := h.Providers[f.Provider]; !ok || p.IsZero() {
		sess.AddFlash(template.Danger("Failed to enable repository hooks: unknow provider "+f.Provider), "alert")
		h.RedirectBack(w, r)
		return
	}

	var rp *repo.Repo

	enabled := func(id int64) (int64, bool, error) {
		rp, err = repo.NewStore(h.DB, u).Get(query.Where("id", "=", id))

		if err != nil {
			return 0, false, errors.Err(err)
		}
		return rp.HookID, rp.Enabled, nil
	}

	tok, _ := crypto.Decrypt(p.AccessToken)

	hookId, err := h.Providers[f.Provider].ToggleRepo(tok, f.RepoID, enabled)

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to enable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}

	rp.UserID = u.ID
	rp.ProviderID = p.ID
	rp.HookID = hookId
	rp.RepoID = f.RepoID
	rp.Enabled = hookId != 0

	fn := h.Repos.Update

	if rp.ID == 0 {
		fn = h.Repos.Create
	}

	if err := fn(rp); err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to enable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Repository hooks enabled"), "alert")
	h.RedirectBack(w, r)
}

func (h Repo) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	rp := h.Model(r)

	if !rp.Enabled {
		sess.AddFlash(template.Success("Repository hooks disabled"), "alert")
		h.RedirectBack(w, r)
		return
	}

	p, err := provider.NewStore(h.DB).Get(query.Where("id", "=", rp.ProviderID))

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to disable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}

	enabled := func(id int64) (int64, bool, error) {
		return rp.HookID, rp.Enabled, nil
	}

	tok, _ := crypto.Decrypt(p.AccessToken)

	if _, err := h.Providers[p.Name].ToggleRepo(tok, rp.RepoID, enabled); err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to disable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}

	rp.Enabled = false

	if err := h.Repos.Update(rp); err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to disable repository hooks"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Repository hooks disabled"), "alert")
	h.RedirectBack(w, r)
}
