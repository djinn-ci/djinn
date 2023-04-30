package image

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/namespace"
	"djinn-ci.com/queue"

	"github.com/andrewpillar/fs"
	"github.com/andrewpillar/query"
)

type Image struct {
	ID          int64
	UserID      int64
	AuthorID    int64
	NamespaceID database.Null[int64]
	Driver      driver.Type
	Hash        string
	Name        string
	CreatedAt   time.Time

	Author    *auth.User
	User      *auth.User
	Download  *Download
	Namespace *namespace.Namespace
}

func LoadRelations(ctx context.Context, pool *database.Pool, ii ...*Image) error {
	if err := namespace.LoadResourceRelations[*Image](ctx, pool, ii...); err != nil {
		return errors.Err(err)
	}

	rel := database.Relation{
		From: "id",
		To:   "image_id",
		Loader: database.ModelLoader(pool, downloadTable, func() database.Model {
			return &Download{}
		}),
	}

	if err := database.LoadRelations[*Image](ctx, ii, rel); err != nil {
		return errors.Err(err)
	}
	return nil
}

var _ database.Model = (*Image)(nil)

func (i *Image) Primary() (string, any) { return "id", i.ID }

func (i *Image) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":           &i.ID,
		"user_id":      &i.UserID,
		"author_id":    &i.AuthorID,
		"namespace_id": &i.NamespaceID,
		"driver":       &i.Driver,
		"hash":         &i.Hash,
		"name":         &i.Name,
		"created_at":   &i.CreatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (i *Image) Params() database.Params {
	return database.Params{
		"id":           database.ImmutableParam(i.ID),
		"user_id":      database.CreateOnlyParam(i.UserID),
		"author_id":    database.CreateOnlyParam(i.AuthorID),
		"namespace_id": database.CreateOnlyParam(i.NamespaceID),
		"driver":       database.CreateOnlyParam(i.Driver),
		"hash":         database.CreateOnlyParam(i.Hash),
		"name":         database.CreateOnlyParam(i.Name),
		"created_at":   database.CreateOnlyParam(i.CreatedAt),
	}
}

func (i *Image) Bind(m database.Model) {
	switch v := m.(type) {
	case *Download:
		if i.ID == v.ImageID {
			i.Download = v
		}
	case *auth.User:
		i.Author = v

		if i.UserID == v.ID {
			i.User = v
		}
	case *namespace.Namespace:
		if i.NamespaceID.Elem == v.ID {
			i.Namespace = v
		}
	}
}

func (i *Image) Endpoint(elems ...string) string {
	if len(elems) > 0 {
		return "/images/" + strconv.FormatInt(i.ID, 10) + "/" + strings.Join(elems, "/")
	}
	return "/images/" + strconv.FormatInt(i.ID, 10)
}

func (i *Image) MarshalJSON() ([]byte, error) {
	if i == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"id":           i.ID,
		"author_id":    i.AuthorID,
		"user_id":      i.UserID,
		"namespace_id": i.NamespaceID,
		"driver":       i.Driver.String(),
		"name":         i.Name,
		"created_at":   i.CreatedAt,
		"url":          env.DJINN_API_SERVER + i.Endpoint(),
		"author":       i.Author,
		"user":         i.User,
		"namespace":    i.Namespace,
		"download":     i.Download,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (i *Image) Downloaded() bool {
	if i.Download != nil {
		return !i.Download.Error.Valid && i.Download.FinishedAt.Valid
	}
	return true
}

// filestore returns the fs.FS that should be used for storing the current
// image.
func (i *Image) filestore(store fs.FS) (fs.FS, error) {
	store, err := store.Sub(strconv.FormatInt(i.UserID, 10))

	if err != nil {
		return nil, errors.Err(err)
	}

	store, err = store.Sub(i.Driver.String())

	if err != nil {
		return nil, errors.Err(err)
	}
	return store, nil
}

func (i *Image) Open(store fs.FS) (fs.File, error) {
	store, err := i.filestore(store)

	if err != nil {
		return nil, errors.Err(err)
	}

	f, err := store.Open(i.Hash)

	if err != nil {
		return nil, errors.Err(err)
	}
	return f, nil
}

type Event struct {
	dis event.Dispatcher

	Image  *Image
	Action string
}

func InitEvent(dis event.Dispatcher) queue.InitFunc {
	return func(j queue.Job) {
		if ev, ok := j.(*Event); ok {
			ev.dis = dis
		}
	}
}

func (ev *Event) Name() string { return "event:" + event.Images.String() }

func (ev *Event) Perform() error {
	if ev.dis == nil {
		return event.ErrNilDispatcher
	}

	data := map[string]any{
		"image":  ev.Image,
		"action": ev.Action,
	}
	return errors.Err(ev.dis.Dispatch(event.New(ev.Image.NamespaceID, event.Images, data)))
}

var _ queue.Job = (*Event)(nil)

const table = "images"

type Store struct {
	*database.Store[*Image]

	FS     fs.FS
	Hasher *crypto.Hasher
}

func NewStore(pool *database.Pool) *database.Store[*Image] {
	return database.NewStore[*Image](pool, table, func() *Image {
		return &Image{}
	})
}

type Params struct {
	User      *auth.User
	Namespace namespace.Path
	Name      string
	Driver    driver.Type
	Image     io.Reader
}

func (s *Store) Create(ctx context.Context, p *Params) (*Image, error) {
	hash, err := s.Hasher.HashNow()

	if err != nil {
		return nil, errors.Err(err)
	}

	i := Image{
		UserID:    p.User.ID,
		AuthorID:  p.User.ID,
		Driver:    p.Driver,
		Hash:      hash,
		Name:      p.Name,
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

		i.UserID = owner.ID
		i.NamespaceID.Elem = n.ID
		i.NamespaceID.Valid = true

		i.User = owner
		i.Namespace = n
	}

	if p.Image != nil {
		f, err := fs.ReadFile(hash, p.Image)

		if err != nil {
			return nil, errors.Err(err)
		}

		defer fs.Cleanup(f)

		store, err := i.filestore(s.FS)

		if err != nil {
			return nil, errors.Err(err)
		}

		f, err = store.Put(f)

		if err != nil {
			return nil, errors.Err(err)
		}
	}

	if err := s.Store.Create(ctx, &i); err != nil {
		return nil, errors.Err(err)
	}
	return &i, nil
}

func (s *Store) Delete(ctx context.Context, i *Image) error {
	if err := s.Store.Delete(ctx, i); err != nil {
		return errors.Err(err)
	}

	store, err := i.filestore(s.FS)

	if err != nil {
		return errors.Err(err)
	}

	if err := store.Remove(i.Hash); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return errors.Err(err)
		}
	}
	return nil
}

func (s *Store) Index(ctx context.Context, vals url.Values, opts ...query.Option) (*database.Paginator[*Image], error) {
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
