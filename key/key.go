package key

import (
	"context"
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

type Key struct {
	loaded []string

	ID          int64
	UserID      int64
	AuthorID    int64
	NamespaceID database.Null[int64]
	Name        string
	Key         []byte
	Config      string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Author    *auth.User
	User      *auth.User
	Namespace *namespace.Namespace
}

var _ database.Model = (*Key)(nil)

func (k *Key) Primary() (string, any) { return "id", k.ID }

func (k *Key) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":           &k.ID,
		"user_id":      &k.UserID,
		"author_id":    &k.AuthorID,
		"namespace_id": &k.NamespaceID,
		"name":         &k.Name,
		"key":          &k.Key,
		"config":       &k.Config,
		"created_at":   &k.CreatedAt,
		"updated_at":   &k.UpdatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}

	k.loaded = r.Columns
	return nil
}

func (k *Key) Params() database.Params {
	params := database.Params{
		"id":           database.ImmutableParam(k.ID),
		"user_id":      database.CreateOnlyParam(k.UserID),
		"author_id":    database.CreateOnlyParam(k.AuthorID),
		"namespace_id": database.CreateOnlyParam(k.NamespaceID),
		"name":         database.CreateOnlyParam(k.Name),
		"key":          database.CreateOnlyParam(k.Key),
		"config":       database.CreateUpdateParam(k.Config),
		"created_at":   database.CreateUpdateParam(k.CreatedAt),
		"updated_at":   database.CreateUpdateParam(k.UpdatedAt),
	}

	if len(k.loaded) > 0 {
		params.Only(k.loaded...)
	}
	return params
}

func (k *Key) Bind(m database.Model) {
	switch v := m.(type) {
	case *auth.User:
		k.Author = v

		if k.UserID == v.ID {
			k.User = v
		}
	case *namespace.Namespace:
		if k.NamespaceID.Elem == v.ID {
			k.Namespace = v
		}
	}
}

func (k *Key) MarshalJSON() ([]byte, error) {
	if k == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"id":           k.ID,
		"author_id":    k.AuthorID,
		"user_id":      k.UserID,
		"namespace_id": k.NamespaceID,
		"name":         k.Name,
		"config":       k.Config,
		"created_at":   k.CreatedAt,
		"updated_at":   k.UpdatedAt,
		"url":          env.DJINN_API_SERVER + k.Endpoint(),
		"author":       k.Author,
		"user":         k.User,
		"namespace":    k.Namespace,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (k *Key) Endpoint(elems ...string) string {
	if len(elems) > 0 {
		return "/keys/" + strconv.FormatInt(k.ID, 10) + "/" + strings.Join(elems, "/")
	}
	return "/keys/" + strconv.FormatInt(k.ID, 10)
}

type Event struct {
	dis event.Dispatcher

	Key    *Key
	Action string
}

var _ queue.Job = (*Event)(nil)

func InitEvent(dis event.Dispatcher) queue.InitFunc {
	return func(j queue.Job) {
		if ev, ok := j.(*Event); ok {
			ev.dis = dis
		}
	}
}

func (e *Event) Name() string { return "event:" + event.SSHKeys.String() }

func (e *Event) Perform() error {
	if e.dis == nil {
		return event.ErrNilDispatcher
	}

	ev := event.New(e.Key.NamespaceID, event.SSHKeys, map[string]any{
		"key":    e.Key,
		"action": e.Action,
	})

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

const table = "keys"

type Store struct {
	*database.Store[*Key]

	AESGCM *crypto.AESGCM
}

func NewStore(pool *database.Pool) *database.Store[*Key] {
	return database.NewStore[*Key](pool, table, func() *Key {
		return &Key{}
	})
}

type Params struct {
	User      *auth.User
	Namespace namespace.Path
	Name      string
	Key       string
	Config    string
}

func (s *Store) Create(ctx context.Context, p *Params) (*Key, error) {
	if s.AESGCM == nil {
		return nil, crypto.ErrNilAESGCM
	}

	p.Key = strings.Replace(p.Key, "\r", "", -1)

	b, err := s.AESGCM.Encrypt([]byte(p.Key))

	if err != nil {
		return nil, errors.Err(err)
	}

	k := Key{
		UserID:    p.User.ID,
		AuthorID:  p.User.ID,
		Name:      p.Name,
		Key:       b,
		Config:    p.Config,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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

		k.UserID = owner.ID
		k.NamespaceID.Elem = n.ID
		k.NamespaceID.Valid = n.ID > 0

		k.User = owner
		k.Namespace = n
	}

	if err := s.Store.Create(ctx, &k); err != nil {
		return nil, errors.Err(err)
	}
	return &k, nil
}

func (s *Store) Update(ctx context.Context, k *Key) error {
	loaded := k.loaded
	k.loaded = []string{"config", "updated_at"}

	k.UpdatedAt = time.Now()

	if err := s.Store.Update(ctx, k); err != nil {
		return errors.Err(err)
	}

	k.loaded = loaded
	return nil
}

func (s *Store) Index(ctx context.Context, vals url.Values, opts ...query.Option) (*database.Paginator[*Key], error) {
	page, _ := strconv.Atoi(vals.Get("page"))

	opts = append([]query.Option{
		database.Search("name", vals.Get("search")),
	}, opts...)

	p, err := s.Paginate(ctx, page, database.PageLimit, opts...)

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := p.Load(ctx, s.Store, append(opts, query.OrderAsc("name"))...); err != nil {
		return nil, errors.Err(err)
	}
	return p, nil
}
