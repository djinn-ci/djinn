package variable

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/namespace"
	"djinn-ci.com/queue"

	"github.com/andrewpillar/query"
)

type Variable struct {
	loaded []string

	ID          int64
	UserID      int64
	AuthorID    int64
	NamespaceID database.Null[int64]
	Key         string
	Value       string
	Masked      bool
	CreatedAt   time.Time

	Author    *auth.User
	User      *auth.User
	Namespace *namespace.Namespace
}

var _ database.Model = (*Variable)(nil)

func (v *Variable) Primary() (string, any) { return "id", v.ID }

func (v *Variable) Params() database.Params {
	params := database.Params{
		"id":           database.ImmutableParam(v.ID),
		"user_id":      database.CreateUpdateParam(v.UserID),
		"author_id":    database.CreateOnlyParam(v.AuthorID),
		"namespace_id": database.CreateOnlyParam(v.NamespaceID),
		"key":          database.CreateOnlyParam(v.Key),
		"value":        database.CreateOnlyParam(v.Value),
		"masked":       database.CreateOnlyParam(v.Masked),
		"created_at":   database.CreateOnlyParam(v.CreatedAt),
	}

	if len(v.loaded) > 0 {
		params.Only(v.loaded...)
	}
	return params
}

func (v *Variable) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":           &v.ID,
		"user_id":      &v.UserID,
		"author_id":    &v.AuthorID,
		"namespace_id": &v.NamespaceID,
		"key":          &v.Key,
		"value":        &v.Value,
		"masked":       &v.Masked,
		"created_at":   &v.CreatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}

	v.loaded = r.Columns
	return nil
}

func (v *Variable) Bind(m database.Model) {
	switch v2 := m.(type) {
	case *auth.User:
		v.Author = v2

		if v.UserID == v2.ID {
			v.User = v2
		}
	case *namespace.Namespace:
		if v.NamespaceID.Elem == v2.ID {
			v.Namespace = v2
		}
	}
}

func (v *Variable) MarshalJSON() ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}

	val := v.Value

	if v.Masked {
		val = MaskString
	}

	b, err := json.Marshal(map[string]any{
		"id":           v.ID,
		"author_id":    v.AuthorID,
		"user_id":      v.UserID,
		"namespace_id": v.NamespaceID,
		"key":          v.Key,
		"value":        val,
		"masked":       v.Masked,
		"created_at":   v.CreatedAt,
		"url":          env.DJINN_API_SERVER + v.Endpoint(),
		"author":       v.Author,
		"user":         v.User,
		"namespace":    v.Namespace,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (v *Variable) String() string { return v.Key + "=" + v.Value }

func (v *Variable) Endpoint(elems ...string) string {
	if len(elems) > 0 {
		return "/variables/" + strconv.FormatInt(v.ID, 10) + "/" + strings.Join(elems, "/")
	}
	return "/variables/" + strconv.FormatInt(v.ID, 10)
}

type Event struct {
	dis event.Dispatcher

	Variable *Variable
	Action   string
}

var _ queue.Job = (*Event)(nil)

func InitEvent(dis event.Dispatcher) queue.InitFunc {
	return func(j queue.Job) {
		if ev, ok := j.(*Event); ok {
			ev.dis = dis
		}
	}
}

func (e *Event) Name() string { return "event:" + event.Variables.String() }

func (e *Event) Perform() error {
	if e.dis == nil {
		return event.ErrNilDispatcher
	}

	ev := event.New(e.Variable.NamespaceID, event.Variables, map[string]any{
		"variable": e.Variable,
		"action":   e.Action,
	})

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

const (
	table = "variables"

	MaskString = "xxxxxx"
)

var (
	MaskLen = len(MaskString)

	variableMaskKey = "unmask_variable_id"
)

type Store struct {
	*database.Store[*Variable]

	AESGCM *crypto.AESGCM
}

func Loader(pool *database.Pool) database.Loader {
	return database.ModelLoader(pool, table, func() database.Model {
		return &Variable{}
	})
}

func NewStore(pool *database.Pool) *database.Store[*Variable] {
	return database.NewStore(pool, table, func() *Variable {
		return &Variable{}
	})
}

func PutUnmasked(sessvals map[any]any, set map[int64]struct{}) {
	sessvals[variableMaskKey] = set
}

func GetUnmasked(sessvals map[any]any) map[int64]struct{} {
	v, ok := sessvals[variableMaskKey]

	if !ok {
		v = make(map[int64]struct{})
	}

	set, _ := v.(map[int64]struct{})
	return set
}

func Unmask(aesgcm *crypto.AESGCM, v *Variable) error {
	if !v.Masked {
		return nil
	}

	dec, err := base64.StdEncoding.DecodeString(v.Value)

	if err != nil {
		return errors.Err(err)
	}

	b, err := aesgcm.Decrypt([]byte(dec))

	if err != nil {
		return errors.Err(err)
	}

	v.Value = string(b)
	return nil
}

type Params struct {
	User      *auth.User
	Namespace namespace.Path
	Key       string
	Value     string
	Masked    bool
}

func (s Store) Create(ctx context.Context, p Params) (*Variable, error) {
	if p.Masked {
		b, err := s.AESGCM.Encrypt([]byte(p.Value))

		if err != nil {
			return nil, errors.Err(err)
		}
		p.Value = base64.StdEncoding.EncodeToString(b)
	}

	v := Variable{
		UserID:    p.User.ID,
		AuthorID:  p.User.ID,
		Key:       p.Key,
		Value:     p.Value,
		Masked:    p.Masked,
		CreatedAt: time.Now(),
		User:      p.User,
		Author:    p.User,
	}

	if p.Namespace.Valid {
		owner, n, err := p.Namespace.Resolve(ctx, s.Pool, p.User)

		if err != nil {
			if !errors.Is(err, namespace.ErrInvalidPath) {
				return nil, errors.Err(err)
			}
		}

		if err := n.IsCollaborator(ctx, s.Pool, p.User); err != nil {
			return nil, errors.Err(err)
		}

		v.UserID = owner.ID
		v.NamespaceID.Elem = n.ID
		v.NamespaceID.Valid = n.ID > 0

		v.User = owner
		v.Namespace = n
	}

	if err := s.Store.Create(ctx, &v); err != nil {
		return nil, errors.Err(err)
	}
	return &v, nil
}

func (s Store) Index(ctx context.Context, vals url.Values, opts ...query.Option) (*database.Paginator[*Variable], error) {
	page, _ := strconv.Atoi(vals.Get("page"))

	opts = append([]query.Option{
		database.Search("key", vals.Get("search")),
	}, opts...)

	p, err := s.Paginate(ctx, page, database.PageLimit, opts...)

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := p.Load(ctx, s.Store, append(opts, query.OrderAsc("key"))...); err != nil {
		return nil, errors.Err(err)
	}
	return p, nil
}
