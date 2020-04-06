package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
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
	Repos     repo.Store
	Providers map[string]oauth2.Provider
}

var cacheKey = "repos-%v"

func (h Repo) cachePut(id int64, rr []*repo.Repo) error {
	buf := &bytes.Buffer{}

	json.NewEncoder(buf).Encode(rr)

	_, err := h.Redis.Set(fmt.Sprintf(cacheKey, id), buf.String(), time.Hour).Result()
	return errors.Err(err)
}

func (h Repo) cacheGet(id int64) ([]*repo.Repo, error) {
	rr := make([]*repo.Repo, 0)

	s, err := h.Redis.Get(fmt.Sprintf(cacheKey, id)).Result()

	if err != nil {
		if err == redis.Nil {
			return rr, nil
		}
		return rr, errors.Err(err)
	}

	err = json.NewDecoder(strings.NewReader(s)).Decode(&rr)
	return rr, errors.Err(err)
}

func (h Repo) loadRepos(pp []*provider.Provider) ([]*repo.Repo, error) {
	rr := make([]*repo.Repo, 0)

	for _, p := range pp {
		if !p.Connected {
			continue
		}

		provider, ok := h.Providers[p.Name]

		if !ok {
			continue
		}

		tok, _ := crypto.Decrypt(p.AccessToken)

		tmp, err := provider.Repos(tok)

		if err != nil {
			return rr, errors.Err(err)
		}

		for _, r := range tmp {
			rr = append(rr, &repo.Repo{
				UserID:     p.UserID,
				ProviderID: p.ID,
				RepoID:     r.ID,
				Name:       r.Name,
				Href:       r.Href,
			})
		}
	}
	return rr, nil
}

func (h Repo) Model(r *http.Request) *repo.Repo {
	val := r.Context().Value("repo")
	rp, _ := val.(*repo.Repo)
	return rp
}

func (h Repo) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u := h.User(r)

	pp, err := provider.NewStore(h.DB, u).All()

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	providers := make(map[int64]*provider.Provider)

	for _, p := range pp {
		if p.Connected {
			u.Connected = true
		}
		providers[p.ID] = p
	}

	rr, err := h.cacheGet(u.ID)

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if len(rr) == 0 {
		rr, err = h.loadRepos(pp)

		if err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if err := h.cachePut(u.ID, rr); err != nil {
			log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	connected, err := repo.NewStore(h.DB, u).All()

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	m := make(map[int64]int64)

	for _, r := range connected {
		m[r.ProviderID+r.RepoID] = r.ID
	}

	for _, r := range rr {
		if id, ok := m[r.ProviderID+r.RepoID]; ok {
			r.ID = id
		}
		r.Provider = providers[r.ProviderID]
	}

	provider := r.URL.Query().Get("provider")

	if provider != "" {
		for i := len(rr) - 1; i > -1; i-- {
			if rr[i].Provider.Name != provider {
				rr = append(rr[:i], rr[i+1:]...)
			}
		}
	}

	csrfField := string(csrf.TemplateField(r))

	p := &repotemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:      csrfField,
		Repos:     rr,
		Provider:  provider,
		Providers: pp,
	}
	d := template.NewDashboard(p, r.URL, h.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Repo) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u := h.User(r)

	pp, err := provider.NewStore(h.DB).All()

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to refresh repository cache"), "alert")
		h.RedirectBack(w, r)
		return
	}

	rr, err := h.loadRepos(pp)

	if err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to refresh repository cache"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := h.cachePut(u.ID, rr); err != nil {
		log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to refresh repository cache"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Danger("Successfully reloaded repository cache"), "alert")
	h.RedirectBack(w, r)
}

func (h Repo) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u := h.User(r)
	f := &repo.Form{}

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

	rp.HookID = hookId

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
