package provider

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/manifest"
	"djinn-ci.com/runner"
)

var (
	ErrUnknownEvent     = errors.New("provider: unknown webhook event")
	ErrInvalidSignature = errors.New("provider: invalid signature")
	ErrStateMismatch    = errors.New("provider: state mismatch")
	ErrLocalhost        = errors.New("provider: cannot use localhost for webhook URL")
	ErrNilRegistry      = errors.New("provider: nil client registry")

	StatusDescriptions = map[runner.Status]string{
		runner.Queued:             "Build is queued.",
		runner.Running:            "Build is running.",
		runner.Passed:             "Build has passed.",
		runner.PassedWithFailures: "Build has passed with failures.",
		runner.Failed:             "Build has failed.",
		runner.Killed:             "Build was killed.",
		runner.TimedOut:           "Build has timed out.",
	}
)

type WebhookEvent uint

const (
	PushEvent WebhookEvent = iota + 1
	PullEvent
)

type WebhookData struct {
	UserID  int64
	RepoID  int64
	Event   WebhookEvent
	DirURL  string
	Ref     string
	Comment string
	Data    map[string]string
	Host    string
}

type ParseWebhookFunc func(r *http.Request) (*WebhookData, error)

type Client interface {
	// Repos returns a list of repos from the provider the user either owns or
	// has access to (via a group). The given page delineates which page of
	// repositories to get if there are multiple pages. This should return the
	// repositories ordered by when they were last updated.
	Repos(page int) (*database.Paginator[*Repo], error)

	// Groups returns the ID of groups that a user is a member of from the
	// provider (if any/supported). This will be used to create multiple
	// providers in the database for each group, allowing for easy lookup
	// during webhook execution, should any webhook contain a group ID in place
	// of a user ID.
	Groups() ([]int64, error)

	// ToggleWebhook will toggle the webhook for the given Repo using the given
	// string as the authentication token. This will create a webhook if it
	// doesn't exist for the Repo, otherwise it will delete it.
	ToggleWebhook(r *Repo) error

	// SetCommitStatus will set the commit status for the given commit hash to
	// the given runner.Status. This should only be called on commits that
	// belong to a merge or pull request.
	SetCommitStatus(r *Repo, status runner.Status, url, sha string) error

	// Manifests returns all of the build manifests that could be found in the
	// repository contents from the root URL at the given ref.
	Manifests(root, ref string) ([]manifest.Manifest, error)

	// Verify verifies the given Request. This is typically used for verifying
	// the contents of a webhook that has been sent from said provider. Upon
	// success it will return a byte slice read from the given io.Reader.
	Verify(r *http.Request) ([]byte, error)
}

type BaseClient struct {
	*http.Client

	token string

	// Init initializes the BaseClient and returns a Client interface.
	Init func(c *BaseClient) Client

	Host     string
	Secret   string
	Endpoint string
}

func (c *BaseClient) NewRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, c.Endpoint+url, body)

	if err != nil {
		return nil, errors.Err(err)
	}

	req.Header.Set("Accept", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	return req, nil
}

func (c *BaseClient) Get(url string) (*http.Request, error) {
	req, err := c.NewRequest("GET", url, nil)

	if err != nil {
		return nil, errors.Err(err)
	}
	return req, nil
}

func (c *BaseClient) Post(url string, body io.Reader) (*http.Request, error) {
	req, err := c.NewRequest("POST", url, body)

	if err != nil {
		return nil, errors.Err(err)
	}
	return req, nil
}

func (c *BaseClient) Delete(url string) (*http.Request, error) {
	req, err := c.NewRequest("DELETE", url, nil)

	if err != nil {
		return nil, errors.Err(err)
	}
	return req, nil
}

func (c *BaseClient) Do(r *http.Request) (*http.Response, error) {
	if c.Client == nil {
		c.Client = http.DefaultClient
		c.Client.Timeout = time.Minute
	}

	resp, err := c.Client.Do(r)

	if err != nil {
		return nil, errors.Err(err)
	}
	return resp, nil
}

func UnmarshalBase64Manifest(r io.Reader) (manifest.Manifest, error) {
	var m manifest.Manifest

	var file struct {
		Encoding, Content string
	}

	json.NewDecoder(r).Decode(&file)

	if file.Encoding != "base64" {
		return m, errors.New("provider: unexpected file encoding: " + file.Encoding)
	}

	raw, err := base64.StdEncoding.DecodeString(file.Content)

	if err != nil {
		return m, errors.Err(err)
	}

	m, err = manifest.Unmarshal([]byte(raw))

	if err != nil {
		return m, errors.Err(err)
	}

	if err := m.Validate(); err != nil {
		return m, errors.Err(err)
	}
	return m, nil
}

func (c *BaseClient) LoadManifests(urls []string, unmarshal func(io.Reader) (manifest.Manifest, error)) ([]manifest.Manifest, error) {
	var wg sync.WaitGroup
	wg.Add(len(urls))

	ch := make(chan manifest.Manifest)
	errs := make(chan error)

	for _, rawurl := range urls {
		go func(wg *sync.WaitGroup, rawurl string) {
			defer wg.Done()

			url, _ := url.Parse(rawurl)

			req, err := c.Get(url.Path)

			if err != nil {
				errs <- err
				return
			}

			resp, err := c.Do(req)

			if err != nil {
				errs <- err
				return
			}

			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				return
			}

			if resp.StatusCode != http.StatusOK {
				errs <- errors.New("provider: unexpected http status for " + rawurl + ": " + resp.Status)
				return
			}

			m, err := unmarshal(resp.Body)

			if err != nil {
				errs <- errors.Err(err)
				return
			}
			ch <- m
		}(&wg, rawurl)
	}

	go func() {
		wg.Wait()
		close(errs)
		close(ch)
	}()

	mm := make([]manifest.Manifest, 0, len(urls))
	msgs := make([]string, 0, len(urls))

	for errs != nil && ch != nil {
		select {
		case err, ok := <-errs:
			if !ok {
				errs = nil
				break
			}
			msgs = append(msgs, err.Error())
		case m, ok := <-ch:
			if !ok {
				ch = nil
				break
			}
			mm = append(mm, m)
		}
	}

	if len(msgs) > 0 {
		return nil, errors.Slice(msgs)
	}
	return mm, nil
}

type Registry struct {
	mu      sync.RWMutex
	clients map[string]*BaseClient
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.clients))

	for name := range r.clients {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

func (r *Registry) Register(name string, cli *BaseClient) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.clients == nil {
		r.clients = make(map[string]*BaseClient)
	}

	if _, ok := r.clients[name]; ok {
		panic("provider: client already registered: " + name)
	}
	r.clients[name] = cli
}

func (r *Registry) Get(tok, name string) (Client, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cli, ok := r.clients[name]

	if !ok {
		return nil, errors.New("provider: no such client: " + name)
	}

	cli2 := (*cli)
	cli2.Client = cli.Client
	cli2.token = tok

	return cli.Init(&cli2), nil
}

// ParseLink parses the given string value as the value of the Link header
// (RFC-5988, RFC-8288). The rel for each of these links is used as the key,
// and if duplicate rels are provided then the values are simply appended to
// the respective string slice in the returned map.
//
// For example, the following,
//
//	Link: <https://example.org/>; rel="start",
//	      <https://example.org/index>; rel="index"
//
// would be parsed into the following map,
//
//	map[string][]string{
//	    "start": []string{"https://example.org"},
//	    "index": []string{"https://example.org/index"},
//	}
func ParseLink(link string) map[string][]string {
	m := make(map[string][]string)
	parts := strings.Split(link, ",")

	for _, part := range parts {
		var (
			raw  []byte
			rel  []byte
			dst  *[]byte
			scan bool
		)

		for i := 0; i < len(part)-1; i++ {
			b := part[i]

			// We hit the raw URL, so set the dst to point to the raw slice.
			if b == '<' {
				scan = true
				dst = &raw
				continue
			}
			if b == '>' {
				scan = false
				continue
			}
			if b == ';' {
				// Make sure we're in bounds of what we're scanning when getting
				// the rel attribute of the value.
				if i+6 > len(part)-1 {
					continue
				}

				// Check to see if the next part is the rel attribute, if so
				// then we skip ahead to the actual value of the attribute.
				if part[i+2:i+5] == "rel" {
					scan = true
					dst = &rel
					i += 6
					continue
				}
			}
			// Put whatever we have into the destination byte slice pointed to
			// via dst. This will either be the raw URL or the value of rel.
			if scan {
				(*dst) = append((*dst), b)
			}
		}
		m[string(rel)] = append(m[string(rel)], string(raw))
	}
	return m
}
