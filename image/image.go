package image

import (
	"database/sql"
	"io"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/fs"
	"djinn-ci.com/namespace"
	"djinn-ci.com/queue"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type Image struct {
	ID          int64
	UserID      int64
	AuthorID    int64
	NamespaceID sql.NullInt64
	Driver      driver.Type
	Hash        string
	Name        string
	CreatedAt   time.Time

	Author    *user.User
	User      *user.User
	Download  *Download
	Namespace *namespace.Namespace
}

var _ database.Model = (*Image)(nil)

func Relations(db database.Pool) []database.RelationFunc {
	rels := namespace.ResourceRelations(db)

	return append(rels, database.Relation("id", "image_id", DownloadStore{Pool: db}))
}

func LoadNamespaces(db database.Pool, ii ...*Image) error {
	mm := make([]database.Model, 0, len(ii))

	for _, i := range ii {
		mm = append(mm, i)
	}

	if err := namespace.Load(db, mm...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func LoadRelations(db database.Pool, ii ...*Image) error {
	mm := make([]database.Model, 0, len(ii))

	for _, i := range ii {
		mm = append(mm, i)
	}

	if err := database.LoadRelations(mm, Relations(db)...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (i *Image) path() string {
	return filepath.Join(i.Driver.String(), i.Hash)
}

func (i *Image) Data(store fs.Store) (fs.Record, error) {
	store, err := store.Partition(i.UserID)

	if err != nil {
		return nil, errors.Err(err)
	}

	rec, err := store.Open(i.path())

	if err != nil {
		return nil, errors.Err(err)
	}
	return rec, nil
}

func (i *Image) Dest() []interface{} {
	return []interface{}{
		&i.ID,
		&i.UserID,
		&i.AuthorID,
		&i.NamespaceID,
		&i.Driver,
		&i.Hash,
		&i.Name,
		&i.CreatedAt,
	}
}

func (i *Image) Bind(m database.Model) {
	switch v := m.(type) {
	case *Download:
		if i.ID == v.ImageID {
			i.Download = v
		}
	case *user.User:
		i.Author = v

		if i.UserID == v.ID {
			i.User = v
		}
	case *namespace.Namespace:
		if i.NamespaceID.Int64 == v.ID {
			i.Namespace = v
		}
	}
}

func (i *Image) Downloaded() bool {
	if i.Download != nil {
		return !i.Download.Error.Valid && i.Download.FinishedAt.Valid
	}
	return true
}

func (i *Image) JSON(addr string) map[string]interface{} {
	if i == nil {
		return nil
	}

	json := map[string]interface{}{
		"id":           i.ID,
		"author_id":    i.AuthorID,
		"user_id":      i.UserID,
		"namespace_id": nil,
		"driver":       i.Driver.String(),
		"name":         i.Name,
		"created_at":   i.CreatedAt.Format(time.RFC3339),
		"url":          addr + i.Endpoint(),
	}

	if i.Author != nil {
		json["author"] = i.Author.JSON(addr)
	}

	if i.User != nil {
		json["user"] = i.User.JSON(addr)
	}

	if i.NamespaceID.Valid {
		json["namespace_id"] = i.NamespaceID.Int64

		if i.Namespace != nil {
			json["namespace"] = i.Namespace.JSON(addr)
		}
	}

	if i.Download != nil {
		download := map[string]interface{}{
			"source":      i.Download.Source.String(),
			"error":       nil,
			"created_at":  i.Download.CreatedAt.Format(time.RFC3339),
			"started_at":  nil,
			"finished_at": nil,
		}

		if i.Download.Error.Valid {
			download["error"] = i.Download.Error.String
		}
		if i.Download.StartedAt.Valid {
			download["started_at"] = i.Download.StartedAt.Time.Format(time.RFC3339)
		}
		if i.Download.FinishedAt.Valid {
			download["finished_at"] = i.Download.FinishedAt.Time.Format(time.RFC3339)
		}
		json["download"] = download
	}
	return json
}

func (i *Image) Endpoint(uri ...string) string {
	if len(uri) > 0 {
		return "/images/" + strconv.FormatInt(i.ID, 10) + "/" + strings.Join(uri, "/")
	}
	return "/images/" + strconv.FormatInt(i.ID, 10)
}

func (i *Image) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":           i.ID,
		"user_id":      i.UserID,
		"author_id":    i.AuthorID,
		"namespace_id": i.NamespaceID,
		"driver":       i.Driver,
		"hash":         i.Hash,
		"name":         i.Name,
		"created_at":   i.CreatedAt,
	}
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

	data := map[string]interface{}{
		"image":  ev.Image.JSON(env.DJINN_API_SERVER),
		"action": ev.Action,
	}
	return errors.Err(ev.dis.Dispatch(event.New(ev.Image.NamespaceID, event.Images, data)))
}

var _ queue.Job = (*Event)(nil)

type Store struct {
	database.Pool
	fs.Store

	Hasher *crypto.Hasher
}

var (
	_ database.Loader = (*Store)(nil)

	table = "images"
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
	Driver    driver.Type
	Image     io.Reader
}

func (s *Store) Create(p Params) (*Image, error) {
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

	hash, err := s.Hasher.HashNow()

	if err != nil {
		return nil, errors.Err(err)
	}

	i := Image{
		UserID:      userId,
		AuthorID:    p.UserID,
		NamespaceID: namespaceId,
		Driver:      p.Driver,
		Hash:        hash,
		Name:        p.Name,
		CreatedAt:   time.Now(),
		Namespace:   n,
	}

	if p.Image != nil {
		store, err := s.Partition(i.UserID)

		if err != nil {
			return nil, errors.Err(err)
		}

		dst, err := store.Create(i.path())

		if err != nil {
			return nil, errors.Err(err)
		}

		defer dst.Close()

		if _, err := io.Copy(dst, p.Image); err != nil {
			return nil, errors.Err(err)
		}
	}

	q := query.Insert(
		table,
		query.Columns("user_id", "author_id", "namespace_id", "driver", "hash", "name", "created_at"),
		query.Values(i.UserID, i.AuthorID, i.NamespaceID, i.Driver, i.Hash, i.Name, i.CreatedAt),
		query.Returning("id"),
	)

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&i.ID); err != nil {
		return nil, errors.Err(err)
	}
	return &i, nil
}

func (s *Store) Chown(from, to int64) error {
	if err := database.Chown(s.Pool, table, from, to); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s *Store) Delete(i *Image) error {
	q := query.Delete(table, query.Where("id", "=", query.Arg(i.ID)))

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}

	store, err := s.Partition(i.UserID)

	if err != nil {
		return errors.Err(err)
	}

	if err := store.Remove(i.path()); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return errors.Err(err)
		}
	}
	return nil
}

func (s *Store) Get(opts ...query.Option) (*Image, bool, error) {
	var i Image

	ok, err := s.Pool.Get(table, &i, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &i, ok, nil
}

func (s *Store) All(opts ...query.Option) ([]*Image, error) {
	ii := make([]*Image, 0)

	new := func() database.Model {
		i := &Image{}
		ii = append(ii, i)
		return i
	}

	if err := s.Pool.All(table, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return ii, nil
}

func (s *Store) Paginate(page, limit int64, opts ...query.Option) (database.Paginator, error) {
	paginator, err := s.Pool.Paginate(table, page, limit, opts...)

	if err != nil {
		return paginator, errors.Err(err)
	}
	return paginator, nil
}

func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Image, database.Paginator, error) {
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

	ii, err := s.All(append(
		opts,
		query.OrderAsc("name"),
		query.Limit(paginator.Limit),
		query.Offset(paginator.Offset),
	)...)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}
	return ii, paginator, nil
}

func (s *Store) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	ii, err := s.All(query.Where(pk, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	for _, i := range ii {
		for _, m := range mm {
			m.Bind(i)
		}
	}
	return nil
}
