package handler

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/config"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/provider"
	"github.com/andrewpillar/djinn/provider/github"
	"github.com/andrewpillar/djinn/provider/gitlab"
	"github.com/andrewpillar/djinn/runner"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"
)

var (
	invalidManifest = `The hook was accepted, however some of the manifests retrieved from the
repository appeared to contain invalid YAML, see below:`

	invalidNamespaceName = `Invalid namespace name in build manifest, the name can only contain letters or
numbers`
)

// hookData represents the minimum data extracted from a webhook's event payload
// we need to submit builds.
type hookData struct {
	userId  int64 // the id of the user from the provider who sent the hook
	repoId  int64 // the id of the repo from the provider
	dirurl  string
	ref     string
	typ     build.TriggerType
	comment string
	data    map[string]string
}

type Hook struct {
	Build

	Repos     *provider.RepoStore
	Providers *provider.Store
	Registry  *provider.Registry
}

type manifestDecoder func(io.Reader) (config.Manifest, error)

func decodeBase64JSONManifest(r io.Reader) (config.Manifest, error) {
	file := struct {
		Encoding string
		Content  string
	}{}

	if err := json.NewDecoder(r).Decode(&file); err != nil {
		return config.Manifest{}, errors.Err(err)
	}

	if file.Encoding != "base64" {
		return config.Manifest{}, errors.New("unexpected file encoding: " + file.Encoding)
	}

	raw, err := base64.StdEncoding.DecodeString(file.Content)

	if err != nil {
		return config.Manifest{}, errors.Err(err)
	}

	m, err := config.UnmarshalManifest([]byte(raw))
	return m, errors.Err(err)
}

func getGitHubURL(m map[string]string) string {
	if m["type"] == "dir" {
		return ""
	}

	if strings.HasSuffix(m["name"], ".yml") || strings.HasSuffix(m["name"], ".yaml") {
		return m["url"]
	}
	return ""
}

func getGitLabURL(rawurl, ref string) func(map[string]string) string {
	return func(m map[string]string) string {
		if m["type"] == "tree" {
			return ""
		}
		return rawurl + "/repository/files/" + url.QueryEscape(m["path"]) + "?ref=" + ref
	}
}

func (h Hook) execute(host, name string, data hookData, geturl func(map[string]string) string) error {
	h.Log.Debug.Println("data.userId =", data.userId)

	u, p, err := h.getUserAndProvider(name, data.repoId)

	if err != nil {
		return errors.Err(err)
	}

	r, err := provider.NewRepoStore(h.DB, p).Get(query.Where("repo_id", "=", data.repoId))

	if err != nil {
		return errors.Err(err)
	}

	b, err := h.Block.Decrypt(p.AccessToken)

	if err != nil {
		return errors.Err(err)
	}

	tok := string(b)

	dirurl := data.dirurl + "?ref=" + data.ref

	h.Log.Debug.Println("getting manifest urls from", dirurl)

	urls, err := h.getManifestURLs(tok, dirurl, geturl)

	if err != nil {
		return errors.Err(err)
	}

	urls = append(urls, data.dirurl + ".yml?ref=" + data.ref)

	mm, manifesterr := h.loadManifests(decodeBase64JSONManifest, tok, urls)

	t := &build.Trigger{
		ProviderID: sql.NullInt64{
			Int64: p.ID,
			Valid: true,
		},
		RepoID :   sql.NullInt64{
			Int64: r.ID,
			Valid: true,
		},
		Type:      data.typ,
		Comment:   data.comment,
		Data:      data.data,
	}

	h.Log.Debug.Println("submitting", len(mm), "build manifests")

	bb, err := h.submitBuilds(mm, host, u, t)

	if err != nil {
		return errors.Err(err)
	}

	if t.Type == build.Pull {
		last := bb[len(bb)-1]

		err = p.SetCommitStatus(h.Block, h.Registry, r, runner.Queued, host + last.Endpoint(), data.ref)

		if err != nil {
			return errors.Err(err)
		}
	}
	return manifesterr
}

func (h Hook) loadManifests(decode manifestDecoder, tok string, urls []string) ([]config.Manifest, error) {
	errs := make(chan error)
	manifests := make(chan config.Manifest)

	wg := &sync.WaitGroup{}

	for _, raw := range urls {
		wg.Add(1)

		go func(wg *sync.WaitGroup, raw string) {
			defer wg.Done()

			url, _ := url.Parse(raw)

			cli := &http.Client{}
			resp, err := cli.Do(&http.Request{
				Method: "GET",
				URL:    url,
				Header: http.Header(map[string][]string{
					"Authorization": []string{"Bearer " + tok},
				}),
			})

			if err != nil {
				errs <- err
				return
			}

			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				return
			}

			if resp.StatusCode != http.StatusOK {
				errs <- errors.New("unexpected http status for " + url.String() + ": " + resp.Status)
				return
			}

			parts := strings.Split(strings.Split(raw, "?")[0], "/")
			path := strings.Join(parts[len(parts)-1:], "/")

			m, err := decode(resp.Body)

			if err != nil {
				errs <- errors.New("failed to decode manifest: " + path + ": " + errors.Cause(err).Error())
				return
			}

			if err := m.Validate(); err != nil {
				errs <- errors.New("manifest invalid: " + path + ": " + err.Error())
				return
			}
			manifests <- m
		}(wg, raw)
	}

	go func() {
		wg.Wait()
		close(errs)
		close(manifests)
	}()

	mm := make([]config.Manifest, 0, len(urls))
	ee := make([]error, 0, len(urls))

	for errs != nil && manifests != nil {
		select {
		case err, ok := <-errs:
			if !ok {
				errs = nil
				break
			}
			ee = append(ee, err)
		case m, ok := <-manifests:
			if !ok {
				manifests = nil
				break
			}
			mm = append(mm, m)
		}
	}

	if len(ee) > 0 {
		return mm, errors.MultiError(ee...)
	}
	return mm, nil
}

func (h Hook) getUserAndProvider(name string, repoId int64) (*user.User, *provider.Provider, error) {
	r, err := h.Repos.Get(
		query.Where("repo_id", "=", repoId),
		query.Where("provider_name", "=", name),
	)

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	p, err := h.Providers.Get(query.Where("id", "=", r.ProviderID))

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	u, err := h.Users.Get(query.Where("id", "=", p.UserID))
	return u, p, errors.Err(err)
}

func (h Hook) getManifestURLs(tok, rawurl string, geturl func(map[string]string) string) ([]string, error) {
	urls := make([]string, 0)

	url, _ := url.Parse(rawurl)

	cli := &http.Client{}

	resp, err := cli.Do(&http.Request{
		Method: "GET",
		URL:    url,
		Header: http.Header(map[string][]string{
			"Authorization": []string{"Bearer " + tok},
		}),
	})

	if err != nil {
		return urls, errors.Err(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		h.Log.Debug.Println("no manifest urls found")
		return urls, nil
	}

	items := make([]map[string]string, 0)

	json.NewDecoder(resp.Body).Decode(&items)

	for _, it := range items {
		if rawurl := geturl(it); rawurl != "" {
			urls = append(urls, rawurl)
		}
	}
	return urls, nil
}

// submitBuilds will create and submit a build for each manifest in the given
// slice of manifests.
func (h Hook) submitBuilds(mm []config.Manifest, host string, u *user.User, t *build.Trigger) ([]*build.Build, error) {
	bb := make([]*build.Build, 0, len(mm))

	for _, m := range mm {
		b, err := build.NewStore(h.DB, u).Create(m, t)

		if err != nil {
			return nil, errors.Err(err)
		}
		bb = append(bb, b)
	}

	submitted := make([]*build.Build, 0, len(mm))

	for _, b := range bb {
		q := h.Queues[b.Manifest.Driver["type"]]

		if err := build.NewStoreWithHasher(h.DB, h.Hasher).Submit(q, host, b); err != nil {
			return nil, errors.Err(err)
		}
		submitted = append(submitted, b)
	}
	return submitted, nil
}

func (h Hook) GitHub(w http.ResponseWriter, r *http.Request) {
	h.Log.Debug.Println("github webhook received")

	event := r.Header.Get("X-GitHub-Event")

	cli, err := h.Registry.Get("github")

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
		return
	}

	body, err := cli.VerifyRequest(r.Body, r.Header.Get("X-Hub-Signature"))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
		return
	}

	h.Log.Debug.Println("github event type:", event)

	data := hookData{}

	switch event {
	case "ping":
		web.Text(w, "pong\n", http.StatusOK)
		return
	case "push":
		data.typ = build.Push

		push := github.PushEvent{}
		json.Unmarshal(body, &push)

		data.userId = push.Repo.Owner.ID
		data.repoId = push.Repo.ID
		data.dirurl = strings.Replace(push.Repo.ContentsURL, "{+path}", ".djinn", 1)
		data.ref = push.HeadCommit.ID
		data.comment = push.HeadCommit.Message
		data.data = map[string]string{
			"url":      push.HeadCommit.URL,
			"ref":      push.Ref,
			"sha":      push.HeadCommit.ID,
			"email":    push.HeadCommit.Author["email"],
			"username": push.HeadCommit.Author["username"],
		}
	case "pull_request":
		data.typ = build.Pull

		pull := github.PullRequestEvent{}
		json.Unmarshal(body, &pull)

		if _, ok := github.PullRequestActions[pull.Action]; !ok {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		data.userId = pull.PullRequest.Base.User.ID
		data.repoId = pull.PullRequest.Head.Repo.ID
		data.dirurl = strings.Replace(pull.PullRequest.Head.Repo.ContentsURL, "{+path}", ".djinn", 1)
		data.ref = pull.PullRequest.Head.Sha
		data.comment = pull.PullRequest.Title

		action := pull.Action

		if action == "synchronize" {
			action = "syncrhonized"
		}

		data.data = map[string]string{
			"id":       strconv.FormatInt(pull.Number, 10),
			"url":      pull.PullRequest.HTMLURL,
			"ref":      pull.PullRequest.Base.Ref,
			"sha":      pull.PullRequest.Head.Sha,
			"username": pull.PullRequest.User.Login,
			"action":   action,
		}
	}

	err = h.execute(web.BaseAddress(r), "github", data, getGitHubURL)

	if err != nil {
		cause := errors.Cause(err)

		if manifesterr, ok := cause.(*errors.Slice); ok {
			h.Log.Debug.Println("found some invalid manifests, responding with 202 Accepted")
			web.Text(w, invalidManifest+"\n\n"+manifesterr.Error(), http.StatusAccepted)
			return
		}

		if cause == namespace.ErrName {
			web.Text(w, invalidNamespaceName, http.StatusBadRequest)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.Text(w, cause.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Hook) GitLab(w http.ResponseWriter, r *http.Request) {
	h.Log.Debug.Println("gitlab webhook received")

	event := r.Header.Get("X-Gitlab-Event")

	cli, err := h.Registry.Get("gitlab")

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
		return
	}

	gl, ok := cli.(*gitlab.GitLab)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to type assert provider.Interface to *gitlab.GitLab")
	}

	body, err := cli.VerifyRequest(r.Body, r.Header.Get("X-Gitlab-Token"))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
		return
	}

	h.Log.Debug.Println("gitlab event type:", event)

	data := hookData{}

	switch event {
	case "Push Hook":
		data.typ = build.Push

		push := gitlab.PushEvent{}
		json.Unmarshal(body, &push)

		head := push.Commits[len(push.Commits)-1]

		data.userId = push.UserID
		data.repoId = push.Project.ID
		data.dirurl = gl.APIEndpoint + "/projects/" + strconv.FormatInt(push.Project.ID, 10) + "/repository/files/.djinn"
		data.ref = head.ID
		data.comment = head.Message
		data.data = map[string]string{
			"url":      head.URL,
			"ref":      push.Ref,
			"sha":      head.ID,
			"email":    head.Author["email"],
			"username": head.Author["username"],
		}
	case "Merge Request Hook":
		data.typ = build.Pull

		merge := gitlab.MergeRequestEvent{}
		json.Unmarshal(body, &merge)

		data.userId = merge.User.ID
		data.repoId = merge.Project.ID
		data.dirurl = gl.APIEndpoint + "/projects/" + strconv.FormatInt(merge.Attrs.SourceProjectID, 10) + "/repository/files/.djinn"
		data.ref = merge.Attrs.LastCommit.ID
		data.comment = merge.Attrs.Title
		data.data = map[string]string{
			"id":       strconv.FormatInt(merge.Attrs.ID, 10),
			"url":      merge.Attrs.URL,
			"ref":      merge.Attrs.TargetBranch,
			"sha":      merge.Attrs.LastCommit.ID,
			"username": merge.User.Name,
			"action":   merge.Attrs.Action,
		}
	}

	err = h.execute(web.BaseAddress(r), "gitlab", data, getGitLabURL(data.dirurl, data.ref))

	if err != nil {
		if manifesterr, ok := err.(*errors.Slice); ok {
			h.Log.Debug.Println("found some invalid manifests, responding with 202 Accepted")
			web.Text(w, invalidManifest+"\n\n"+manifesterr.Error(), http.StatusAccepted)
			return
		}

		cause := errors.Cause(err)

		if cause == namespace.ErrName {
			web.Text(w, invalidNamespaceName, http.StatusBadRequest)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.Text(w, cause.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
