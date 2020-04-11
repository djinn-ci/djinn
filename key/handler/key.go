package handler

import (
	"database/sql"
	"net/http"
	"net/url"
	"strings"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/key"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/sessions"
)

type Key struct {
	web.Handler

	Loaders    model.Loaders
	Namespaces namespace.Store
	Keys       key.Store
}

func (h Key) Model(r *http.Request) *key.Key {
	val := r.Context().Value("key")
	k, _ := val.(*key.Key)
	return k
}

func (h Key) IndexWithRelations(s key.Store, vals url.Values) ([]*key.Key, model.Paginator, error) {
	kk, paginator, err := s.Index(vals)

	if err != nil {
		return kk, paginator, errors.Err(err)
	}

	if err := key.LoadRelations(h.Loaders, kk...); err != nil {
		return kk, paginator, errors.Err(err)
	}

	nn := make([]model.Model, 0, len(kk))

	for _, k := range kk {
		if k.Namespace != nil {
			nn = append(nn, k.Namespace)
		}
	}

	err = h.Users.Load("id", model.MapKey("user_id", nn), model.Bind("user_id", "id", nn...))
	return kk, paginator, errors.Err(err)
}

func (h Key) ShowWithRelations(r *http.Request) (*key.Key, error) {
	k := h.Model(r)

	if err := key.LoadRelations(h.Loaders, k); err != nil {
		return k, errors.Err(err)
	}

	err := h.Users.Load(
		"id",
		[]interface{}{k.Namespace.Values()["user_id"]},
		model.Bind("user_id", "id", k.Namespace),
	)
	return k, errors.Err(err)
}

func (h Key) StoreModel(r *http.Request, sess *sessions.Session) (*key.Key, error) {
	u := h.User(r)

	keys := key.NewStore(h.DB, u)

	f := &key.Form{
		ResourceForm: namespace.ResourceForm{
			User:          u,
			Namespaces:    namespace.NewStore(h.DB, u),
			Collaborators: namespace.NewCollaboratorStore(h.DB),
		},
		Keys: keys,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		return &key.Key{}, errors.Err(err)
	}

	if f.Namespace != "" {
		n, err := h.Namespaces.GetByPath(f.Namespace)

		if err != nil {
			return &key.Key{}, errors.Err(err)
		}

		if !n.CanAdd(u) {
			return &key.Key{}, namespace.ErrPermission
		}
		keys.Bind(n)
	}

	enc, err := crypto.Encrypt([]byte(f.PrivateKey))

	if err != nil {
		return &key.Key{}, errors.Err(err)
	}

	k := keys.New()
	k.Name = strings.Replace(f.Name, " ", "_", - 1)
	k.Key = enc
	k.Config = f.Config

	err = keys.Create(k)
	return k, errors.Err(err)
}

func (h Key) UpdateModel(r *http.Request, sess *sessions.Session) (*key.Key, error) {
	u := h.User(r)
	k := h.Model(r)

	keys := key.NewStore(h.DB, u)

	f := &key.Form{
		ResourceForm: namespace.ResourceForm{
			User:          u,
			Namespaces:    namespace.NewStore(h.DB, u),
			Collaborators: namespace.NewCollaboratorStore(h.DB),
		},
		Keys: keys,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		return &key.Key{}, errors.Err(err)
	}

	if f.Namespace != "" {
		n, err := h.Namespaces.GetByPath(f.Namespace)

		if err != nil {
			return &key.Key{}, errors.Err(err)
		}

		if !n.CanAdd(u) {
			return &key.Key{}, namespace.ErrPermission
		}

		if n.ID != k.NamespaceID.Int64 {
			k.NamespaceID = sql.NullInt64{
				Int64: n.ID,
				Valid: true,
			}
		}
	}

	k.Config = f.Config

	err := h.Keys.Update(k)
	return k, errors.Err(err)
}
