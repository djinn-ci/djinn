// Package key providers the model.Model implementation for the Key entity.
package key

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
)

type Key struct {
	ID          int64         `db:"id"`
	UserID      int64         `db:"user_id"`
	NamespaceID sql.NullInt64 `db:"namespace_id"`
	Name        string        `db:"name"`
	Key         []byte        `db:"key"`
	Config      string        `db:"config"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`

	User      *user.User           `db:"-"`
	Namespace *namespace.Namespace `db:"-"`
}

type Store struct {
	model.Store

	User      *user.User
	Namespace *namespace.Namespace
}

var (
	_ model.Model  = (*Key)(nil)
	_ model.Binder = (*Store)(nil)
	_ model.Loader = (*Store)(nil)

	table     = "keys"
	relations = map[string]model.RelationFunc{
		"namespace": model.Relation("namespace_id", "id"),
	}
)

func NewStore(db *sqlx.DB, mm ...model.Model) *Store {
	s := &Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func LoadRelations(loaders model.Loaders, kk ...*Key) error {
	mm := model.Slice(len(kk), Model(kk))
	return errors.Err(model.LoadRelations(relations, loaders, mm...))
}

func Model(kk []*Key) func(int)model.Model {
	return func(i int) model.Model {
		return kk[i]
	}
}

func (k *Key) Bind(mm ...model.Model) {

	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			k.User = m.(*user.User)
		case *namespace.Namespace:
			k.Namespace = m.(*namespace.Namespace)
		}
	}
}

func (k *Key) SetPrimary(id int64) {
	k.ID = id
}

func (k *Key) Primary() (string, int64) {
	return "id", k.ID
}

func (k *Key) Endpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/keys/%v", k.ID)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
}

func (k *Key) IsZero() bool {
	return k == nil || k.ID == 0 &&
		k.UserID == 0 &&
		!k.NamespaceID.Valid &&
		k.Name == "" &&
		len(k.Key) == 0 &&
		k.Config == "" &&
		k.CreatedAt == time.Time{}
}

func (k *Key) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      k.UserID,
		"namespace_id": k.NamespaceID,
		"name":         k.Name,
		"key":          k.Key,
		"config":       k.Config,
	}
}

func (s *Store) New() *Key {
	k := &Key{
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		k.UserID = s.User.ID
	}

	if s.Namespace != nil {
		k.NamespaceID = sql.NullInt64{
			Int64: s.Namespace.ID,
			Valid: true,
		}
	}
	return k
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

func (s *Store) Create(kk ...*Key) error {
	models := model.Slice(len(kk), Model(kk))
	return errors.Err(s.Store.Create(table, models...))
}

func (s *Store) Update(kk ...*Key) error {
	models := model.Slice(len(kk), Model(kk))
	return errors.Err(s.Store.Update(table, models...))
}

func (s *Store) Delete(kk ...*Key) error {
	models := model.Slice(len(kk), Model(kk))
	return errors.Err(s.Store.Delete(table, models...))
}

func (s *Store) Paginate(page int64, opts ...query.Option) (model.Paginator, error) {
	paginator, err := s.Store.Paginate(table, page, opts...)
	return paginator, errors.Err(err)
}

func (s *Store) All(opts ...query.Option) ([]*Key, error) {
	kk := make([]*Key, 0)

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.All(&kk, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, k := range kk {
		k.User = s.User
		k.Namespace = s.Namespace
	}
	return kk, errors.Err(err)
}

func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Key, model.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		model.Search("name", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return []*Key{}, paginator, errors.Err(err)
	}

	kk, err := s.All(append(
		opts,
		query.OrderAsc("key"),
		query.Limit(model.PageLimit),
		query.Offset(paginator.Offset),
	)...)
	return kk, paginator, errors.Err(err)
}

func (s *Store) Get(opts ...query.Option) (*Key, error) {
	k := &Key{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(k, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return k, errors.Err(err)
}

func (s *Store) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	kk, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, k := range kk {
			load(i, k)
		}
	}
	return nil
}
