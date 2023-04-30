package object

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/json"
	"io"
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

	"github.com/andrewpillar/fs"
	"github.com/andrewpillar/query"
)

type Object struct {
	ID          int64
	UserID      int64
	AuthorID    int64
	NamespaceID database.Null[int64]
	Hash        string
	Name        string
	Type        string
	Size        int64
	MD5         database.Bytea
	SHA256      database.Bytea
	CreatedAt   time.Time
	DeletedAt   database.Null[time.Time]

	Author    *auth.User
	User      *auth.User
	Namespace *namespace.Namespace
}

var _ database.Model = (*Object)(nil)

func (o *Object) Primary() (string, any) { return "id", o.ID }

func (o *Object) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":           &o.ID,
		"user_id":      &o.UserID,
		"author_id":    &o.AuthorID,
		"namespace_id": &o.NamespaceID,
		"hash":         &o.Hash,
		"name":         &o.Name,
		"type":         &o.Type,
		"size":         &o.Size,
		"md5":          &o.MD5,
		"sha256":       &o.SHA256,
		"created_at":   &o.CreatedAt,
		"deleted_at":   &o.DeletedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (o *Object) Params() database.Params {
	return database.Params{
		"id":           database.ImmutableParam(o.ID),
		"user_id":      database.CreateOnlyParam(o.UserID),
		"author_id":    database.CreateOnlyParam(o.AuthorID),
		"namespace_id": database.CreateOnlyParam(o.NamespaceID),
		"hash":         database.CreateOnlyParam(o.Hash),
		"name":         database.CreateOnlyParam(o.Name),
		"type":         database.CreateOnlyParam(o.Type),
		"size":         database.CreateOnlyParam(o.Size),
		"md5":          database.CreateOnlyParam(o.MD5),
		"sha256":       database.CreateOnlyParam(o.SHA256),
		"created_at":   database.CreateOnlyParam(o.CreatedAt),
		"deleted_at":   database.CreateOnlyParam(o.DeletedAt),
	}
}

func (o *Object) Bind(m database.Model) {
	switch v := m.(type) {
	case *auth.User:
		o.Author = v

		if o.UserID == v.ID {
			o.User = v
		}
	case *namespace.Namespace:
		if o.NamespaceID.Elem == v.ID {
			o.Namespace = v
		}
	}
}

func (o *Object) Endpoint(elems ...string) string {
	if len(elems) > 0 {
		return "/objects/" + strconv.FormatInt(o.ID, 10) + "/" + strings.Join(elems, "/")
	}
	return "/objects/" + strconv.FormatInt(o.ID, 10)
}

func (o *Object) MarshalJSON() ([]byte, error) {
	if o == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"id":           o.ID,
		"author_id":    o.AuthorID,
		"user_id":      o.UserID,
		"namespace_id": o.NamespaceID,
		"name":         o.Name,
		"type":         o.Type,
		"size":         o.Size,
		"md5":          o.MD5,
		"sha256":       o.SHA256,
		"created_at":   o.CreatedAt,
		"url":          env.DJINN_API_SERVER + o.Endpoint(),
		"builds_url":   env.DJINN_API_SERVER + o.Endpoint("builds"),
		"author":       o.Author,
		"user":         o.User,
		"namespace":    o.Namespace,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (o *Object) filestore(store fs.FS) (fs.FS, error) {
	store, err := store.Sub(strconv.FormatInt(o.UserID, 10))

	if err != nil {
		return nil, errors.Err(err)
	}
	return store, nil
}

func (o *Object) Open(store fs.FS) (fs.File, error) {
	store, err := o.filestore(store)

	if err != nil {
		return nil, errors.Err(err)
	}

	f, err := store.Open(o.Hash)

	if err != nil {
		return nil, errors.Err(err)
	}
	return f, nil
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

	ev := event.New(e.Object.NamespaceID, event.Objects, map[string]any{
		"object": e.Object,
		"action": e.Action,
	})

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

const table = "objects"

type Store struct {
	*database.Store[*Object]

	FS     fs.FS
	Hasher *crypto.Hasher
}

func NewStore(pool *database.Pool) *database.Store[*Object] {
	return database.NewStore[*Object](pool, table, func() *Object {
		return &Object{}
	})
}

type Params struct {
	User      *auth.User
	Namespace namespace.Path
	Name      string
	Object    io.ReadSeeker
}

func (s *Store) Create(ctx context.Context, p *Params) (*Object, error) {
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

	md5 := md5.New()
	sha256 := sha256.New()

	f, err := fs.ReadFile(hash, io.TeeReader(p.Object, io.MultiWriter(md5, sha256)))

	if err != nil {
		return nil, errors.Err(err)
	}

	defer fs.Cleanup(f)

	o := Object{
		UserID:    p.User.ID,
		AuthorID:  p.User.ID,
		Hash:      hash,
		Name:      p.Name,
		Type:      detectContentType(hdr),
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

		o.UserID = owner.ID
		o.NamespaceID.Elem = n.ID
		o.NamespaceID.Valid = n.ID > 0

		o.User = owner
		o.Namespace = n
	}

	store, err := o.filestore(s.FS)

	if err != nil {
		return nil, errors.Err(err)
	}

	f, err = store.Put(f)

	if err != nil {
		return nil, errors.Err(err)
	}

	info, err := f.Stat()

	if err != nil {
		return nil, errors.Err(err)
	}

	o.Size = info.Size()
	o.MD5 = database.Bytea(md5.Sum(nil))
	o.SHA256 = database.Bytea(sha256.Sum(nil))

	if err := s.Store.Create(ctx, &o); err != nil {
		return nil, errors.Err(err)
	}
	return &o, nil
}

func (s *Store) Delete(ctx context.Context, o *Object) error {
	if err := s.Store.Delete(ctx, o); err != nil {
		return errors.Err(err)
	}

	store, err := o.filestore(s.FS)

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

func (s *Store) Index(ctx context.Context, vals url.Values, opts ...query.Option) (*database.Paginator[*Object], error) {
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

func (s *Store) Open(name string) (fs.File, error) {
	f, err := s.FS.Open(name)

	if err != nil {
		return nil, errors.Err(err)
	}
	return f, nil
}

func (s *Store) Sub(dir string) (fs.FS, error) {
	sub, err := s.FS.Sub(dir)

	if err != nil {
		return nil, errors.Err(err)
	}
	return sub, nil
}

func (s *Store) Stat(name string) (fs.FileInfo, error) {
	info, err := s.FS.Stat(name)

	if err != nil {
		return nil, errors.Err(err)
	}
	return info, nil
}

func (s *Store) Put(f fs.File) (fs.File, error) {
	f, err := s.FS.Put(f)

	if err != nil {
		return nil, errors.Err(err)
	}
	return f, nil
}

func (s *Store) Remove(name string) error {
	if err := s.FS.Remove(name); err != nil {
		return errors.Err(err)
	}
	return nil
}
