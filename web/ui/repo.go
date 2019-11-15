package ui

import (
	"bytes"
	"context"
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
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/repo"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"

	"github.com/go-redis/redis"
)


type Repo struct {
	web.Handler

	Redis     *redis.Client
	Providers map[string]oauth2.Provider
}

func (h Repo) cacheRepos(id int64, rr []*model.Repo) error {
	buf := &bytes.Buffer{}

	enc := json.NewEncoder(buf)
	enc.Encode(rr)

	_, err := h.Redis.Set(fmt.Sprintf("repos-%v", id), buf.String(), time.Hour).Result()

	return errors.Err(err)
}

func (h Repo) getCached(id int64) ([]*model.Repo, error) {
	rr := make([]*model.Repo, 0)

	s, err := h.Redis.Get(fmt.Sprintf("repos-%v", id)).Result()

	if err != nil {
		if err == redis.Nil {
			return rr, nil
		}

		return rr, errors.Err(err)
	}

	dec := json.NewDecoder(strings.NewReader(s))
	dec.Decode(&rr)

	return rr, nil
}

func (h Repo) loadRepos(c context.Context, pp []*model.Provider) ([]*model.Repo, error) {
	rr := make([]*model.Repo, 0)

	for _, p := range pp {
		if !p.Connected {
			continue
		}

		provider, ok := h.Providers[p.Name]

		if !ok {
			continue
		}

		b, _ := crypto.Decrypt(p.AccessToken)

		tmp, err := provider.Repos(c, string(b))

		if err != nil {
			return rr, errors.Err(err)
		}

		for _, t := range tmp {
			t.ProviderID = p.ID
		}

		rr = append(rr, tmp...)
	}

	return rr, nil
}

func (h Repo) Index(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	if err := u.LoadProviders(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	providers := make(map[int64]*model.Provider)

	for _, p := range u.Providers {
		if p.Connected {
			u.Connected = true
		}

		providers[p.ID] = p
	}

	rr, err := h.getCached(u.ID)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if len(rr) == 0 {
		rr, err = h.loadRepos(r.Context(), u.Providers)

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if err := h.cacheRepos(u.ID, rr); err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	repos := u.RepoStore()

	userRepos, err := repos.All()

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	enabled := make(map[string]int64)

	for _, repo := range userRepos {
		key := fmt.Sprintf("%s-%v", providers[repo.ProviderID].Name, repo.RepoID)

		if repo.Enabled {
			enabled[key] = repo.ID
		}
	}

	for _, r := range rr {
		key := fmt.Sprintf("%s-%v", r.Provider.Name, r.RepoID)

		id, ok := enabled[key]

		r.ID = id
		r.Enabled = ok
		r.Provider = providers[r.ProviderID]
	}

	provider := r.URL.Query().Get("provider")

	p := repo.IndexPage{
		BasePage: template.BasePage{
			User: u,
			URL:  r.URL,
		},
		CSRF:     string(csrf.TemplateField(r)),
		Repos:    rr,
		Provider: provider,
	}

	d := template.NewDashboard(&p, r.URL, h.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Repo) Reload(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	pp, err := u.ProviderStore().All()

	if err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to refresh repository cache: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	rr, err := h.loadRepos(r.Context(), pp)

	if err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to refresh repository cache: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if err := h.cacheRepos(u.ID, rr); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to refresh repository cache: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Successfully reloaded repositories"))
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h Repo) Store(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	f := &form.Repo{}

	if err := form.Unmarshal(f, r); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to enable repository hooks: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	p, err := u.ProviderStore().FindByName(f.Provider)

	if err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to enable repository hooks: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	b, _ := crypto.Decrypt(p.AccessToken)

	provider := h.Providers[f.Provider]

	if err := provider.AddHook(r.Context(), string(b), f.RepoID); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to enable repository hooks: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	repos := u.RepoStore()

	rp := repos.New()
	rp.ProviderID = p.ID
	rp.Name = f.Name
	rp.RepoID = f.RepoID
	rp.Enabled = true

	if err := repos.Create(rp); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to enable repository hooks: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Enabled repository hooks for: " + rp.Name))
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h Repo) Destroy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	u := h.User(r)

	repos := u.RepoStore()

	id, _ := strconv.ParseInt(vars["repo"], 10, 64)

	rp, err := repos.Find(id)

	if err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to disable repository hooks: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if rp.IsZero() {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if err := rp.LoadProvider(); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to disable repository hooks: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	rp.Enabled = false

	if err := repos.Update(rp); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to disable repository hooks: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Disabled repository hooks for: " + rp.Name))
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
