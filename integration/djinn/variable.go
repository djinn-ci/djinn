package djinn

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Variable struct {
	ID          int64      `json:"id"`
	AuthorID    int64      `json:"author_id"`
	UserID      int64      `json:"user_id"`
	NamespaceID NullInt64  `json:"namespace_id"`
	Key         string     `json:"key"`
	Value       string     `json:"value"`
	CreatedAt   Time       `json:"created_at"`
	URL         URL        `json:"url"`
	Namespace   *Namespace `json:"namespace"`
}

type VariableParams struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Mask      bool   `json:"mask"`
}

func CreateVariable(cli *Client, p VariableParams) (*Variable, error) {
	var body bytes.Buffer

	if err := json.NewEncoder(&body).Encode(p); err != nil {
		return nil, err
	}

	resp, err := cli.Post("/variables", "application/json; charset=utf-8", &body)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, cli.err(resp)
	}

	var v Variable

	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

func (v *Variable) Get(cli *Client) error {
	resp, err := cli.Get(v.URL.Path, "application/json; charset=utf-8")

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cli.err(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return err
	}
	return nil
}

func (v *Variable) Delete(cli *Client) error {
	resp, err := cli.Delete(v.URL.Path)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return cli.err(resp)
	}
	return nil
}
