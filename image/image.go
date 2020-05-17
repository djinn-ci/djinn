// Package image provides the model.Model implementation for the Image entity.
package image

import (
	"database/sql"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Image struct {
	ID          int64         `db:"id"`
	UserID      int64         `db:"user_id"`
	NamespaceID sql.NullInt64 `db:"namespace_id"`
	Driver      driver.Type   `db:"driver"`
	Hash        string        `db:"hash"`
	Name        string        `db:"name"`
	CreatedAt   time.Time     `db:"created_at"`

	User      *user.User           `db:"-"`
	Namespace *namespace.Namespace `db:"-"`
}

type Store struct {
	model.Store

	User      *user.User
	Namespace *namespace.Namespace
}

var (
	_ model.Model  = (*Image)(nil)
	_ model.Loader = (*Store)(nil)
	_ model.Binder = (*Store)(nil)

	table     = "images"
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

func LoadRelations(loaders model.Loaders, ii ...*Image) error {
	mm := model.Slice(len(ii), Model(ii))
	return model.LoadRelations(relations, loaders, mm...)
}

func Model(ii []*Image) func(int) model.Model {
	return func(i int) model.Model {
		return ii[i]
	}
}

func (i *Image) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			i.User = m.(*user.User)
		case *namespace.Namespace:
			i.Namespace = m.(*namespace.Namespace)
		}
	}
}

func (i *Image) SetPrimary(id int64) {
	i.ID = id
}

func (i *Image) Primary() (string, int64) {
	return "id", i.ID
}

func (i *Image) IsZero() bool {
	return i == nil || i.ID == 0 &&
		i.UserID == 0 &&
		!i.NamespaceID.Valid &&
		i.Driver == driver.Type(0) &&
		i.Hash == "" &&
		i.Name == "" &&
		i.CreatedAt == time.Time{}
}

func (i *Image) Endpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/images/%v", i.ID)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
}

func (i *Image) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      i.UserID,
		"namespace_id": i.NamespaceID,
		"driver":       i.Driver,
		"hash":         i.Hash,
		"name":         i.Name,
		"created_at":   i.CreatedAt,
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

func (s *Store) Create(ii ...*Image) error {
	models := model.Slice(len(ii), Model(ii))
	return errors.Err(s.Store.Create(table, models...))
}

func (s *Store) Delete(ii ...*Image) error {
	models := model.Slice(len(ii), Model(ii))
	return errors.Err(s.Store.Delete(table, models...))
}

func (s *Store) All(opts ...query.Option) ([]*Image, error) {
	ii := make([]*Image, 0)

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.All(&ii, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, i := range ii {
		i.User = s.User
		i.Namespace = s.Namespace
	}
	return ii, errors.Err(err)
}

func (s *Store) Index(vals url.Values, opts ...query.Option) ([]*Image, model.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		model.Search("name", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return []*Image{}, paginator, errors.Err(err)
	}

	ii, err := s.All(append(
		opts,
		query.OrderAsc("name"),
		query.Limit(model.PageLimit),
		query.Offset(paginator.Offset),
	)...)
	return ii, paginator, errors.Err(err)
}

func (s *Store) Load(key string, vals []interface{}, fn model.LoaderFunc) error {
	ii, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for j := range vals {
		for _, i := range ii {
			fn(j, i)
		}
	}
	return nil
}

func (s *Store) Get(opts ...query.Option) (*Image, error) {
	i := &Image{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(i, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return i, errors.Err(err)
}

func (s *Store) New() *Image {
	i := &Image{
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		_, id := s.User.Primary()
		i.UserID = id
	}

	if s.Namespace != nil {
		_, id := s.Namespace.Primary()
		i.NamespaceID = sql.NullInt64{
			Int64: id,
			Valid: true,
		}
	}
	return i
}

func (s *Store) Paginate(page int64, opts ...query.Option) (model.Paginator, error) {
	paginator, err := s.Store.Paginate(table, page, opts...)
	return paginator, errors.Err(err)
}
