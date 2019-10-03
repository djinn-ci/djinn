package core

import (
	"database/sql"
	"net/http"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"
)

type Key struct {
	web.Handler

	Namespaces model.NamespaceStore
	Keys       model.KeyStore
}

func (h Key) Key(r *http.Request) *model.Key {
	val := r.Context().Value("key")

	k, _ := val.(*model.Key)

	return k
}

func (h Key) Index(keys model.KeyStore, r *http.Request, opts ...query.Option) ([]*model.Key, error) {
	index := []query.Option{
		model.Search("name", r.URL.Query().Get("search")),
	}

	kk, err := keys.All(append(index, opts...)...)

	if err != nil {
		return kk, errors.Err(err)
	}

	if err := keys.LoadNamespaces(kk); err != nil {
		return kk, errors.Err(err)
	}

	nn := make([]*model.Namespace, 0, len(kk))

	for _, k := range kk {
		if k.Namespace != nil {
			nn = append(nn, k.Namespace)
		}
	}

	err = h.Namespaces.LoadUsers(nn)

	return kk, errors.Err(err)
}

func (h Key) Store(w http.ResponseWriter, r *http.Request) (*model.Key, error) {
	u := h.User(r)

	keys := u.KeyStore()
	namespaces := u.NamespaceStore()

	f := &form.Key{
		Keys: keys,
	}

	if err := form.Unmarshal(f, r); err != nil {
		return &model.Key{}, errors.Err(err)
	}

	n := &model.Namespace{}

	if f.Namespace != "" {
		n, err := namespaces.FindOrCreate(f.Namespace)

		if err != nil {
			return &model.Key{}, errors.Err(err)
		}

		f.Keys = n.KeyStore()
	}

	if err := f.Validate(); err != nil {
		if ferr, ok := err.(form.Errors); ok {
			h.FlashErrors(w, r, ferr)
			h.FlashForm(w, r, f)

			return &model.Key{}, ErrValidationFailed
		}

		return &model.Key{}, errors.Err(err)
	}

	enc, err := crypto.Encrypt([]byte(f.Priv))

	if err != nil {
		return &model.Key{}, errors.Err(err)
	}

	k := keys.New()
	k.NamespaceID = sql.NullInt64{
		Int64: n.ID,
		Valid: n.ID > 0,
	}
	k.Name = f.Name
	k.Key = []byte(enc)
	k.Config = f.Config

	err = keys.Create(k)

	return k, errors.Err(err)
}

func (h Key) Update(w http.ResponseWriter, r *http.Request) (*model.Key, error) {
	u := h.User(r)
	k := h.Key(r)

	namespaces := u.NamespaceStore()

	f := &form.Key{
		Keys: h.Keys,
	}

	if err := form.Unmarshal(f, r); err != nil {
		return &model.Key{}, errors.Err(err)
	}

	if f.Namespace != "" {
		n, err := namespaces.FindOrCreate(f.Namespace)

		if err != nil {
			return &model.Key{}, errors.Err(err)
		}

		k.NamespaceID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}
	}

	k.Config = f.Config

	err := h.Keys.Update(k)

	return k, errors.Err(err)
}
