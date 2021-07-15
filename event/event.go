package event

import (
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"fmt"
	"time"

	"djinn-ci.com/errors"
)

type Dispatcher interface {
	Dispatch(ev *Event) error
}

type Type uint

type Event struct {
	ID          int64
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
		ID:          now.UnixNano(),
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

func (t Type) Value() (driver.Value, error) {
	size := binary.Size(t)

	if size < 0 {
		return nil, errors.New("invalid event")
	}

	buf := make([]byte, size)

	binary.PutVarint(buf, int64(t))
	return buf, nil
}

func (t *Type) Scan(v interface{}) error {
	if v == nil {
		return nil
	}

	str, err := driver.String.ConvertValue(v)

	if err != nil {
		return errors.Err(err)
	}

	b, ok := str.([]byte)

	if !ok {
		return errors.New(fmt.Sprintf("unexpected event type %T", str))
	}

	i, n := binary.Varint(b)

	if n < 0 {
		return errors.New("64 bit overflow")
	}

	(*t) = Type(i)
	return nil
}
