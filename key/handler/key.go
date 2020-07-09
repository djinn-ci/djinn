package handler

import (
	"net/http"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/key"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"
)

type Key struct {
	web.Handler

	Loaders *database.Loaders
	Block   *crypto.Block
	Keys    *key.Store
}

func (h Key) ShowWithRelations(r *http.Request) (*key.Key, error) {
	var err error

	k, ok := key.FromContext(r.Context())

	if !ok {
		return nil, errors.New("failed to get key from context")
	}

	if err := key.LoadRelations(h.Loaders, k); err != nil {
		return k, errors.Err(err)
	}

	if k.Namespace != nil {
		err = h.Users.Load(
			"id", []interface{}{k.Namespace.Values()["user_id"]}, database.Bind("user_id", "id", k.Namespace),
		)
	}
	return k, errors.Err(err)
}

func (h Key) IndexWithRelations(r *http.Request) ([]*key.Key, database.Paginator, error) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, database.Paginator{}, errors.New("user not in request context")
	}

	kk, paginator, err := key.NewStore(h.DB, u).Index(r.URL.Query())

	if err != nil {
		return kk, paginator, errors.Err(err)
	}

	if err := key.LoadRelations(h.Loaders, kk...); err != nil {
		return kk, paginator, errors.Err(err)
	}

	nn := make([]database.Model, 0, len(kk))

	for _, k := range kk {
		if k.Namespace != nil {
			nn = append(nn, k.Namespace)
		}
	}

	err = h.Users.Load("id", database.MapKey("user_id", nn), database.Bind("user_id", "id", nn...))
	return kk, paginator, errors.Err(err)
}

func (h Key) StoreModel(r *http.Request) (*key.Key, key.Form, error) {
	f := key.Form{}

	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	keys := key.NewStoreWithBlock(h.DB, h.Block, u)

	f.Resource = namespace.Resource{
		User:       u,
		Namespaces: namespace.NewStore(h.DB, u),
	}
	f.Keys = keys

	if err := form.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	k, err := keys.Create(f.Name, f.PrivateKey, f.Config)
	return k, f, errors.Err(err)
}

func (h Key) UpdateModel(r *http.Request) (*key.Key, key.Form, error) {
	f := key.Form{}

	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	k, ok := key.FromContext(r.Context())

	if !ok {
		return k, f, errors.New("no key in request context")
	}

	keys := key.NewStore(h.DB, u)

	f.Resource = namespace.Resource{
		User:       u,
		Namespaces: namespace.NewStore(h.DB, u),
	}
	f.Key = k
	f.Keys = keys

	if err := form.UnmarshalAndValidate(&f, r); err != nil {
		return k, f, errors.Err(err)
	}

	namespaceId := k.NamespaceID.Int64

	if f.Namespace != "" {
		n, err := namespace.NewStore(h.DB).GetByPath(f.Namespace)

		if err != nil {
			return k, f, errors.Err(err)
		}
		namespaceId = n.ID
	}

	err := keys.Update(k.ID, namespaceId, f.Config)
	return k, f, errors.Err(err)
}

func (h Key) DeleteModel(r *http.Request) error {
	k, ok := key.FromContext(r.Context())

	if !ok {
		return errors.New("no key in request context")
	}
	return errors.Err(h.Keys.Delete(k.ID))
}
