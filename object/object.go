package object

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"
)

type Object struct {
	ID          int64         `db:"id"`
	UserID      int64         `db:"user_id"`
	NamespaceID sql.NullInt64 `db:"namespace_id"`
	Hash        string        `db:"hash"`
	Name        string        `db:"name"`
	Type        string        `db:"type"`
	Size        int64         `db:"size"`
	MD5         []byte        `db:"md5"`
	SHA256      []byte        `db:"sha256"`
	CreatedAt   time.Time     `db:"created_at"`
	DeletedAt   pq.NullTime   `db:"deleted_at"`

	User      *user.User           `db:"-"`
	Namespace *namespace.Namespace `db:"-"`
}

type Store struct {
	model.Store

	User      *user.User
	Namespace *namespace.Namespace
}

var (
	_ model.Model  = (*Object)(nil)
	_ model.Binder = (*Store)(nil)
	_ model.Loader = (*Store)(nil)

	table     = "objects"
	relations = map[string]model.RelationFunc{
		"namespace": model.Relation("namespace_id", "id"),
	}
)

func NewStore(db *sqlx.DB, mm ...model.Model) Store {
	s := Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func LoadRelations(loaders model.Loaders, oo ...*Object) error {
	mm := model.Slice(len(oo), Model(oo))
	return errors.Err(model.LoadRelations(relations, loaders, mm...))
}

func Model(oo []*Object) func(int)model.Model {
	return func(i int) model.Model {
		return oo[i]
	}
}

func (o *Object) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			o.User = m.(*user.User)
		case *namespace.Namespace:
			o.Namespace = m.(*namespace.Namespace)
		}
	}
}

func (o *Object) Kind() string { return "object" }

func (o *Object) SetPrimary(id int64) {
	if o == nil {
		return
	}
	o.ID = id
}

func (o *Object) Primary() (string, int64) {
	if o == nil {
		return "id", 0
	}
	return "id", o.ID
}

func (o *Object) Endpoint(uri ...string) string {
	if o == nil {
		return ""
	}

	endpoint := fmt.Sprintf("/objects/%v", o.ID)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
}

func (o *Object) IsZero() bool {
	return o == nil || o.ID == 0 &&
		o.UserID == 0 &&
		!o.NamespaceID.Valid &&
		o.Hash == "" &&
		o.Name == "" &&
		o.Type == "" &&
		o.Size == 0 &&
		len(o.MD5) == 0 &&
		len(o.SHA256) == 0 &&
		o.CreatedAt == time.Time{} &&
		!o.DeletedAt.Valid
}

func (o *Object) Values() map[string]interface{} {
	if o == nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"user_id":      o.UserID,
		"namespace_id": o.NamespaceID,
		"hash":         o.Hash,
		"name":         o.Name,
		"type":         o.Type,
		"size":         o.Size,
		"md5":          o.MD5,
		"sha256":       o.SHA256,
		"deleted_at":   o.DeletedAt,
	}
}

func (s *Store) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		case *namespace.Namespace:
			s.Namespace = m.(*namespace.Namespace)
		}
	}
}

func (s Store) Create(oo ...*Object) error {
	models := model.Slice(len(oo), Model(oo))
	return errors.Err(s.Store.Create(table, models...))
}

func (s Store) Update(oo ...*Object) error {
	models := model.Slice(len(oo), Model(oo))
	return errors.Err(s.Store.Update(table, models...))
}

func (s Store) Delete(oo ...*Object) error {
	models := model.Slice(len(oo), Model(oo))
	return errors.Err(s.Store.Delete(table, models...))
}

func (s Store) Paginate(page int64, opts ...query.Option) (model.Paginator, error) {
	paginator, err := s.Store.Paginate(table, page, opts...)
	return paginator, errors.Err(err)
}

func (s Store) New() *Object {
	o := &Object{
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		_, id := s.User.Primary()
		o.UserID = id
	}

	if s.Namespace != nil {
		_, id := s.Namespace.Primary()
		o.NamespaceID = sql.NullInt64{
			Int64: id,
			Valid: true,
		}
	}
	return o
}

func (s Store) All(opts ...query.Option) ([]*Object, error) {
	oo := make([]*Object, 0)

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.All(&oo, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, o := range oo {
		o.User = s.User
		o.Namespace = s.Namespace
	}
	return oo, errors.Err(err)
}

func (s Store) Index(vals url.Values, opts ...query.Option) ([]*Object, model.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		model.Search("name", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return []*Object{}, paginator, errors.Err(err)
	}

	oo, err := s.All(append(
		opts,
		query.OrderAsc("name"),
		query.Limit(model.PageLimit),
		query.Offset(paginator.Offset),
	)...)
	return oo, paginator, errors.Err(err)
}

func (s Store) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	oo, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, o := range oo {
			load(i, o)
		}
	}
	return nil
}

func (s Store) Get(opts ...query.Option) (*Object, error) {
	o := &Object{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(o, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return o, errors.Err(err)
}
