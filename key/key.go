package key

import (
	"database/sql"
	"net/url"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/namespace"
	"djinn-ci.com/queue"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type Key struct {
	ID          int64
	UserID      int64
	AuthorID    int64
	NamespaceID sql.NullInt64
	Name        string
	Key         []byte
	Config      string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Author    *user.User
	User      *user.User
	Namespace *namespace.Namespace
}

var _ database.Model = (*Key)(nil)

func LoadNamespaces(db database.Pool, kk ...*Key) error {
	mm := make([]database.Model, 0, len(kk))

	for _, k := range kk {
		mm = append(mm, k)
	}

	if err := namespace.Load(db, mm...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func LoadRelations(db database.Pool, kk ...*Key) error {
	mm := make([]database.Model, 0, len(kk))

	for _, k := range kk {
		mm = append(mm, k)
	}

	if err := database.LoadRelations(mm, namespace.ResourceRelations(db)...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (k *Key) Dest() []interface{} {
	return []interface{}{
		&k.ID,
		&k.UserID,
		&k.AuthorID,
		&k.NamespaceID,
		&k.Name,
		&k.Key,
		&k.Config,
		&k.CreatedAt,
		&k.UpdatedAt,
	}
}

func (k *Key) Bind(m database.Model) {
	switch v := m.(type) {
	case *user.User:
		k.Author = v

		if k.UserID == v.ID {
			k.User = v
		}
	case *namespace.Namespace:
		if k.NamespaceID.Int64 == v.ID {
			k.Namespace = v
		}
	}
}

func (k *Key) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/keys/" + strconv.FormatInt(k.ID, 10) + "/" + strings.Join(uri, "/")
	}
	return "/keys/" + strconv.FormatInt(k.ID, 10)
}

func (k *Key) JSON(addr string) map[string]interface{} {
	if k == nil {
		return nil
	}

	json := map[string]interface{}{
		"id":           k.ID,
		"author_id":    k.AuthorID,
		"user_id":      k.UserID,
		"namespace_id": nil,
		"name":         k.Name,
		"config":       k.Config,
		"created_at":   k.CreatedAt.Format(time.RFC3339),
		"updated_at":   k.UpdatedAt.Format(time.RFC3339),
		"url":          addr + k.Endpoint(),
	}

	if k.Author != nil {
		json["author"] = k.Author.JSON(addr)
	}

	if k.User != nil {
		json["user"] = k.User.JSON(addr)
	}

	if k.NamespaceID.Valid {
		json["namespace_id"] = k.NamespaceID.Int64

		if k.Namespace != nil {
			json["namespace"] = k.Namespace.JSON(addr)
		}
	}
	return json
}

func (k *Key) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":           k.ID,
		"user_id":      k.UserID,
		"author_id":    k.AuthorID,
		"namespace_id": k.NamespaceID,
		"name":         k.Name,
		"key":          k.Key,
		"config":       k.Config,
		"created_at":   k.CreatedAt,
		"updated_at":   k.UpdatedAt,
	}
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

	ev := event.New(e.Key.NamespaceID, event.SSHKeys, map[string]interface{}{
		"key":    e.Key.JSON(env.DJINN_API_SERVER),
		"action": e.Action,
	})

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

type Store struct {
	database.Pool

	AESGCM *crypto.AESGCM
}

var table = "keys"

func Chown(db database.Pool, from, to int64) error {
	if err := database.Chown(db, table, from, to); err != nil {
		return errors.Err(err)
	}
	return nil
}

type Params struct {
	UserID    int64
	Namespace namespace.Path
	Name      string
	Key       string
	Config    string
}

func stripCRLF(s string) string {
	return strings.Replace(s, "\r", "", -1)
}

func (s *Store) Create(p Params) (*Key, error) {
	if s.AESGCM == nil {
		return nil, crypto.ErrNilAESGCM
	}

	key := []byte(stripCRLF(p.Key))

	b, err := s.AESGCM.Encrypt(key)

	if err != nil {
		return nil, errors.Err(err)
	}

	u, n, err := p.Namespace.ResolveOrCreate(s.Pool, p.UserID)

	if err != nil {
		if !errors.Is(err, namespace.ErrInvalidPath) {
			return nil, errors.Err(err)
		}
	}

	userId := p.UserID

	var namespaceId sql.NullInt64

	if u != nil {
		n.User = u

		userId = u.ID
		namespaceId = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}

		if err := n.IsCollaborator(s.Pool, p.UserID); err != nil {
			return nil, errors.Err(err)
		}
	}

	now := time.Now()

	q := query.Insert(
		table,
		query.Columns("user_id", "author_id", "namespace_id", "name", "key", "config", "created_at", "updated_at"),
		query.Values(userId, p.UserID, namespaceId, p.Name, b, p.Config, now, now),
		query.Returning("id"),
	)

	var id int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}

	return &Key{
		ID:          id,
		UserID:      userId,
		AuthorID:    p.UserID,
		NamespaceID: namespaceId,
		Name:        p.Name,
		Key:         b,
		Config:      p.Config,
		CreatedAt:   now,
		UpdatedAt:   now,
		Namespace:   n,
	}, nil
}

func (s *Store) Update(id int64, p Params) error {
	q := query.Update(
		table,
		query.Set("config", query.Arg(p.Config)),
		query.Set("updated_at", query.Arg(time.Now())),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Store) Delete(id int64) error {
	q := query.Delete(table, query.Where("id", "=", query.Arg(id)))

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Store) Get(opts ...query.Option) (*Key, bool, error) {
	var k Key

	ok, err := s.Pool.Get(table, &k, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &k, ok, nil
}

func (s *Store) All(opts ...query.Option) ([]*Key, error) {
	kk := make([]*Key, 0)

	new := func() database.Model {
		k := &Key{}
		kk = append(kk, k)
		return k
	}

	if err := s.Pool.All(table, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return kk, nil
}

func (s *Store) Paginate(page, limit int64, opts ...query.Option) (database.Paginator, error) {
	paginator, err := s.Pool.Paginate(table, page, limit, opts...)

	if err != nil {
		return paginator, errors.Err(err)
	}
	return paginator, nil
}

func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Key, database.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		database.Search("name", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, database.PageLimit, opts...)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	kk, err := s.All(append(
		opts,
		query.OrderAsc("name"),
		query.Limit(paginator.Limit),
		query.Offset(paginator.Offset),
	)...)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}
	return kk, paginator, nil
}
