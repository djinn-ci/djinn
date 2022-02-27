package object

import (
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/fs"
	"djinn-ci.com/namespace"
	"djinn-ci.com/queue"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type Object struct {
	ID          int64
	UserID      int64
	AuthorID    int64
	NamespaceID sql.NullInt64
	Hash        string
	Name        string
	Type        string
	Size        int64
	MD5         []byte
	SHA256      []byte
	CreatedAt   time.Time
	DeletedAt   sql.NullTime

	Author    *user.User
	User      *user.User
	Namespace *namespace.Namespace
}

var _ database.Model = (*Object)(nil)

func LoadNamespaces(db database.Pool, oo ...*Object) error {
	mm := make([]database.Model, 0, len(oo))

	for _, o := range oo {
		mm = append(mm, o)
	}

	if err := namespace.Load(db, mm...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func LoadRelations(db database.Pool, oo ...*Object) error {
	mm := make([]database.Model, 0, len(oo))

	for _, o := range oo {
		mm = append(mm, o)
	}

	if err := database.LoadRelations(mm, namespace.ResourceRelations(db)...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (o *Object) Data(store fs.Store) (fs.Record, error) {
	store, err := store.Partition(o.UserID)

	if err != nil {
		return nil, errors.Err(err)
	}

	rec, err := store.Open(o.Hash)

	if err != nil {
		return nil, errors.Err(err)
	}
	return rec, nil
}

func (o *Object) Dest() []interface{} {
	return []interface{}{
		&o.ID,
		&o.UserID,
		&o.AuthorID,
		&o.NamespaceID,
		&o.Hash,
		&o.Name,
		&o.Type,
		&o.Size,
		&o.MD5,
		&o.SHA256,
		&o.CreatedAt,
		&o.DeletedAt,
	}
}

func (o *Object) Bind(m database.Model) {
	switch v := m.(type) {
	case *user.User:
		o.Author = v

		if o.UserID == v.ID {
			o.User = v
		}
	case *namespace.Namespace:
		if o.NamespaceID.Int64 == v.ID {
			o.Namespace = v
		}
	}
}

func (o *Object) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/objects/" + strconv.FormatInt(o.ID, 10) + "/" + strings.Join(uri, "/")
	}
	return "/objects/" + strconv.FormatInt(o.ID, 10)
}

func (o *Object) JSON(addr string) map[string]interface{} {
	if o == nil {
		return nil
	}

	json := map[string]interface{}{
		"id":           o.ID,
		"author_id":    o.AuthorID,
		"user_id":      o.UserID,
		"namespace_id": nil,
		"name":         o.Name,
		"type":         o.Type,
		"size":         o.Size,
		"md5":          hex.EncodeToString(o.MD5),
		"sha256":       hex.EncodeToString(o.SHA256),
		"created_at":   o.CreatedAt.Format(time.RFC3339),
		"url":          addr + o.Endpoint(),
		"builds_url":   addr + o.Endpoint("builds"),
		"author":       o.Author.JSON(addr),
		"user":         o.User.JSON(addr),
		"namespace":    o.Namespace.JSON(addr),
	}

	if o.Author != nil {
		json["author"] = o.Author.JSON(addr)
	}

	if o.User != nil {
		json["user"] = o.User.JSON(addr)
	}

	if o.NamespaceID.Valid {
		json["namespace_id"] = o.NamespaceID.Int64

		if o.Namespace != nil {
			json["namespace"] = o.Namespace.JSON(addr)
		}
	}
	return json
}

func (o *Object) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":           o.ID,
		"user_id":      o.UserID,
		"author_id":    o.AuthorID,
		"namespace_id": o.NamespaceID,
		"hash":         o.Hash,
		"name":         o.Name,
		"type":         o.Type,
		"size":         o.Size,
		"md5":          o.MD5,
		"sha256":       o.SHA256,
		"created_at":   o.CreatedAt,
		"deleted_at":   o.DeletedAt,
	}
}

type Event struct {
	dis event.Dispatcher

	Object *Object
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

func (e *Event) Name() string { return "event:" + event.Objects.String() }

func (e *Event) Perform() error {
	if e.dis == nil {
		return event.ErrNilDispatcher
	}

	ev := event.New(e.Object.NamespaceID, event.Objects, map[string]interface{}{
		"object": e.Object.JSON(env.DJINN_API_SERVER),
		"action": e.Action,
	})

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

type Store struct {
	database.Pool
	fs.Store

	Hasher *crypto.Hasher
}

var (
	_ database.Loader = (*Store)(nil)

	table = "objects"
)

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
	Object    io.ReadSeeker
}

func (s *Store) Create(p Params) (*Object, error) {
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

	hdr := make([]byte, 512)

	if _, err := p.Object.Read(hdr); err != nil {
		return nil, errors.Err(err)
	}

	if _, err := p.Object.Seek(0, io.SeekStart); err != nil {
		return nil, errors.Err(err)
	}

	hash, err := s.Hasher.HashNow()

	if err != nil {
		return nil, errors.Err(err)
	}

	store, err := s.Partition(userId)

	if err != nil {
		return nil, errors.Err(err)
	}

	md5 := md5.New()
	sha256 := sha256.New()

	tee := io.TeeReader(p.Object, io.MultiWriter(md5, sha256))

	dst, err := store.Create(hash)

	if err != nil {
		return nil, errors.Err(err)
	}

	defer dst.Close()

	size, err := io.Copy(dst, tee)

	if err != nil {
		return nil, errors.Err(err)
	}

	now := time.Now()

	typ := detectContentType(hdr)

	md5sum := md5.Sum(nil)
	sha256sum := sha256.Sum(nil)

	q := query.Insert(
		table,
		query.Columns("user_id", "author_id", "namespace_id", "hash", "name", "type", "size", "md5", "sha256", "created_at"),
		query.Values(userId, p.UserID, namespaceId, hash, p.Name, typ, size, md5sum, sha256sum, now),
		query.Returning("id"),
	)

	var id int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}

	return &Object{
		ID:          id,
		UserID:      userId,
		AuthorID:    p.UserID,
		NamespaceID: namespaceId,
		Hash:        hash,
		Name:        p.Name,
		Type:        typ,
		Size:        int64(size),
		MD5:         md5sum,
		SHA256:      sha256sum,
		CreatedAt:   now,
		Namespace:   n,
	}, nil
}

func (s *Store) Delete(o *Object) error {
	q := query.Delete(table, query.Where("id", "=", query.Arg(o.ID)))

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}

	store, err := s.Partition(o.UserID)

	if err != nil {
		return errors.Err(err)
	}

	if err := store.Remove(o.Hash); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return errors.Err(err)
		}
	}
	return nil
}

func (s *Store) Get(opts ...query.Option) (*Object, bool, error) {
	var o Object

	ok, err := s.Pool.Get(table, &o, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &o, ok, nil
}

func (s *Store) All(opts ...query.Option) ([]*Object, error) {
	oo := make([]*Object, 0)

	new := func() database.Model {
		o := &Object{}
		oo = append(oo, o)
		return o
	}

	if err := s.Pool.All(table, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return oo, nil
}

func (s *Store) Paginate(page, limit int64, opts ...query.Option) (database.Paginator, error) {
	paginator, err := s.Pool.Paginate(table, page, limit, opts...)

	if err != nil {
		return paginator, errors.Err(err)
	}
	return paginator, nil
}

func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Object, database.Paginator, error) {
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

	oo, err := s.All(append(
		opts,
		query.OrderAsc("name"),
		query.Limit(paginator.Limit),
		query.Offset(paginator.Offset),
	)...)
	return oo, paginator, errors.Err(err)
}

func (s *Store) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	oo, err := s.All(query.Where(pk, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	for _, o := range oo {
		for _, m := range mm {
			m.Bind(o)
		}
	}
	return nil
}
