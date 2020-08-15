package build

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type triggerType uint8

type triggerData map[string]string

// Trigger is the type that represents what triggered a build.
type Trigger struct {
	ID         int64         `db:"id"`
	BuildID    int64         `db:"build_id"`
	ProviderID sql.NullInt64 `db:"provider_id"`
	Type       triggerType   `db:"type"`
	Comment    string        `db:"comment"`
	Data       triggerData   `db:"data"`
	CreatedAt  time.Time     `db:"created_at"`

	Build *Build `db:"-"`
}

// TriggerStore is the type for creating and modifying Trigger models in the
// database.
type TriggerStore struct {
	database.Store

	Build *Build
}

//go:generate stringer -type triggerType -linecomment
const (
	// There are three different trigger types for a build trigger,
	// Manual - for when a build was manually submitted for either via the API
	// or UI.
	// Push - for when a build was triggered via a commit hook.
	// Pull - for when a build was triggered via a pull-request hook.
	Manual triggerType = iota // manual
	Push                      // push
	Pull                      // pull
)

var (
	_ database.Model  = (*Trigger)(nil)
	_ database.Binder = (*TriggerStore)(nil)
	_ database.Loader = (*TriggerStore)(nil)

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

// NewTriggerStore returns a new TriggerStore for querying the build_triggers
// table. Each database passed to this function will be bound to the returned
// TriggerStore.
func NewTriggerStore(db *sqlx.DB, mm ...database.Model) *TriggerStore {
	s := &TriggerStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

// NewTriggerData returns an empty set of data for a build trigger.
func NewTriggerData() triggerData { return triggerData(make(map[string]string)) }

// TriggerModel is called along with database.ModelSlice to convert the given slice of
// Trigger models to a slice of database.Model interfaces.
func TriggerModel(tt []*Trigger) func(int) database.Model {
	return func(i int) database.Model {
		return tt[i]
	}
}

func (t *triggerType) Scan(val interface{}) error {
	b, err := database.Scan(val)

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
		return errors.New("unknown trigger " + s)
	}
	return nil
}

func (t triggerType) Value() (driver.Value, error) { return driver.Value(t.String()), nil }

func (d *triggerData) Scan(val interface{}) error {
	b, err := database.Scan(val)

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
	json.NewEncoder(buf).Encode(d)
	return buf.String()
}

func (d triggerData) Value() (driver.Value, error) { return driver.Value(d.String()), nil }

// Bind implements the database.Binder interface. This will only bind the models
// if they are pointers to a Build model.
func (t *Trigger) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			t.Build = m.(*Build)
		}
	}
}

func (t *Trigger) SetPrimary(i int64) { t.ID = i }

func (t Trigger) Primary() (string, int64) { return "id", t.ID }

// IsZero implements the database.Model interface.
func (t *Trigger) IsZero() bool {
	return t == nil || t.ID == 0 &&
		t.BuildID == 0 &&
		t.Type == triggerType(0) &&
		t.Comment == "" &&
		len(t.Data) == 0 &&
		t.CreatedAt == time.Time{}
}

// JSON implements the database.Model interface. This will return a map with
// the current Trigger's values under each key.
func (t *Trigger) JSON(_ string) map[string]interface{} {
	return map[string]interface{}{
		"type":    t.Type.String(),
		"comment": t.Comment,
		"data":    t.Data,
	}
}

// Endpoint is a stub to fulfill the database.Model interface. It returns an empty
// string.
func (*Trigger) Endpoint(_ ...string) string { return "" }

// Values implements the database.Model interface. This will return a map with
// the following values, build_id, provider_id, type, comment, and data.
func (t Trigger) Values() map[string]interface{} {
	return map[string]interface{}{
		"build_id":    t.BuildID,
		"provider_id": t.ProviderID,
		"type":        t.Type,
		"comment":     t.Comment,
		"data":        t.Data,
	}
}

// CommentBody parses the trigger comment to get the body of the comment. This
// will typically return the lines of the comment that appear after the first
// newline character that is found. If there is no newline character, and the
// trigger comment itself is less than 72 characters in length, then nothing
// is returned. The first 72 characters are summed to be the title of the
// comment.
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
		return "..." + body
	}
	return body
}

// CommentTitle parses the trigger comment to get the title of the comment.
// This treats the first line of the trigger comment as the title. If that
// first line is longer than 72 characters, then only the first 72 characters
// will be returned.
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
		return title + "..."
	}
	return title
}

// String returns a formatted string of the trigger itself, this will detail the
// user who submitted it, if not nil, and format the comment into the title and
// body.
func (t Trigger) String() string {
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
		buf.WriteString("Committed " + t.Data["id"][:7] + " to " + t.Data["ref"] + "\n")
	case Pull:
		buf.WriteString(strings.Title(t.Data["action"]) + " pull request to " + t.Data["ref"] + "\n")
	}

	if t.Comment != "" {
		buf.WriteString("\n" + t.CommentTitle() + "\n")
		buf.WriteString(t.CommentBody() + "\n")
	}
	return buf.String()
}

// Bind implements the database.Binder interface. This will only bind the models
// if they are pointers to a Build model.
func (s *TriggerStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *Build:
			s.Build = m.(*Build)
		}
	}
}

func (s *TriggerStore) Create(tt ...*Trigger) error {
	mm := database.ModelSlice(len(tt), TriggerModel(tt))
	return errors.Err(s.Store.Create(triggerTable, mm...))
}

func (s TriggerStore) Get(opts ...query.Option) (*Trigger, error) {
	t := &Trigger{
		Build: s.Build,
	}

	opts = append([]query.Option{
		database.Where(s.Build, "build_id"),
	}, opts...)

	err := s.Store.Get(t, triggerTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return t, errors.Err(err)
}

// All returns a slice of Trigger models, applying each query.Option that is
// given. The database.Where option is used on the Build bound database to limit the
// query to those relations.
func (s TriggerStore) All(opts ...query.Option) ([]*Trigger, error) {
	tt := make([]*Trigger, 0)

	opts = append([]query.Option{
		database.Where(s.Build, "build_id"),
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

// Load loads in a slice of Trigger models where the given key is in the list of
// given vals. Each database is loaded individually via a call to the given load
// callback. This method calls StageStore.All under the hood, so any bound
// models will impact the models being loaded.
func (s TriggerStore) Load(key string, vals []interface{}, load database.LoaderFunc) error {
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
