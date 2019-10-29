package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/repo"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"

	"github.com/go-redis/redis"
)

type Repo struct {
	web.Handler

	Redis     *redis.Client
	Providers map[string]oauth2.Provider
}

func (h Repo) cacheRepos(id int64, rr []model.Repo) error {
	buf := &bytes.Buffer{}

	enc := json.NewEncoder(buf)
	enc.Encode(rr)

	_, err := h.Redis.Set(fmt.Sprintf("repos-%v", id), buf.String(), time.Hour).Result()

	return errors.Err(err)
}

func (h Repo) getCached(id int64) ([]model.Repo, error) {
	rr := make([]model.Repo, 0)

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

func (h Repo) loadRepos(c context.Context, providers model.ProviderStore) ([]model.Repo, error) {
	pp, err := providers.All()

	rr := make([]model.Repo, 0)

	if err != nil {
		return rr, errors.Err(err)
	}

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

		rr = append(rr, tmp...)
	}

	return rr, nil
}

func (h Repo) Index(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	rr, err := h.getCached(u.ID)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if len(rr) == 0 {
		rr, err = h.loadRepos(r.Context(), u.ProviderStore())

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

	if err := repos.LoadProviders(userRepos); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	enabled := make(map[string]struct{})

	for _, repo := range userRepos {
		enabled[fmt.Sprintf("%s-%v", repo.Provider.Name, repo.RepoID)] = struct{}{}
	}

	for _, r := range rr {
		_, ok := enabled[fmt.Sprintf("%s-%v", r.Provider.Name, r.RepoID)]

		r.Enabled = ok
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

	rr, err := h.loadRepos(r.Context(), u.ProviderStore())

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

}

func (h Repo) Destroy(w http.ResponseWriter, r *http.Request) {

}
