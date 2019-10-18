package model

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/types"

	"github.com/andrewpillar/query"
)

type Image struct {
	Model

	UserID      int64         `db:"user_id"`
	NamespaceID sql.NullInt64 `db:"namespace_id"`
	Driver      types.Driver  `db:"driver"`
	Hash        string        `db:"hash"`
	Name        string        `db:"name"`

	User      *User
	Namespace *Namespace
}

type ImageStore struct {
	Store

	User      *User
	Namespace *Namespace
}

func imageToInterface(ii []*Image) func(i int) Interface {
	return func(i int) Interface {
		return ii[i]
	}
}

func (i Image) LoadNamespace() error {
	var err error

	namespaces := NamespaceStore{
		Store: Store{
			DB: i.DB,
		},
	}

	i.Namespace, err = namespaces.Find(i.NamespaceID.Int64)

	return errors.Err(err)
}

func (i Image) UIEndpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/images/%v", i.ID)

	if len(uri) > 0 {
		endpoint = fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}

	return endpoint
}

func (i Image) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      i.UserID,
		"namespace_id": i.NamespaceID,
		"driver":       i.Driver,
		"hash":         i.Hash,
		"name":         i.Name,
		"created_at":   i.CreatedAt,
		"updated_at":   i.UpdatedAt,
	}
}

func (s ImageStore) All(opts ...query.Option) ([]*Image, error) {
	ii := make([]*Image, 0)

	opts = append([]query.Option{ForCollaborator(s.User), ForNamespace(s.Namespace)}, opts...)

	err := s.Store.All(&ii, ImageTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, i := range ii {
		i.DB = s.DB
		i.User = s.User
		i.Namespace = s.Namespace
	}

	return ii, errors.Err(err)
}

func (s ImageStore) Create(ii ...*Image) error {
	models := interfaceSlice(len(ii), imageToInterface(ii))

	return errors.Err(s.Store.Create(ImageTable, models...))
}

func (s ImageStore) Delete(ii ...*Image) error {
	models := interfaceSlice(len(ii), imageToInterface(ii))

	return errors.Err(s.Store.Delete(ImageTable, models...))
}

func (s ImageStore) Find(id int64) (*Image, error) {
	i, err := s.findBy("id", id)

	return i, errors.Err(err)
}

func (s ImageStore) findBy(col string, val interface{}) (*Image, error) {
	i := &Image{
		Model: Model{
			DB: s.DB,
		},
		User:      s.User,
		Namespace: s.Namespace,
	}

	q := query.Select(
		query.Columns("*"),
		query.From(ImageTable),
		query.Where(col, "=", val),
		ForUser(s.User),
		ForNamespace(s.Namespace),
	)

	err := s.Get(i, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return i, errors.Err(err)
}

func (s ImageStore) FindByName(name string) (*Image, error) {
	i, err := s.findBy("name", name)

	return i, errors.Err(err)
}

func (s ImageStore) loadNamespace(ii []*Image) func(i int, n *Namespace) {
	return func(i int, n *Namespace) {
		img := ii[i]

		if img.NamespaceID.Int64 == n.ID {
			img.Namespace = n
		}
	}
}

func (s ImageStore) LoadNamespaces(ii []*Image) error {
	if len(ii) == 0 {
		return nil
	}

	models := interfaceSlice(len(ii), imageToInterface(ii))

	namespaces := NamespaceStore{
		Store: s.Store,
	}

	err := namespaces.Load(mapKey("namespace_id", models), s.loadNamespace(ii))

	return errors.Err(err)
}

func (s ImageStore) New() *Image {
	i := &Image{
		Model: Model{
			DB: s.DB,
		},
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		i.UserID = s.User.ID
	}

	if s.Namespace != nil {
		i.NamespaceID = sql.NullInt64{
			Int64: s.Namespace.ID,
			Valid: true,
		}
	}

	return i
}

func (s ImageStore) Paginate(page int64, opts ...query.Option) (Paginator, error) {
	paginator, err := s.Store.Paginate(ImageTable, page, opts...)

	return paginator, errors.Err(err)
}
