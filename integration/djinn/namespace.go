package djinn

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

type Namespace struct {
	ID               int64      `json:"id"`
	UserID           int64      `json:"user_id"`
	RootID           int64      `json:"root_id"`
	ParentID         NullInt64  `json:"parent_id"`
	Name             string     `json:"name"`
	Path             string     `json:"path"`
	Description      string     `json:"description"`
	Visibility       Visibility `json:"visibility"`
	CreatedAt        Time       `json:"created_at"`
	URL              URL        `json:"url"`
	BuildsURL        URL        `json:"builds_url"`
	NamespacesURL    URL        `json:"namespaces_url"`
	ImagesURL        URL        `json:"images_url"`
	ObjectsURL       URL        `json:"objects_url"`
	VariablesURL     URL        `json:"variables_url"`
	KeysURL          URL        `json:"keys_url"`
	InvitesURL       URL        `json:"invites_url"`
	CollaboratorsURL URL        `json:"collaborators_url"`
	WebhooksURL      URL        `json:"webhooks_url"`
	User             *User      `json:"user"`
	Parent           *Namespace `json:"parent"`
	Build            *Build     `json:"build"`
}

type NamespaceParams struct {
	Parent      string     `json:"parent"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Visibility  Visibility `json:"visibility"`
}

type Webhook struct {
	ID           int64            `json:"id"`
	AuthorID     int64            `json:"author_id"`
	UserID       int64            `json:"user_id"`
	NamespaceID  NullInt64        `json:"namespace_id"`
	PayloadURL   URL              `json:"payload_url"`
	SSL          bool             `json:"ssl"`
	Active       bool             `json:"active"`
	URL          URL              `json:"url"`
	Events       []string         `json:"events"`
	LastResponse *WebhookResponse `json:"last_response"`
}

type WebhookResponse struct {
	Error     NullString `json:"error"`
	Code      int        `json:"code"`
	Duration  Duration   `json:"duration"`
	CreatedAt Time       `json:"created_at"`
}

type WebhookParams struct {
	PayloadURL   string   `json:"payload_url"`
	Secret       string   `json:"secret"`
	RemoveSecret bool     `json:"remove_secret"`
	SSL          bool     `json:"ssl"`
	Active       bool     `json:"active"`
	Events       []string `json:"events"`
}

var WebhookEvents = []string{
	"build.submitted",
	"invite.sent",
	"invite.accepted",
	"invite.rejected",
	"namespaces",
	"cron",
	"images",
	"objects",
	"variables",
	"ssh_keys",
}

type Visibility uint8

//go:generate stringer -type Visibility -linecomment
const (
	Private  Visibility = iota // private
	Internal                   // internal
	Public                     // public
)

func (v Visibility) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.String())
}

func (v *Visibility) UnmarshalJSON(p []byte) error {
	var s string

	if err := json.Unmarshal(p, &s); err != nil {
		return err
	}

	switch s {
	case "private":
		(*v) = Private
	case "internal":
		(*v) = Internal
	case "public":
		(*v) = Public
	default:
		return errors.New("unknown namespace visibility " + s)
	}
	return nil
}

func CreateNamespace(cli *Client, p NamespaceParams) (*Namespace, error) {
	var body bytes.Buffer

	if err := json.NewEncoder(&body).Encode(p); err != nil {
		return nil, err
	}

	resp, err := cli.Post("/namespaces", "application/json; charset=utf-8", &body)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, cli.err(resp)
	}

	var n Namespace

	if err := json.NewDecoder(resp.Body).Decode(&n); err != nil {
		return nil, err
	}
	return &n, nil
}

func GetNamespace(cli *Client, owner, path string) (*Namespace, error) {
	resp, err := cli.Get("/n/"+owner+"/"+path, "application/json; charset=utf-8")

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, cli.err(resp)
	}

	var n Namespace

	if err := json.NewDecoder(resp.Body).Decode(&n); err != nil {
		return nil, err
	}
	return &n, nil
}

func (n *Namespace) Get(cli *Client) error {
	n2, err := GetNamespace(cli, n.User.Username, n.Path)

	if err != nil {
		return err
	}

	(*n) = (*n2)
	return nil
}

func (n *Namespace) CreateWebhook(cli *Client, p WebhookParams) (*Webhook, error) {
	var body bytes.Buffer

	tmp := struct {
		PayloadURL string   `json:"payload_url"`
		Secret     string   `json:"secret"`
		SSL        bool     `json:"ssl"`
		Active     bool     `json:"active"`
		Events     []string `json:"events"`
	}{
		PayloadURL: p.PayloadURL,
		Secret:     p.Secret,
		SSL:        p.SSL,
		Active:     p.Active,
		Events:     p.Events,
	}

	if err := json.NewEncoder(&body).Encode(tmp); err != nil {
		return nil, err
	}

	resp, err := cli.Post(n.WebhooksURL.Path, "application/json; charset=utf-8", &body)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, cli.err(resp)
	}

	var wh Webhook

	if err := json.NewDecoder(resp.Body).Decode(&wh); err != nil {
		return nil, err
	}
	return &wh, nil
}

func (n *Namespace) Invite(cli *Client, handle string) (*Invite, error) {
	var body bytes.Buffer

	json.NewEncoder(&body).Encode(map[string]string{"handle": handle})

	resp, err := cli.Post(n.InvitesURL.Path, "application/json; charset=utf-8", &body)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, cli.err(resp)
	}

	var i Invite

	if err := json.NewDecoder(resp.Body).Decode(&i); err != nil {
		return nil, err
	}
	return &i, nil
}

func (n *Namespace) DeleteCollaborator(cli *Client, username string) error {
	resp, err := cli.Delete(n.CollaboratorsURL.Path+"/"+username)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return cli.err(resp)
	}
	return nil
}

func (n *Namespace) Update(cli *Client, p NamespaceParams) error {
	var body bytes.Buffer

	p.Parent = ""
	p.Name = ""

	if err := json.NewEncoder(&body).Encode(p); err != nil {
		return err
	}

	resp, err := cli.Patch(n.URL.Path, "application/json; charset=utf-8", &body)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cli.err(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(n); err != nil {
		return err
	}
	return nil
}

func (n *Namespace) Delete(cli *Client) error {
	resp, err := cli.Delete(n.URL.Path)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return cli.err(resp)
	}
	return nil
}
