package djinn

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type Build struct {
	ID           int64         `json:"id"`
	UserID       int64         `json:"user_id"`
	NamespaceID  NullInt64     `json:"namespace_id"`
	Number       int64         `json:"number"`
	Manifest     string        `json:"manifest"`
	Status       Status        `json:"status"`
	Output       NullString    `json:"output"`
	Tags         []string      `json:"tags"`
	CreatedAt    Time          `json:"created_at"`
	StartedAt    NullTime      `json:"started_at"`
	FinishedAt   NullTime      `json:"finished_at"`
	URL          URL           `json:"url"`
	ObjectsURL   URL           `json:"objects_url"`
	VariablesURL URL           `json:"variables_url"`
	JobsURL      URL           `json:"jobs_url"`
	ArtifactsURL URL           `json:"artifacts_url"`
	TagsURL      URL           `json:"tags_url"`
	User         *User         `json:"user"`
	Trigger      *BuildTrigger `json:"trigger"`
	Namespace    *Namespace    `json:"namespace"`
}

type BuildJob struct {
	ID         int64      `json:"id"`
	BuildID    int64      `json:"id"`
	Name       string     `json:"name"`
	Commands   string     `json:"commands"`
	Status     Status     `json:"status"`
	Output     NullString `json:"output"`
	CreatedAt  Time       `json:"created_at"`
	StartedAt  NullTime   `json:"started_at"`
	FinishedAt NullTime   `json:"finished_at"`
	URL        URL        `json:"url"`
}

type Artifact struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	BuildID   int64      `json:"build_id"`
	JobID     int64      `json:"job_id"`
	Source    string     `json:"source"`
	Name      string     `json:"name"`
	Size      NullInt64  `json:"size"`
	MD5       NullString `json:"md5"`
	SHA256    NullString `json:"sha256"`
	CreatedAt Time       `json:"created_at"`
	DeletedAt NullTime   `json:"deleted_at"`
	URL       URL        `json:"url"`
}

type BuildTrigger struct {
	Type    string            `json:"type"`
	Comment string            `json:"comment"`
	Data    map[string]string `json:"data"`
}

type BuildTag struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	BuildID   int64  `json:"build_id"`
	Name      string `json:"name"`
	CreatedAt Time   `json:"created_at"`
	URL       URL    `json:"url"`
}

type Status uint8

//go:generate stringer -type Status -linecomment
const (
	Queued             Status = iota // queued
	Running                          // running
	Passed                           // passed
	PassedWithFailures               // passed_with_failures
	Failed                           // failed
	Killed                           // killed
	TimedOut                         // timed_out
)

func (s *Status) UnmarshalJSON(p []byte) error {
	var str string

	if err := json.Unmarshal(p, &str); err != nil {
		return err
	}

	switch str {
	case "queued":
		(*s) = Queued
	case "running":
		(*s) = Running
	case "passed":
		(*s) = Passed
	case "passed_with_failures":
		(*s) = PassedWithFailures
	case "failed":
		(*s) = Failed
	case "killed":
		(*s) = Killed
	case "timed_out":
		(*s) = TimedOut
	default:
		return errors.New("unknown build status " + str)
	}
	return nil
}

type ManifestPassthrough map[string]string

func (m ManifestPassthrough) MarshalYAML() (interface{}, error) {
	if m == nil {
		return []string{}, nil
	}

	ss := make([]string, 0, len(m))

	for k, v := range m {
		if v == "" {
			v = filepath.Base(k)
		}
		ss = append(ss, k+" => "+v)
	}
	return ss, nil
}

func (m *ManifestPassthrough) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if (*m) == nil {
		(*m) = make(map[string]string)
	}

	ss := make([]string, 0)

	if err := unmarshal(&ss); err != nil {
		return err
	}

	for _, s := range ss {
		parts := strings.Split(s, "=>")

		k := strings.TrimSpace(parts[0])
		v := filepath.Base(k)

		if len(parts) > 1 {
			v = strings.TrimSpace(parts[1])
		}
		(*m)[k] = v
	}
	return nil
}

type Manifest struct {
	Namespace     string
	Driver        map[string]string
	Env           []string
	Objects       ManifestPassthrough
	Sources       []ManifestSource
	Stages        []string
	AllowFailures []string `yaml:"allow_failures"`
	Jobs          []ManifestJob
}

type ManifestSource struct {
	URL string
	Ref string
	Dir string
}

func (s ManifestSource) MarshalYAML() (interface{}, error) {
	source := s.URL

	if s.Ref != "" {
		source += " " + s.Ref
	}

	if s.Dir != "" {
		source += " => " + s.Dir
	}
	return source, nil
}

func base(s string) string {
	parts := strings.Split(s, "/")
	return parts[len(parts)-1]
}

func (s *ManifestSource) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var str string

	if err := unmarshal(&str); err != nil {
		return err
	}

	i := strings.Index(str, " ")

	if i < 0 {
		i = len(str)
	}

	s.URL = str[:i]
	s.Dir = strings.TrimSuffix(base(s.URL), ".git")

	tmp := make([]rune, 0, len(str[i:]))

	for _, r := range str[i:] {
		if r == ' ' {
			continue
		}
		tmp = append(tmp, r)
	}

	str = string(tmp)

	i = strings.Index(str, "=>")

	if i >= 0 {
		s.Dir = str[i+2:]
		str = str[:i]
	}
	if len(str) > 0 {
		s.Ref = str
	}
	return nil
}

type ManifestJob struct {
	Stage     string
	Name      string
	Commands  []string
	Artifacts ManifestPassthrough
}

func (m Manifest) MarshalJSON() ([]byte, error) {
	b, err := yaml.Marshal(m)

	if err != nil {
		return nil, err
	}
	return json.Marshal(string(b))
}

func (m *Manifest) UnmarshalJSON(p []byte) error {
	var s string

	if err := json.Unmarshal(p, &s); err != nil {
		return err
	}
	return yaml.Unmarshal([]byte(s), m)
}

type BuildParams struct {
	Manifest Manifest `json:"manifest"`
	Comment  string   `json:"comment"`
	Tags     []string `json:"tags"`
}

func SubmitBuild(cli *Client, p BuildParams) (*Build, error) {
	var body bytes.Buffer

	if err := json.NewEncoder(&body).Encode(p); err != nil {
		return nil, err
	}

	resp, err := cli.Post("/builds", "application/json; charset=utf-8", &body)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, cli.err(resp)
	}

	var b Build

	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		return nil, err
	}
	return &b, nil
}

func GetBuild(cli *Client, owner string, number int64) (*Build, error) {
	resp, err := cli.Get("/b/"+owner+"/"+strconv.FormatInt(number, 10), "application/json; charset=utf-8")

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, cli.err(resp)
	}

	var b Build

	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		return nil, err
	}
	return &b, nil
}

func (b *Build) Get(cli *Client) error {
	resp, err := cli.Get(b.URL.Path, "application/json; charset=utf-8")

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cli.err(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(b); err != nil {
		return err
	}
	return nil
}

func (b *Build) GetJobs(cli *Client) ([]*BuildJob, error) {
	resp, err := cli.Get(b.JobsURL.Path, "application/json; charset=utf-8")

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, cli.err(resp)
	}

	jj := make([]*BuildJob, 0)

	if err := json.NewDecoder(resp.Body).Decode(&jj); err != nil {
		return nil, err
	}
	return jj, nil
}

func (b *Build) Tag(cli *Client, tags ...string) ([]*BuildTag, error) {
	var body bytes.Buffer

	if err := json.NewEncoder(&body).Encode(tags); err != nil {
		return nil, err
	}

	resp, err := cli.Post(b.TagsURL.Path, "application/json; charset=utf-8", &body)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, cli.err(resp)
	}

	tt := make([]*BuildTag, 0, len(tags))

	if err := json.NewDecoder(resp.Body).Decode(&tt); err != nil {
		return nil, err
	}
	return tt, nil
}

func (b *Build) Kill(cli *Client) error {
	if b.Status != Running {
		return nil
	}

	resp, err := cli.Delete(b.URL.Path)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return cli.err(resp)
	}
	return nil
}

func (t *BuildTag) Delete(cli *Client) error {
	resp, err := cli.Delete(t.URL.Path)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return cli.err(resp)
	}
	return nil
}
