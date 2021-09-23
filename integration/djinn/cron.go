package djinn

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

type Cron struct {
	ID          int64     `json:"id"`
	AuthorID    int64     `json:"author_id"`
	UserID      int64     `json:"user_id"`
	NamespaceID NullInt64 `json:"namespace_id"`
	Name        string    `json:"name"`
	Schedule    Schedule  `json:"schedule"`
	Manifest    Manifest  `json:"manifest"`
	NextRun     Time      `json:"next_run"`
	CreatedAt   Time      `json:"created_at"`
	URL         URL       `json:"url"`
}

type CronParams struct {
	Name     string   `json:"name"`
	Schedule Schedule `json:"schedule"`
	Manifest Manifest `json:"manifest"`
}

type Schedule uint

//go:generate stringer -type Schedule -linecomment
const (
	Daily   Schedule = iota // daily
	Weekly                  // weekly
	Monthly                 // monthly
)

func (s Schedule) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *Schedule) UnmarshalJSON(p []byte) error {
	var str string

	if err := json.Unmarshal(p, &str); err != nil {
		return err
	}

	switch str {
	case "daily":
		(*s) = Daily
	case "weekly":
		(*s) = Weekly
	case "monthly":
		(*s) = Monthly
	default:
		return errors.New("unknown schedule " + str)
	}
	return nil
}

func CreateCron(cli *Client, p CronParams) (*Cron, error) {
	var body bytes.Buffer

	if err := json.NewEncoder(&body).Encode(p); err != nil {
		return nil, err
	}

	resp, err := cli.Post("/cron", "application/json; charset=utf-8", &body)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, cli.err(resp)
	}

	var c Cron

	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Cron) Get(cli *Client) error {
	resp, err := cli.Get(c.URL.Path, "application/json; charset=utf-8")

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cli.err(resp)
	}

	var c2 Cron

	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return err
	}

	(*c) = c2
	return nil
}

func (c *Cron) Update(cli *Client, p CronParams) error {
	var body bytes.Buffer

	if err := json.NewEncoder(&body).Encode(p); err != nil {
		return err
	}

	resp, err := cli.Patch(c.URL.Path, "application/json; charset=utf-8", &body)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cli.err(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(c); err != nil {
		return err
	}
	return nil
}

func (c *Cron) Delete(cli *Client) error {
	resp, err := cli.Delete(c.URL.Path)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return cli.err(resp)
	}
	return nil
}
