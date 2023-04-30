package http

import (
	"bytes"
	"io"
	"net/http"

	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/provider"
	"djinn-ci.com/provider/github"
	"djinn-ci.com/provider/gitlab"
	"djinn-ci.com/runner"
	"djinn-ci.com/server"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"
)

type Hook struct {
	*server.Server
}

func (h Hook) Handler(name string, parseHook provider.ParseWebhookFunc) http.HandlerFunc {
	errh := func(w http.ResponseWriter, err error, code int) {
		webutil.Text(w, err.Error(), code)
	}

	providers := &provider.Store{
		Store:   provider.NewStore(h.DB),
		AESGCM:  h.AESGCM,
		Clients: h.Providers,
	}
	repos := provider.NewRepoStore(h.DB)

	users := user.NewStore(h.DB)

	builds := build.Store{
		Store:  build.NewStore(h.DB),
		Hasher: h.Hasher,
		Queues: h.DriverQueues,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		h.Log.Debug.Println(name, "webhook received")

		cli, err := h.Providers.Get("", name)

		if err != nil {
			h.Log.Error.Println(req.Method, req.URL.Path, errors.Err(err))
			errh(w, err, http.StatusInternalServerError)
			return
		}

		body, err := cli.Verify(req)

		if err != nil {
			if errors.Is(err, provider.ErrInvalidSignature) {
				errh(w, err, http.StatusBadRequest)
				return
			}
			h.Log.Error.Println(req.Method, req.URL.Path, errors.Err(err))
			errh(w, err, http.StatusInternalServerError)
			return
		}

		req.Body = io.NopCloser(bytes.NewReader(body))

		data, err := parseHook(req)

		if err != nil {
			if errors.Is(err, provider.ErrUnknownEvent) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if errors.Is(err, github.ErrInvalidAction) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if errors.Is(err, gitlab.ErrNoCommits) {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			h.Log.Error.Println(req.Method, req.URL.Path, errors.Err(err))
			errh(w, err, http.StatusInternalServerError)
			return
		}

		data.Host = h.Host

		h.Log.Debug.Println("data.UserID =", data.UserID)

		ctx := req.Context()

		r, _, err := repos.SelectOne(
			ctx,
			[]string{"provider_id"},
			query.Where("repo_id", "=", query.Arg(data.RepoID)),
			query.Where("provider_name", "=", query.Arg(name)),
		)

		if err != nil {
			h.Log.Error.Println(req.Method, req.URL.Path, errors.Err(err))
			errh(w, err, http.StatusInternalServerError)
			return
		}

		p, _, err := providers.SelectOne(
			ctx,
			[]string{"id", "user_id", "access_token"},
			query.Where("repo_id", "=", query.Arg(data.RepoID)),
			query.Where("provider_name", "=", query.Arg(name)),
		)

		if err != nil {
			h.Log.Error.Println(req.Method, req.URL.Path, errors.Err(err))
			errh(w, err, http.StatusInternalServerError)
			return
		}

		u, _, err := users.Get(ctx, user.WhereID(p.UserID))

		if err != nil {
			h.Log.Error.Println(req.Method, req.URL.Path, errors.Err(err))
			errh(w, err, http.StatusInternalServerError)
			return
		}

		cli = p.Client()

		h.Log.Debug.Println("getting build manifests from", data.DirURL, "at", data.Ref)

		mm, err := cli.Manifests(data.DirURL, data.Ref)

		if err != nil {
			h.Log.Error.Println(req.Method, req.URL.Path, errors.Err(err))
			errh(w, err, http.StatusInternalServerError)
			return
		}

		h.Log.Debug.Println("submitting", len(mm), "build manifests")

		triggerTypes := map[provider.WebhookEvent]build.TriggerType{
			provider.PushEvent: build.Push,
			provider.PullEvent: build.Pull,
		}

		t := build.Trigger{
			ProviderID: database.Null[int64]{
				Elem:  p.ID,
				Valid: true,
			},
			RepoID: database.Null[int64]{
				Elem:  r.ID,
				Valid: true,
			},
			Type:    triggerTypes[data.Event],
			Comment: data.Comment,
			Data:    data.Data,
		}

		var tag string

		if t.Type == build.Push || t.Type == build.Pull {
			tag = t.Data["ref"]
		}

		bb := make([]*build.Build, 0, len(mm))

		for _, m := range mm {
			b, err := builds.Create(ctx, &build.Params{
				User:     u,
				Trigger:  &t,
				Manifest: m,
				Tags:     []string{tag},
			})

			if err != nil {
				h.Log.Error.Println(req.Method, req.URL.Path, errors.Err(err))
				errh(w, err, http.StatusInternalServerError)
				return
			}
			bb = append(bb, b)
		}

		for _, b := range bb {
			if err := builds.Submit(ctx, data.Host, b); err != nil {
				h.Log.Error.Println(req.Method, req.URL.Path, errors.Err(err))
				errh(w, err, http.StatusInternalServerError)
				return
			}
		}

		if t.Type == build.Pull {
			last := bb[len(bb)-1]

			if err := cli.SetCommitStatus(r, runner.Queued, data.Host+last.Endpoint(), data.Ref); err != nil {
				h.Log.Error.Println(req.Method, req.URL.Path, errors.Err(err))
				errh(w, err, http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	})
}

func RegisterHooks(srv *server.Server) {
	h := Hook{
		Server: srv,
	}

	srv.Router.HandleFunc("/hook/github", h.Handler("github", github.ParseWebhook))
	srv.Router.HandleFunc("/hook/gitlab", h.Handler("gitlab", gitlab.ParseWebhook))
}
