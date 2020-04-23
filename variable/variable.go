package variable

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

type Variable struct {
	ID          int64         `db:"id"`
	UserID      int64         `db:"user_id"`
	NamespaceID sql.NullInt64 `db:"namespace_id"`
	Key         string        `db:"key"`
	Value       string        `db:"value"`
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
	_ model.Model  = (*Variable)(nil)
	_ model.Binder = (*Store)(nil)
	_ model.Loader = (*Store)(nil)

	table     = "variables"
	relations = map[string]model.RelationFunc{
		"namespace": model.Relation("namespace_id", "id"),
	}
)

func Model(vv []*Variable) func(int) model.Model {
	return func(i int) model.Model {
		return vv[i]
	}
}

func NewStore(db *sqlx.DB, mm ...model.Model) Store {
	s := Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func LoadRelations(loaders model.Loaders, vv ...*Variable) error {
	mm := model.Slice(len(vv), Model(vv))
	return errors.Err(model.LoadRelations(relations, loaders, mm...))
}

func (v *Variable) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			v.User = m.(*user.User)
		case *namespace.Namespace:
			v.Namespace = m.(*namespace.Namespace)
		}
	}
}

func (v *Variable) Kind() string { return "variable" }

func (v *Variable) SetPrimary(i int64) {
	v.ID = i
}

func (v *Variable) Primary() (string, int64) {
	return "id", v.ID
}

func (v *Variable) Endpoint(uri ...string) string {
	if v == nil {
		return ""
	}

	endpoint := fmt.Sprintf("/variables/%v", v.ID)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
}

func (v *Variable) IsZero() bool {
	return v == nil || v.ID == 0 &&
		v.UserID == 0 &&
		!v.NamespaceID.Valid &&
		v.Key == "" &&
		v.Value == "" &&
		v.CreatedAt == time.Time{}
}

func (v *Variable) Values() map[string]interface{} {
	if v == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"user_id":      v.UserID,
		"namespace_id": v.NamespaceID,
		"key":          v.Key,
		"value":        v.Value,
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

func (s Store) Create(vv ...*Variable) error {
	models := model.Slice(len(vv), Model(vv))
	return s.Store.Create(table, models...)
}

func (s Store) Delete(vv ...*Variable) error {
	models := model.Slice(len(vv), Model(vv))
	return s.Store.Delete(table, models...)
}

func (s Store) Paginate(page int64, opts ...query.Option) (model.Paginator, error) {
	paginator, err := s.Store.Paginate(table, page, opts...)
	return paginator, errors.Err(err)
}

func (s Store) New() *Variable {
	v := &Variable{
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		v.UserID = s.User.ID
	}

	if s.Namespace != nil {
		v.NamespaceID = sql.NullInt64{
			Int64: s.Namespace.ID,
			Valid: true,
		}
	}
	return v
}

func (s Store) All(opts ...query.Option) ([]*Variable, error) {
	vv := make([]*Variable, 0)

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.All(&vv, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, v := range vv {
		v.User = s.User
		v.Namespace = s.Namespace
	}
	return vv, errors.Err(err)
}

func (s Store) Index(vals url.Values, opts ...query.Option) ([]*Variable, model.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		model.Search("key", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return []*Variable{}, paginator, errors.Err(err)
	}

	vv, err := s.All(append(
		opts,
		query.OrderAsc("key"),
		query.Limit(model.PageLimit),
		query.Offset(paginator.Offset),
	)...)
	return vv, paginator, errors.Err(err)
}

func (s Store) Get(opts ...query.Option) (*Variable, error) {
	v := &Variable{
		User:      s.User,
		Namespace: s.Namespace,
	}

	opts = append([]query.Option{
		namespace.WhereCollaborator(s.User),
		model.Where(s.Namespace, "namespace_id"),
	}, opts...)

	err := s.Store.Get(v, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return v, errors.Err(err)
}

func (s Store) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	vv, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, v := range vv {
			load(i, v)
		}
	}
	return nil
}
