package djinn

import (
	"encoding/json"
	"net/url"
	"strconv"
	"time"
)

type NullInt64 struct {
	Int64 int64
	Valid bool
}

func (n *NullInt64) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(n.Int64, 10)), nil
}

func (n *NullInt64) UnmarshalJSON(p []byte) error {
	if string(p) == "null" {
		n.Int64, n.Valid = 0, false
		return nil
	}

	n.Valid = true
	return json.Unmarshal(p, &n.Int64)
}

type NullString struct {
	String string
	Valid  bool
}

func (n *NullString) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return []byte(n.String), nil
}

func (n *NullString) UnmarshalJSON(p []byte) error {
	if string(p) == "null" {
		n.String, n.Valid = "", false
		return nil
	}

	n.Valid = true
	return json.Unmarshal(p, &n.String)
}

type Duration struct {
	time.Duration
}

func (d *Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration)
}

func (d *Duration) UnmarshalJSON(p []byte) error {
	var i int64

	if err := json.Unmarshal(p, &i); err != nil {
		return err
	}

	d.Duration = time.Duration(i)
	return nil
}

type Time struct {
	time.Time
}

func (t *Time) MarshalJSON() ([]byte, error) {
	return []byte(t.Format(time.RFC3339)), nil
}

func (t *Time) UnmarshalJSON(p []byte) error {
	var s string

	if err := json.Unmarshal(p, &s); err != nil {
		return err
	}

	tmp, err := time.Parse(time.RFC3339, s)

	if err != nil {
		return err
	}

	t.Time = tmp
	return nil
}

type NullTime struct {
	Time  Time
	Valid bool
}

func (n *NullTime) MarshalJSON() ([]byte, error) {
	if !n.Valid {
		return []byte("null"), nil
	}
	return n.Time.MarshalJSON()
}

func (n *NullTime) UnmarshalJSON(p []byte) error {
	if string(p) == "null" {
		n.Valid = false
		return nil
	}
	return n.Time.UnmarshalJSON(p)
}

type URL struct {
	*url.URL
}

func (u *URL) MarshalJSON() ([]byte, error) {
	return []byte(u.String()), nil
}

func (u *URL) UnmarshalJSON(p []byte) error {
	var s string

	if err := json.Unmarshal(p, &s); err != nil {
		return err
	}

	url, err := url.Parse(s)

	if err != nil {
		return err
	}

	u.URL = url
	return nil
}
