package djinn

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
)

type Key struct {
	ID          int64  `json:"id"`
	AuthorID    int64  `json:"author_id"`
	UserID      int64  `json:"user_id"`
	NamespaceID NullInt64  `json:"namespace_id"`
	Name        string `json:"name"`
	Config      string `json:"config"`
	CreatedAt   Time   `json:"created_at"`
	UpdatedAt   Time   `json:"updated_at"`
	URL         URL    `json:"url"`
}

type KeyParams struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Key       string `json:"key"`
	Config    string `json:"config"`
}

func CreateKey(cli *Client, p KeyParams) (*Key, error) {
	var body bytes.Buffer

	if err := json.NewEncoder(&body).Encode(p); err != nil {
		return nil, err
	}

	resp, err := cli.Post("/keys", "application/json; charset=utf-8", &body)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, cli.err(resp)
	}

	var k Key

	if err := json.NewDecoder(resp.Body).Decode(&k); err != nil {
		return nil, err
	}
	return &k, nil
}

func GetKey(cli *Client, id int64) (*Key, error) {
	resp, err := cli.Get("/keys/"+strconv.FormatInt(id, 10), "application/json; charset=utf-8")

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, cli.err(resp)
	}

	var k Key

	if err := json.NewDecoder(resp.Body).Decode(&k); err != nil {
		return nil, err
	}
	return &k, nil
}

func (k *Key) Get(cli *Client) error {
	k2, err := GetKey(cli, k.ID)

	if err != nil {
		return err
	}

	(*k) = (*k2)
	return nil
}

func (k *Key) Update(cli *Client, p KeyParams) error {
	var body bytes.Buffer

	p.Namespace = ""
	p.Name = ""
	p.Key = ""

	if err := json.NewEncoder(&body).Encode(p); err != nil {
		return err
	}

	resp, err := cli.Patch(k.URL.Path, "application/json; charset=utf-8", &body)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cli.err(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(k); err != nil {
		return err
	}
	return nil
}

func (k *Key) Delete(cli *Client) error {
	resp, err := cli.Delete(k.URL.Path)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return cli.err(resp)
	}
	return nil
}
