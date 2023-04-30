package build

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
)

type TriggerType uint8

//go:generate stringer -type TriggerType -linecomment
const (
	Manual   TriggerType = iota // manual
	Push                        // push
	Pull                        // pull
	Schedule                    // schedule
)

var (
	_ sql.Scanner   = (*TriggerType)(nil)
	_ driver.Valuer = (*TriggerType)(nil)

	triggersMap = map[string]TriggerType{
		"manual":   Manual,
		"push":     Push,
		"pull":     Pull,
		"schedule": Schedule,
	}
)

func (t *TriggerType) Scan(val any) error {
	v, err := driver.String.ConvertValue(val)

	if err != nil {
		return errors.Err(err)
	}

	s, ok := v.(string)

	if !ok {
		return errors.New("build: could not type assert Trigger to string")
	}

	if err := t.UnmarshalText([]byte(s)); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (t *TriggerType) UnmarshalText(b []byte) error {
	var ok bool

	s := string(b)
	(*t), ok = triggersMap[s]

	if !ok {
		return errors.New("build: unknown trigger " + s)
	}
	return nil
}

func (t TriggerType) Value() (driver.Value, error) { return driver.Value(t.String()), nil }

type triggerData map[string]string

var (
	_ sql.Scanner   = (*triggerData)(nil)
	_ driver.Valuer = (*triggerData)(nil)
)

func NewTriggerData() triggerData { return triggerData(make(map[string]string)) }

func (d *triggerData) Scan(val any) error {
	v, err := driver.String.ConvertValue(val)

	if err != nil {
		return errors.Err(err)
	}

	b, ok := v.([]byte)

	if !ok {
		return errors.New("build: could not type assert triggerData to byte slice")
	}

	if len(b) == 0 {
		return nil
	}

	if err := json.Unmarshal(b, d); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (d *triggerData) Set(key, val string) {
	if (*d) == nil {
		(*d) = make(map[string]string)
	}
	(*d)[key] = val
}

func (d *triggerData) String() string {
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(d)
	return buf.String()[:buf.Len()-1]
}

func (d triggerData) Value() (driver.Value, error) { return driver.Value(d.String()), nil }

type Trigger struct {
	ID         int64
	BuildID    int64
	ProviderID database.Null[int64]
	RepoID     database.Null[int64]
	Type       TriggerType
	Comment    string
	Data       triggerData
	CreatedAt  time.Time

	Build *Build
}

func (t *Trigger) Primary() (string, any) { return "id", t.ID }

func (t *Trigger) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":          &t.ID,
		"build_id":    &t.BuildID,
		"provider_id": &t.ProviderID,
		"repo_id":     &t.RepoID,
		"type":        &t.Type,
		"comment":     &t.Comment,
		"data":        &t.Data,
		"created_at":  &t.CreatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (t *Trigger) Params() database.Params {
	return database.Params{
		"id":          database.ImmutableParam(t.ID),
		"build_id":    database.CreateOnlyParam(t.BuildID),
		"provider_id": database.CreateOnlyParam(t.ProviderID),
		"repo_id":     database.CreateOnlyParam(t.RepoID),
		"type":        database.CreateOnlyParam(t.Type),
		"comment":     database.CreateOnlyParam(t.Comment),
		"data":        database.CreateOnlyParam(t.Data),
		"created_at":  database.CreateOnlyParam(t.CreatedAt),
	}
}

var _ database.Model = (*Trigger)(nil)

func (t *Trigger) Bind(m database.Model) {
	if b, ok := m.(*Build); ok {
		if t.BuildID == b.ID {
			t.Build = b
		}
	}
}

func (t *Trigger) MarshalJSON() ([]byte, error) {
	if t == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"type":    t.Type.String(),
		"comment": t.Comment,
		"data":    t.Data,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (*Trigger) Endpoint(...string) string { return "" }

func (t *Trigger) CommentBody() string {
	i := strings.Index(t.Comment, "\n")
	wrap := false

	if i == -1 {
		if len(t.Comment) <= 72 {
			return ""
		}
		i = 72
		wrap = true
	}

	body := strings.TrimSpace(t.Comment[i:])

	if i > 72 || wrap {
		return "..." + body
	}
	return body
}

func (t *Trigger) CommentTitle() string {
	i := strings.Index(t.Comment, "\n")
	wrap := false

	if i == -1 {
		if len(t.Comment) <= 72 {
			return t.Comment
		}
		i = 72
		wrap = true
	}

	title := strings.TrimSpace(t.Comment[:i])

	if i > 72 || wrap {
		return title + "..."
	}
	return title
}

func (t *Trigger) String() string {
	var (
		buf bytes.Buffer

		username string
		email    string
	)

	if t.Build != nil && t.Build.User != nil {
		username = t.Build.User.Username
		email = t.Build.User.Email
	}

	switch t.Type {
	case Manual:
		buf.WriteString("Submitted by " + username + " <" + email + ">\n")
	case Push:
		buf.WriteString("Committed " + t.Data["sha"][:7] + " to " + t.Data["ref"] + "\n")
	case Pull:
		buf.WriteString(strings.Title(t.Data["action"]) + " pull request to " + t.Data["ref"] + "\n")
	}

	if t.Comment != "" {
		buf.WriteString("\n" + t.CommentTitle() + "\n\n")
		buf.WriteString(t.CommentBody() + "\n")
	}
	return buf.String()
}

const triggerTable = "build_triggers"

func NewTriggerStore(pool *database.Pool) *database.Store[*Trigger] {
	return database.NewStore[*Trigger](pool, triggerTable, func() *Trigger {
		return &Trigger{}
	})
}
