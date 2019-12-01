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

		tmp, err := provider.Repos(p)

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

	userRepos, err := u.RepoStore().All()

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	for _, r := range rr {
		for _, userRepo := range userRepos {
			if userRepo.ProviderID == r.ProviderID && userRepo.RepoID == r.RepoID {
				r.ID = userRepo.ID
				r.Enabled = userRepo.Enabled
			}
		}

		r.Provider = providers[r.ProviderID]
	}

	provider := r.URL.Query().Get("provider")

	if provider != "" {
		for i, r := range rr {
			if r.Provider.Name != provider {
				rr = append(rr[:i], rr[i+1:]...)
				i--
			}
		}
	}

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

	if err := h.Providers[f.Provider].ToggleRepo(p, f.RepoID); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to enable repository hooks: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Repository hooks enabled"))
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h Repo) Destroy(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	u := h.User(r)

	id, _ := strconv.ParseInt(vars["repo"], 10, 64)

	repo, err := u.RepoStore().Find(id)

	if err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to disable repository hooks: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if repo.IsZero() {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if !repo.Enabled {
		h.FlashAlert(w, r, template.Success("Repository hooks disabled"))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if err := repo.LoadProvider(); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to disable repository hooks: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if err := h.Providers[repo.Provider.Name].ToggleRepo(repo.Provider, repo.RepoID); err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to disable repository hooks: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Repository hooks disabled"))
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
