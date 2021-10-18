package event

import (
	"database/sql"
	"database/sql/driver"
	"time"

	"djinn-ci.com/errors"

	"github.com/google/uuid"
)

type Dispatcher interface {
	Dispatch(ev *Event) error
}

type multiDispatcher struct {
	dispatchers []Dispatcher
}

func MultiDispatcher(dispatchers ...Dispatcher) Dispatcher {
	return &multiDispatcher{
		dispatchers: dispatchers,
	}
}

func (md *multiDispatcher) Dispatch(ev *Event) error {
	for _, d := range md.dispatchers {
		if err := d.Dispatch(ev); err != nil {
			return err
		}
	}
	return nil
}

type Type uint

type Event struct {
	ID          uuid.UUID
	NamespaceID sql.NullInt64
	Type        Type
	Data        map[string]interface{}
	CreatedAt   time.Time
}

//go:generate stringer -type Type -linecomment
const (
	BuildSubmitted Type = 1 << iota // build.submitted
	BuildStarted                    // build.started
	BuildFinished                   // build.finished
	BuildTagged                     // build.tagged
	InviteSent                      // invite.sent
	InviteAccepted                  // invite.accepted
	InviteRejected                  // invite.rejected
	Namespaces                      // namespaces
	Cron                            // cron
	Images                          // images
	Objects                         // objects
	Variables                       // variables
	SSHKeys                         // ssh_keys
)

var (
	_ sql.Scanner   = (*Type)(nil)
	_ driver.Valuer = (*Type)(nil)

	typesMap = map[string]Type{
		"build.submitted": BuildSubmitted,
		"build.started":   BuildStarted,
		"build.finished":  BuildFinished,
		"build.tagged":    BuildTagged,
		"invite.sent":     InviteSent,
		"invite.accepted": InviteAccepted,
		"invite.rejected": InviteRejected,
		"namespaces":      Namespaces,
		"cron":            Cron,
		"images":          Images,
		"objects":         Objects,
		"variables":       Variables,
		"ssh_keys":        SSHKeys,
	}

	Types = []Type{
		BuildSubmitted,
		BuildStarted,
		BuildFinished,
		BuildTagged,
		InviteSent,
		InviteAccepted,
		Namespaces,
		Cron,
		Images,
		Objects,
		Variables,
		SSHKeys,
	}

	ErrUnknown       = errors.New("unknown event")
	ErrNilDispatcher = errors.New("nil dispatcher")
)

func Lookup(name string) (Type, bool) {
	typ, ok := typesMap[name]
	return typ, ok
}

func New(namespaceId sql.NullInt64, typ Type, data map[string]interface{}) *Event {
	now := time.Now()

	return &Event{
		ID:          uuid.New(),
		NamespaceID: namespaceId,
		Type:        typ,
		Data:        data,
		CreatedAt:   now,
	}
}

func UnmarshalType(names ...string) (Type, error) {
	var typs Type

	for _, name := range names {
		typ, ok := typesMap[name]

		if !ok {
			return 0, ErrUnknown
		}
		typs |= typ
	}
	return typs, nil
}

func (t Type) Has(mask Type) bool { return (t & mask) == mask }

func (t Type) Value() (driver.Value, error) { return driver.Value(int32(t)), nil }

func (t *Type) Scan(v interface{}) error {
	if v == nil {
		return nil
	}

	i32, err := driver.Int32.ConvertValue(v)

	if err != nil {
		return errors.Err(err)
	}

	// driver.Value will be of int64 under the hood, but respects int32 bounds.
	i, ok := i32.(int64)

	if !ok {
		return errors.New("could not type assert event to int64")
	}

	(*t) = Type(i)
	return nil
}
