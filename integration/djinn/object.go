package djinn

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

type Object struct {
	ID          int64     `json:"id"`
	AuthorID    int64     `json:"author_id"`
	UserID      int64     `json:"user_id"`
	NamespaceID NullInt64 `json:"namespace_id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Size        int64     `json:"size"`
	MD5         string    `json:"md5"`
	SHA256      string    `json:"sha256"`
	CreatedAt   Time      `json:"created_at"`
	URL         URL       `json:"url"`
}

type ObjectParams struct {
	Namespace string    `json:"-"`
	Object    io.Reader `json:"-"`
	Name      string    `json:"-"`
}

func CreateObject(cli *Client, p ObjectParams) (*Object, error) {
	vals := url.Values(make(map[string][]string))

	vals.Set("namespace", p.Namespace)
	vals.Set("name", p.Name)

	resp, err := cli.Post("/objects?"+vals.Encode(), "application/octet-stream", p.Object)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, cli.err(resp)
	}

	var o Object

	if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (o *Object) Data(cli *Client) (io.ReadCloser, error) {
	resp, err := cli.Get(o.URL.Path, o.Type)

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()

		return nil, cli.err(resp)
	}
	return resp.Body, nil
}

func (o *Object) Delete(cli *Client) error {
	resp, err := cli.Delete(o.URL.Path)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return cli.err(resp)
	}
	return nil
}
