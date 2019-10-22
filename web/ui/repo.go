package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/repo"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"

	"github.com/google/go-github/github"

	"github.com/go-redis/redis"

	"golang.org/x/oauth2"
)

type Repo struct {
	web.Handler

	Redis *redis.Client
}

type loadRepos func(c context.Context, tok string) ([]model.Repository, error)

var repoLoaders = map[string]loadRepos{
	"github": githubRepos,
	"gitlab": gitlabRepos,
}

func forProvider(name string) query.Option {
	return func(q query.Query) query.Query {
		if name == "" {
			return q
		}

		return query.Where("name", "=", name)(q)
	}
}

func githubRepos(c context.Context, tok string) ([]model.Repository, error) {
	oauthTok := &oauth2.Token{
		AccessToken: tok,
	}

	src := oauth2.StaticTokenSource(oauthTok)
	cli := github.NewClient(oauth2.NewClient(c, src))

	opt := &github.RepositoryListOptions{
		Sort:      "updated",
		Direction: "desc",
	}

	repos, _, err := cli.Repositories.List(c, "", opt)

	if err != nil {
		return []model.Repository{}, errors.Err(err)
	}

	rr := make([]model.Repository, 0, len(repos))

	for _, repo := range repos {
		var (
			id   int64
			name string
			href string
		)

		if repo.ID != nil {
			id = *repo.ID
		}

		if repo.FullName != nil {
			name = *repo.FullName
		}

		if repo.HTMLURL != nil {
			href = *repo.HTMLURL
		}

		r := model.Repository{
			ID:       id,
			Name:     name,
			Href:     href,
			Provider: "github",
		}

		rr = append(rr, r)
	}

	return rr, nil
}

func gitlabRepos(c context.Context, tok string) ([]model.Repository, error) {
	return []model.Repository{}, nil
}

func (h Repo) getCached() ([]model.Repository, error) {
	repos := make([]model.Repository, 0)

	s, err := h.Redis.Get("repos").Result()

	if err != nil {
		return repos, errors.Err(err)
	}

	dec := json.NewDecoder(strings.NewReader(s))
	dec.Decode(repos)

	return repos, nil
}

func (h Repo) cacheRepos(rr []model.Repository) error {
	buf := &bytes.Buffer{}

	enc := json.NewEncoder(buf)
	enc.Encode(rr)

	_, err := h.Redis.Set("repos", buf.String(), time.Hour).Result()

	return errors.Err(err)
}

func (h Repo) loadRepos(c context.Context, providers model.ProviderStore) ([]model.Repository, error) {
	pp, err := providers.All()

	rr := make([]model.Repository, 0)

	for _, p := range pp {
		b, _ := crypto.Decrypt(p.AccessToken)

		tmp, err := repoLoaders[p.Name](c, string(b))

		if err != nil {
			return rr, errors.Err(err)
		}

		rr = append(rr, tmp...)
	}

	return rr, nil
}

func (h Repo) Index(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	rr, err := h.getCached()

	if len(rr) == 0 {
		rr, err = h.loadRepos(r.Context(), u.ProviderStore())

		if err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		if err := h.cacheRepos(rr); err != nil {
			log.Error.Println(errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}
	}

	name := r.URL.Query().Get("provider")

	if name != "" {
		tmp := make([]model.Repository, 0, len(rr))

		for _, r := range rr {
			if r.Name == name {
				tmp = append(tmp, r)
			}
		}

		rr = tmp
	}

	p := &repo.IndexPage{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Repos:    rr,
		CSRF:     string(csrf.TemplateField(r)),
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r), string(csrf.TemplateField(r)))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Repo) Reload(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	rr, err := h.loadRepos(r.Context(), u.ProviderStore());

	if err != nil {
		log.Error.Println(errors.Err(err))

		cause := errors.Cause(err)

		h.FlashAlert(w, r, template.Danger("Failed to refresh repository cache: " + cause.Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if err := h.cacheRepos(rr); err != nil {
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
