package build

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type triggerType uint8

type triggerData map[string]string

type Trigger struct {
	ID        int64       `db:"id"`
	BuildID   int64       `db:"build_id"`
	Type      triggerType `db:"type"`
	Comment   string      `db:"comment"`
	Data      triggerData `db:"data"`
	CreatedAt time.Time   `db:"created_at"`

	Build *Build `db:"-"`
}

type TriggerStore struct {
	model.Store

	Build *Build
}

//go:generate stringer -type triggerType -linecomment
const (
	Manual triggerType = iota // manual
	Push                      // push
	Pull                      // pull
)

var (
	_ model.Model  = (*Trigger)(nil)
	_ model.Binder = (*TriggerStore)(nil)
	_ model.Loader = (*TriggerStore)(nil)

	_ sql.Scanner   = (*triggerData)(nil)
	_ driver.Valuer = (*triggerData)(nil)

	_ sql.Scanner   = (*triggerType)(nil)
	_ driver.Valuer = (*triggerType)(nil)

	triggerTable = "build_triggers"
	triggersMap  = map[string]triggerType{
		"manual": Manual,
		"push":   Push,
		"pull":   Pull,
	}
)

func NewTriggerStore(db *sqlx.DB, mm ...model.Model) TriggerStore {
	s := TriggerStore{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func NewTriggerData() triggerData {
	return triggerData(make(map[string]string))
}

func TriggerModel(tt []*Trigger) func(int) model.Model {
	return func(i int) model.Model {
		return tt[i]
	}
}

func (t *triggerType) Scan(val interface{}) error {
	b, err := model.Scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		(*t) = triggerType(0)
		return nil
	}
	return errors.Err(t.UnmarshalText(b))
}

func (t *triggerType) UnmarshalText(b []byte) error {
	var ok bool

	s := string(b)
	(*t), ok = triggersMap[s]

	if !ok {
		return errors.New("unknown trigger "+s)
	}
	return nil
}

func (t triggerType) Value() (driver.Value, error) {
	return driver.Value(t.String()), nil
}

func (d *triggerData) Scan(val interface{}) error {
	b, err := model.Scan(val)

	if err != nil {
		return errors.Err(err)
	}

	if len(b) == 0 {
		return nil
	}

	buf := bytes.NewBuffer(b)
	dec := json.NewDecoder(buf)
	return errors.Err(dec.Decode(d))
}

func (d *triggerData) Set(key, val string) {
	if (*d) == nil {
		(*d) = make(map[string]string)
	}
	(*d)[key] = val
}

func (d *triggerData) String() string {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.Encode(d)
	return buf.String()
}

func (d triggerData) Value() (driver.Value, error) {
	return driver.Value(d.String()), nil
}

func (t *Trigger) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			t.Build = m.(*Build)
			break
		}
	}
}

func (*Trigger) Kind() string { return "build_trigger" }

func (t *Trigger) SetPrimary(i int64) {
	if t == nil {
		return
	}
	t.ID = i
}

func (t *Trigger) Primary() (string, int64) {
	if t == nil {
		return "id", 0
	}
	return "id", t.ID
}

func (t *Trigger) IsZero() bool {
	return t == nil || t.ID == 0 &&
		t.BuildID == 0 &&
		t.Type == triggerType(0) &&
		t.Comment == "" &&
		len(t.Data) == 0 &&
		t.CreatedAt == time.Time{}
}

func (*Trigger) Endpoint(uri ...string) string { return "" }

func (t *Trigger) Values() map[string]interface{} {
	if t == nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"build_id": t.BuildID,
		"type":     t.Type,
		"comment":  t.Comment,
		"data":     t.Data,
	}
}

func (t Trigger) CommentBody() string {
	i := strings.Index(t.Comment, "\n")

	if i == -1 {
		if len(t.Comment) <= 72 {
			return ""
		}

		i = 72
	}

	body := strings.TrimSpace(t.Comment[i:])

	if strings.TrimSpace(t.Comment[:i]) != "" && body != "" {
		return "..."+body
	}
	return body
}

func (t Trigger) CommentTitle() string {
	i := strings.Index(t.Comment, "\n")

	if i == -1 {
		if len(t.Comment) <= 72 {
			return t.Comment
		}

		i = 72
	}

	title := strings.TrimSpace(t.Comment[:i])

	if strings.TrimSpace(t.Comment[i:]) != "" {
		return title+"..."
	}
	return title
}

func (s *TriggerStore) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
			break
		}
	}
}

func (s TriggerStore) Create(tt ...*Trigger) error {
	models := model.Slice(len(tt), TriggerModel(tt))
	return errors.Err(s.Store.Create(triggerTable, models...))
}

func (s TriggerStore) All(opts ...query.Option) ([]*Trigger, error) {
	tt := make([]*Trigger, 0)

	opts = append([]query.Option{
		model.Where(s.Build, "build_id"),
	}, opts...)

	err := s.Store.All(&tt, triggerTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, t := range tt {
		t.Build = s.Build
	}
	return tt, errors.Err(err)
}

func (s TriggerStore) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	tt, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, t := range tt {
			load(i, t)
		}
	}
	return nil
}
