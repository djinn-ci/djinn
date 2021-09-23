package djinn

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type Image struct {
	ID          int64          `json:"id"`
	AuthorID    int64          `json:"author_id"`
	UserID      int64          `json:"user_id"`
	NamespaceID NullInt64      `json:"namespace_id"`
	Driver      Driver         `json:"driver"`
	Name        string         `json:"name"`
	CreatedAt   Time           `json:"created_at"`
	URL         URL            `json:"url"`
	Download    *ImageDownload `json:"download"`
}

type ImageDownload struct {
	Source     URL        `json:"source"`
	Error      NullString `json:"error"`
	CreatedAt  Time       `json:"created_at"`
	StartedAt  NullTime   `json:"started_at"`
	FinishedAt NullTime   `json:"finished_at"`
}

type ImageParams struct {
	Namespace   string    `json:"namespace"`
	Image       io.Reader `json:"-"`
	Name        string    `json:"name"`
	DownloadURL string    `json:"download_url"`
}

type Driver uint8

//go:generate stringer -type Driver -linecomment
const (
	DriverSSH    Driver = iota // ssh
	DriverQEMU                 // qemu
	DriverDocker               // docker
)

func (d *Driver) MarshalJSON() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *Driver) UnmarshalJSON(p []byte) error {
	var s string

	if err := json.Unmarshal(p, &s); err != nil {
		return err
	}

	switch s {
	case "ssh":
		(*d) = DriverSSH
	case "qemu":
		(*d) = DriverQEMU
	case "docker":
		(*d) = DriverDocker
	default:
		return errors.New("unknown image driver " + s)
	}
	return nil
}

func CreateImage(cli *Client, p ImageParams) (*Image, error) {
	endpoint := "/images"

	var (
		contentType string
		body        io.Reader
	)

	if p.Image != nil {
		vals := url.Values(make(map[string][]string))
		vals.Set("namespace", p.Namespace)
		vals.Set("name", p.Name)

		endpoint += "?" + vals.Encode()
		contentType = "application/x-qemu-disk"
		body = p.Image
	} else {
		var buf bytes.Buffer

		if err := json.NewEncoder(&buf).Encode(p); err != nil {
			return nil, err
		}

		contentType = "application/json; charset=utf-8"
		body = &buf
	}

	resp, err := cli.Post(endpoint, contentType, body)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, cli.err(resp)
	}

	var i Image

	if err := json.NewDecoder(resp.Body).Decode(&i); err != nil {
		return nil, err
	}
	return &i, nil
}

func GetImage(cli *Client, id int64) (*Image, error) {
	resp, err := cli.Get("/images/"+strconv.FormatInt(id, 10), "application/json; charset=utf-8")

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, cli.err(resp)
	}

	var i Image

	if err := json.NewDecoder(resp.Body).Decode(&i); err != nil {
		return nil, err
	}
	return &i, nil
}

func (i *Image) Get(cli *Client) error {
	i2, err := GetImage(cli, i.ID)

	if err != nil {
		return err
	}

	(*i) = (*i2)
	return nil
}

func (i *Image) Data(cli *Client) (io.ReadCloser, error) {
	resp, err := cli.Get(i.URL.Path, "application/x-qemu-disk")

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()

		return nil, cli.err(resp)
	}
	return resp.Body, nil
}

func (i *Image) Delete(cli *Client) error {
	resp, err := cli.Delete(i.URL.Path)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return cli.err(resp)
	}
	return nil
}
