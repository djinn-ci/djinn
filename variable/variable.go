package variable

import (
	"database/sql"
	"encoding/base64"
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

type Variable struct {
	ID          int64
	UserID      int64
	AuthorID    int64
	NamespaceID sql.NullInt64
	Key         string
	Value       string
	Masked      bool
	CreatedAt   time.Time

	Author    *user.User
	User      *user.User
	Namespace *namespace.Namespace
}

var _ database.Model = (*Variable)(nil)

func LoadNamespaces(db database.Pool, vv ...*Variable) error {
	mm := make([]database.Model, 0, len(vv))

	for _, v := range vv {
		mm = append(mm, v)
	}

	if err := namespace.Load(db, mm...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func LoadRelations(db database.Pool, vv ...*Variable) error {
	mm := make([]database.Model, 0, len(vv))

	for _, v := range vv {
		mm = append(mm, v)
	}

	if err := database.LoadRelations(mm, namespace.ResourceRelations(db)...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (v *Variable) Dest() []interface{} {
	return []interface{}{
		&v.ID,
		&v.UserID,
		&v.AuthorID,
		&v.NamespaceID,
		&v.Key,
		&v.Value,
		&v.CreatedAt,
		&v.Masked,
	}
}

func (v *Variable) Bind(m database.Model) {
	switch v2 := m.(type) {
	case *user.User:
		v.Author = v2

		if v.UserID == v2.ID {
			v.User = v2
		}
	case *namespace.Namespace:
		if v.NamespaceID.Int64 == v2.ID {
			v.Namespace = v2
		}
	}
}

func (v *Variable) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/variables/" + strconv.FormatInt(v.ID, 10) + "/" + strings.Join(uri, "/")
	}
	return "/variables/" + strconv.FormatInt(v.ID, 10)
}

func (v *Variable) JSON(addr string) map[string]interface{} {
	if v == nil {
		return nil
	}

	val := v.Value

	if v.Masked {
		val = MaskString
	}

	json := map[string]interface{}{
		"id":           v.ID,
		"author_id":    v.AuthorID,
		"user_id":      v.UserID,
		"namespace_id": nil,
		"key":          v.Key,
		"value":        val,
		"masked":       v.Masked,
		"created_at":   v.CreatedAt.Format(time.RFC3339),
		"url":          addr + v.Endpoint(),
	}

	if v.Author != nil {
		json["author"] = v.Author.JSON(addr)
	}

	if v.User != nil {
		json["user"] = v.User.JSON(addr)
	}

	if v.NamespaceID.Valid {
		json["namespace_id"] = v.NamespaceID.Int64

		if v.Namespace != nil {
			json["namespace"] = v.Namespace.JSON(addr)
		}
	}
	return json
}

func (v *Variable) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":           v.ID,
		"user_id":      v.UserID,
		"author_id":    v.AuthorID,
		"namespace_id": v.NamespaceID,
		"key":          v.Key,
		"value":        v.Value,
		"created_at":   v.CreatedAt,
	}
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

	ev := event.New(e.Variable.NamespaceID, event.Variables, map[string]interface{}{
		"variable": e.Variable.JSON(env.DJINN_API_SERVER),
		"action":   e.Action,
	})

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

type Store struct {
	database.Pool
	*crypto.AESGCM
}

var (
	_ database.Loader = (*Store)(nil)

	table = "variables"
)

func Chown(db database.Pool, from, to int64) error {
	if err := database.Chown(db, table, from, to); err != nil {
		return errors.Err(err)
	}
	return nil
}

var (
	MaskString = "xxxxxx"
	MaskLen    = len(MaskString)
)

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
	UserID    int64
	Namespace namespace.Path
	Key       string
	Value     string
	Masked    bool
}

func (s Store) Create(p Params) (*Variable, error) {
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

	if p.Masked {
		b, err := s.AESGCM.Encrypt([]byte(p.Value))

		if err != nil {
			return nil, errors.Err(err)
		}
		p.Value = base64.StdEncoding.EncodeToString(b)
	}

	now := time.Now()

	q := query.Insert(
		table,
		query.Columns("user_id", "author_id", "namespace_id", "key", "value", "masked", "created_at"),
		query.Values(userId, p.UserID, namespaceId, p.Key, p.Value, p.Masked, now),
		query.Returning("id"),
	)

	var id int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}

	return &Variable{
		ID:          id,
		UserID:      userId,
		AuthorID:    p.UserID,
		NamespaceID: namespaceId,
		Key:         p.Key,
		Value:       p.Value,
		Masked:      p.Masked,
		CreatedAt:   now,
		Namespace:   n,
	}, nil
}

func (s Store) Delete(id int64) error {
	q := query.Delete(table, query.Where("id", "=", query.Arg(id)))

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) Get(opts ...query.Option) (*Variable, bool, error) {
	var v Variable

	ok, err := s.Pool.Get(table, &v, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &v, ok, nil
}

func (s Store) All(opts ...query.Option) ([]*Variable, error) {
	vv := make([]*Variable, 0)

	new := func() database.Model {
		v := &Variable{}
		vv = append(vv, v)
		return v
	}

	if err := s.Pool.All(table, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return vv, nil
}

func (s Store) Paginate(page, limit int64, opts ...query.Option) (database.Paginator, error) {
	paginator, err := s.Pool.Paginate(table, page, limit, opts...)

	if err != nil {
		return paginator, errors.Err(err)
	}
	return paginator, nil
}

func (s Store) Index(vals url.Values, opts ...query.Option) ([]*Variable, database.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		database.Search("key", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, database.PageLimit, opts...)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	vv, err := s.All(append(
		opts,
		query.OrderAsc("key"),
		query.Limit(paginator.Limit),
		query.Offset(paginator.Offset),
	)...)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}
	return vv, paginator, nil
}

func (s Store) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	vv, err := s.All(query.Where(pk, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	for _, v := range vv {
		for _, m := range mm {
			m.Bind(v)
		}
	}
	return nil
}
