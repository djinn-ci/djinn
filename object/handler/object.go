package handler

import (
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/block"
	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/object"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"
)

type Object struct {
	web.Handler

	Loaders    *database.Loaders
	Objects    *object.Store
	Builds     *build.Store
	Hasher     *crypto.Hasher
	BlockStore block.Store
	Limit      int64
}

func (h Object) IndexWithRelations(r *http.Request) ([]*object.Object, database.Paginator, error) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, database.Paginator{}, errors.New("no user in request context")
	}

	oo, paginator, err := object.NewStore(h.DB, u).Index(r.URL.Query())

	if err != nil {
		return oo, paginator, errors.Err(err)
	}

	if err := object.LoadRelations(h.Loaders, oo...); err != nil {
		return oo, paginator, errors.Err(err)
	}

	nn := make([]database.Model, 0, len(oo))

	for _, o := range oo {
		if o.Namespace != nil {
			nn = append(nn, o.Namespace)
		}
	}

	err = h.Users.Load("id", database.MapKey("user_id", nn), database.Bind("user_id", "id", nn...))
	return oo, paginator, errors.Err(err)
}

func (h Object) ShowWithRelations(r *http.Request) (*object.Object, error) {
	var err error

	o, ok := object.FromContext(r.Context())

	if !ok {
		return o, errors.New("no object in request context")
	}

	if err := object.LoadRelations(h.Loaders, o); err != nil {
		return o, errors.Err(err)
	}

	if o.Namespace != nil {
		err = h.Users.Load(
			"id", []interface{}{o.Namespace.Values()["user_id"]}, database.Bind("user_id", "id", o.Namespace),
		)
	}
	return o, errors.Err(err)
}

func (h Object) StoreModel(w http.ResponseWriter, r *http.Request) (*object.Object, object.Form, error) {
	f := object.Form{}

	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	objects := object.NewStoreWithBlockStore(h.DB, h.BlockStore, u)

	f.File = form.File{
		Writer:  w,
		Request: r,
		Limit:   h.Limit,
	}
	f.Resource = namespace.Resource{
		User:       u,
		Namespaces: namespace.NewStore(h.DB, u),
	}
	f.Objects = objects

	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		q := r.URL.Query()

		f.Resource.Namespace = q.Get("namespace")
		f.Name = q.Get("name")
	} else {
		if err := form.Unmarshal(&f, r); err != nil {
			return nil, f, errors.Err(err)
		}
	}

	if err := f.Validate(); err != nil {
		return nil, f, errors.Err(err)
	}

	hash, err := h.Hasher.HashNow()

	if err != nil {
		return nil, f, errors.Err(err)
	}

	o, err := objects.Create(f.Name, hash, f.MIMEType, f.File)
	return o, f, errors.Err(err)
}

func (h Object) DeleteModel(r *http.Request) error {
	o, ok := object.FromContext(r.Context())

	if !ok {
		return errors.New("failed to get object from context")
	}
	return errors.Err(h.Objects.Delete(o.ID, o.Hash))
}
