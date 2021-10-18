package handler

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"djinn-ci.com/build"
	"djinn-ci.com/crypto"
	"djinn-ci.com/errors"
	"djinn-ci.com/manifest"
	"djinn-ci.com/namespace"
	"djinn-ci.com/provider"
	"djinn-ci.com/provider/github"
	"djinn-ci.com/provider/gitlab"
	"djinn-ci.com/runner"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
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
	userId  int64  // the id of the user from the provider who sent the hook
	repoId  int64  // the id of the repo from the provider
	dirurl  string // the url in the repo that can contain multiple manifests, typically ".djinn"
	ref     string // the git ref we should use when retrieving manifests from the repo
	typ     build.TriggerType
	comment string
	data    map[string]string // additional information about the hook, such as build author
}

// Hook providers the handlers used for consuming webhooks from the various
// providers we integrate with.
type Hook struct {
	Build

	crypto    *crypto.AESGCM
	providers *provider.Registry
}

type manifestDecoder func(io.Reader) (manifest.Manifest, error)

func NewHook(b Build, crypto *crypto.AESGCM, providers *provider.Registry) Hook {
	return Hook{
		Build:     b,
		crypto:    crypto,
		providers: providers,
	}
}

// decodeBase64JSONManifest is the decoded used for decoding manifest files
// from the GitHub and GitLab API whereby the actual file contents if a base64
// encoded string.
func decodeBase64JSONManifest(r io.Reader) (manifest.Manifest, error) {
	file := struct {
		Encoding string
		Content  string
	}{}

	if err := json.NewDecoder(r).Decode(&file); err != nil {
		return manifest.Manifest{}, errors.Err(err)
	}

	if file.Encoding != "base64" {
		return manifest.Manifest{}, errors.New("unexpected file encoding: " + file.Encoding)
	}

	raw, err := base64.StdEncoding.DecodeString(file.Content)

	if err != nil {
		return manifest.Manifest{}, errors.Err(err)
	}

	m, err := manifest.Unmarshal([]byte(raw))
	return m, errors.Err(err)
}

func getGitHubURL(m map[string]string) string {
	if m["type"] == "dir" {
		return ""
	}

	if strings.HasSuffix(m["name"], ".yml") {
		return m["url"]
	}
	return ""
}

func getGitLabURL(rawurl, ref string) func(map[string]string) string {
	return func(m map[string]string) string {
		if m["type"] == "tree" {
			return ""
		}

		if strings.HasSuffix(m["path"], ".yml") {
			return rawurl + "/repository/files/" + url.QueryEscape(m["path"]) + "?ref=" + ref
		}
		return ""
	}
}

// execute is what will actual get the manifests, decode them, and then submit
// them to the build queue. It uses the information provided in the given hook
// data to determine the type of build being submitted.
func (h Hook) execute(ctx context.Context, host, name string, data hookData, geturl func(map[string]string) string) error {
	h.Log.Debug.Println("data.userId =", data.userId)

	u, p, err := h.getUserAndProvider(name, data.repoId)

	if err != nil {
		return errors.Err(err)
	}

	r, err := provider.NewRepoStore(h.DB, p).Get(query.Where("repo_id", "=", query.Arg(data.repoId)))

	if err != nil {
		return errors.Err(err)
	}

	b, err := h.crypto.Decrypt(p.AccessToken)

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

	urls = append(urls, data.dirurl+".yml?ref="+data.ref)

	mm, manifesterr := h.loadManifests(decodeBase64JSONManifest, tok, urls)

	t := &build.Trigger{
		ProviderID: sql.NullInt64{
			Int64: p.ID,
			Valid: true,
		},
		RepoID: sql.NullInt64{
			Int64: r.ID,
			Valid: true,
		},
		Type:    data.typ,
		Comment: data.comment,
		Data:    data.data,
	}

	h.Log.Debug.Println("submitting", len(mm), "build manifests")

	bb, err := h.submitBuilds(ctx, mm, host, u, t)

	if err != nil {
		return errors.Err(err)
	}

	if t.Type == build.Pull {
		last := bb[len(bb)-1]

		err = p.SetCommitStatus(h.crypto, h.providers, r, runner.Queued, host+last.Endpoint(), data.ref)

		if err != nil {
			return errors.Err(err)
		}
	}
	return manifesterr
}

// loadManifests concurrently hits each of the given urls to get the manifest
// from each. It uses the given decode function to decode the manifest into
// a useable form. If any errors occur during decoding of the manifest then
// this will be aggregated into errors.MultiError and returned. Any manifest
// decoding errors that do occur does not stop the rest of the manifest urls
// from being requested.
func (h Hook) loadManifests(decode manifestDecoder, tok string, urls []string) ([]manifest.Manifest, error) {
	errs := make(chan error)
	manifests := make(chan manifest.Manifest)

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
					"Authorization": {"Bearer " + tok},
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

	mm := make([]manifest.Manifest, 0, len(urls))
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
	r, err := provider.NewRepoStore(h.DB).Get(
		query.Where("repo_id", "=", query.Arg(repoId)),
		query.Where("provider_name", "=", query.Arg(name)),
	)

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	p, err := provider.NewStore(h.DB).Get(query.Where("id", "=", query.Arg(r.ProviderID)))

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	u, err := h.Users.Get(query.Where("id", "=", query.Arg(p.UserID)))
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
			"Authorization": {"Bearer " + tok},
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
func (h Hook) submitBuilds(ctx context.Context, mm []manifest.Manifest, host string, u *user.User, t *build.Trigger) ([]*build.Build, error) {
	bb := make([]*build.Build, 0, len(mm))

	var tag string

	switch t.Type {
	case build.Push:
		tag = t.Data["ref"]
	case build.Pull:
		tag = t.Data["ref"]
	}

	for _, m := range mm {
		b, err := build.NewStore(h.DB, u).Create(m, t, tag)

		if err != nil {
			return nil, errors.Err(err)
		}
		bb = append(bb, b)
	}

	submitted := make([]*build.Build, 0, len(mm))

	for _, b := range bb {
		prd, ok := h.getDriverQueue(b.Manifest)

		if !ok {
			return nil, build.ErrDriverDisabled
		}

		if err := build.NewStoreWithHasher(h.DB, h.hasher).Submit(ctx, prd, host, b); err != nil {
			return nil, errors.Err(err)
		}
		submitted = append(submitted, b)
	}
	return submitted, nil
}

// GitHub serves the response for webhooks received from GitHub. This will only
// process webhooks for "ping", "push", and "pull_request" events. The owner of
// each build submitted will be set to the owner of the GitHub access token.
// Each request sent from GitHub is verified using the HMAC signature sent with
// the request. This will respond with 204 No Content on a successful execution
// of a hook, otherwise it will respond with 500 Internal Server Error. If any
// invalid manifest is found during hook execution, then the response
// 202 Accepted is sent, with a message detailing what was wrong with any
// invalid manifest that was found. If an invalid namespace name is found in
// any of the manifests then 400 Bad Request is sent.
//
// On a "push" event, the repository being pushed to will be scraped to have
// all manifests extraced from it. Each of these will be submitted as a new
// build. The most recent commit is used as the build not, and the commit
// author is used as the author of the build.
//
// On a "pull_request" event, the repository from which the pull request was
// sent is scraped for all manifests to be extracted. Each of these will be
// submitted as a new build. The title and body of the pull request are used
// as the build note.
func (h Hook) GitHub(w http.ResponseWriter, r *http.Request) {
	h.Log.Debug.Println("github webhook received")

	event := r.Header.Get("X-GitHub-Event")

	cli, err := h.providers.Get("github")

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		webutil.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
		return
	}

	body, err := cli.VerifyRequest(r.Body, r.Header.Get("X-Hub-Signature"))

	if err != nil {
		webutil.Text(w, errors.Cause(err).Error(), http.StatusBadRequest)
		return
	}

	h.Log.Debug.Println("github event type:", event)

	data := hookData{}

	switch event {
	case "ping":
		webutil.Text(w, "pong\n", http.StatusOK)
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
			"repo_id":  strconv.FormatInt(push.Repo.ID, 10),
			"provider": "github",
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
		data.repoId = pull.PullRequest.Base.Repo.ID
		data.dirurl = strings.Replace(pull.PullRequest.Head.Repo.ContentsURL, "{+path}", ".djinn", 1)
		data.ref = pull.PullRequest.Head.Sha
		data.comment = pull.PullRequest.Title

		action := pull.Action

		if action == "synchronize" {
			action = "syncrhonized"
		}

		data.data = map[string]string{
			"repo_id":  strconv.FormatInt(pull.PullRequest.Base.Repo.ID, 10),
			"provider": "github",
			"id":       strconv.FormatInt(pull.Number, 10),
			"url":      pull.PullRequest.HTMLURL,
			"ref":      pull.PullRequest.Base.Ref,
			"sha":      pull.PullRequest.Head.Sha,
			"username": pull.PullRequest.User.Login,
			"action":   action,
		}
	}

	err = h.execute(r.Context(), webutil.BaseAddress(r), "github", data, getGitHubURL)

	if err != nil {
		cause := errors.Cause(err)

		if manifesterr, ok := cause.(*errors.Slice); ok {
			h.Log.Debug.Println("found some invalid manifests, responding with 202 Accepted")
			webutil.Text(w, invalidManifest+"\n\n"+manifesterr.Error(), http.StatusAccepted)
			return
		}

		if cause == build.ErrDriverDisabled {
			webutil.Text(w, cause.Error(), http.StatusBadRequest)
			return
		}

		if cause == namespace.ErrName {
			webutil.Text(w, invalidNamespaceName, http.StatusBadRequest)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		webutil.Text(w, cause.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GitLab serves the response for webhooks received from GitLab. This will only
// process webhooks for "Push Hook", and "Merge Request Hook", events. The
// owner of each build submitted will be set to the owner of the GitLab access
// token. Each request sent from GitLab is verified by checking the webhook
// token that was sent in the request. This will respond with 204 No Content on
// a successful execution of a hook, otherwise it will respond with
// 500 Internal Server Error. If any invalid manifest is found during hook
// execution, then the response 202 Accepted is sent, with a message detailing
// what was wrong with any invalid manifest that was found. If an invalid
// namespace name is found in any of the manifests then 400 Bad Request is sent.
//
// On a "Push Hook" event, the repository being pushed to will be scraped to
// have all manifests extraced from it. Each of these will be submitted as a
// new build. The most recent commit is used as the build not, and the commit
// author is used as the author of the build.
//
// On a "Merge Request Hook" event, the repository from which the pull request
// was sent is scraped for all manifests to be extracted. Each of these will be
// submitted as a new build. The title and body of the pull request are used
// as the build note.
func (h Hook) GitLab(w http.ResponseWriter, r *http.Request) {
	h.Log.Debug.Println("gitlab webhook received")

	event := r.Header.Get("X-Gitlab-Event")

	cli, err := h.providers.Get("gitlab")

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		webutil.Text(w, errors.Cause(err).Error(), http.StatusInternalServerError)
		return
	}

	gl, ok := cli.(*gitlab.Client)

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to type assert provider.Client to *gitlab.Client")
	}

	body, err := cli.VerifyRequest(r.Body, r.Header.Get("X-Gitlab-Token"))

	if err != nil {
		webutil.Text(w, errors.Cause(err).Error(), http.StatusBadRequest)
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
			"repo_id":  strconv.FormatInt(push.Project.ID, 10),
			"provider": "gitlab",
			"url":      head.URL,
			"ref":      push.Ref,
			"sha":      head.ID,
			"email":    head.Author["email"],
			"username": head.Author["username"],
		}
	case "Merge Request Hook":
		actions := map[string]string{
			"open":   "opened",
			"update": "updated",
		}

		data.typ = build.Pull

		merge := gitlab.MergeRequestEvent{}
		json.Unmarshal(body, &merge)

		data.userId = merge.User.ID
		data.repoId = merge.Project.ID
		data.dirurl = gl.APIEndpoint + "/projects/" + strconv.FormatInt(merge.Attrs.SourceProjectID, 10) + "/repository/files/.djinn"
		data.ref = merge.Attrs.LastCommit.ID
		data.comment = merge.Attrs.Title
		data.data = map[string]string{
			"repo_id":  strconv.FormatInt(merge.Attrs.TargetProjectID, 10),
			"provider": "gitlab",
			"id":       strconv.FormatInt(merge.Attrs.ID, 10),
			"url":      merge.Attrs.URL,
			"ref":      merge.Attrs.TargetBranch,
			"sha":      merge.Attrs.LastCommit.ID,
			"username": merge.User.Username,
			"action":   actions[merge.Attrs.Action],
		}
	}

	err = h.execute(r.Context(), webutil.BaseAddress(r), "gitlab", data, getGitLabURL(data.dirurl, data.ref))

	if err != nil {
		if manifesterr, ok := err.(*errors.Slice); ok {
			h.Log.Debug.Println("found some invalid manifests, responding with 202 Accepted")
			webutil.Text(w, invalidManifest+"\n\n"+manifesterr.Error(), http.StatusAccepted)
			return
		}

		cause := errors.Cause(err)

		if cause == build.ErrDriverDisabled {
			webutil.Text(w, cause.Error(), http.StatusBadRequest)
			return
		}

		if cause == namespace.ErrName {
			webutil.Text(w, invalidNamespaceName, http.StatusBadRequest)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		webutil.Text(w, cause.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
