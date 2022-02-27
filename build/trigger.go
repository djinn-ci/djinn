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

	"github.com/andrewpillar/query"
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

func (t *TriggerType) Scan(val interface{}) error {
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
		return errors.New("unknown trigger " + s)
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

func (d *triggerData) Scan(val interface{}) error {
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
	ProviderID sql.NullInt64
	RepoID     sql.NullInt64
	Type       TriggerType
	Comment    string
	Data       triggerData
	CreatedAt  time.Time

	Build *Build
}

var _ database.Model = (*Trigger)(nil)

func (t *Trigger) Dest() []interface{} {
	return []interface{}{
		&t.ID,
		&t.BuildID,
		&t.ProviderID,
		&t.RepoID,
		&t.Type,
		&t.Comment,
		&t.Data,
		&t.CreatedAt,
	}
}

func (t *Trigger) Bind(m database.Model) {
	if b, ok := m.(*Build); ok {
		if t.BuildID == b.ID {
			t.Build = b
		}
	}
}

func (t *Trigger) JSON(_ string) map[string]interface{} {
	if t == nil {
		return nil
	}

	return map[string]interface{}{
		"type":    t.Type.String(),
		"comment": t.Comment,
		"data":    t.Data,
	}
}

func (*Trigger) Endpoint(_ ...string) string { return "" }

func (t Trigger) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":          t.ID,
		"build_id":    t.BuildID,
		"provider_id": t.ProviderID,
		"repo_id":     t.RepoID,
		"type":        t.Type,
		"comment":     t.Comment,
		"data":        t.Data,
		"created_at":  t.CreatedAt,
	}
}

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
	buf := bytes.Buffer{}

	var username, email string

	if t.Build != nil && t.Build.User != nil {
		username = t.Build.User.Username
		email = t.Build.User.Email
	}

	switch t.Type {
	case Manual:
		buf.WriteString("Submitted by " + username + "<" + email + ">\n")
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

type TriggerStore struct {
	database.Pool
}

var (
	_ database.Loader = (*TriggerStore)(nil)

	triggerTable = "build_triggers"
)

func (s TriggerStore) All(opts ...query.Option) ([]*Trigger, error) {
	tt := make([]*Trigger, 0)

	new := func() database.Model {
		t := &Trigger{}
		tt = append(tt, t)
		return t
	}

	if err := s.Pool.All(triggerTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return tt, nil
}

func (s TriggerStore) Get(opts ...query.Option) (*Trigger, bool, error) {
	var t Trigger

	ok, err := s.Pool.Get(triggerTable, &t, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &t, ok, nil
}

func (s TriggerStore) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	tt, err := s.All(query.Where(pk, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	for _, t := range tt {
		for _, m := range mm {
			m.Bind(t)
		}
	}
	return nil
}
