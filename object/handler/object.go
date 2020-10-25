package handler

import (
	"net/http"
	"strings"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/object"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"
)

// Object is the base handler that provides shared logic for the UI and API
// handlers for object creation, and management.
type Object struct {
	web.Handler

	// Loaders are the relationship loaders to use for loading the
	// relationships we need when working with objects.
	Loaders *database.Loaders

	// Objects is the store used for deletion of objects.
	Objects *object.Store

	// Builds is the build store for retrieving the builds an object has been
	// placed on to.
	Builds *build.Store

	// Hasher is the hashing mechanism to use when generating hashes for
	// objects.
	Hasher *crypto.Hasher

	// BlockStore is the block store implementation to use for storing objects 
	// that are uploaded.
	BlockStore block.Store

	// Limit is the maximum size of images that can be uploaded.
	Limit int64
}

// IndexWithRelations retrieves a slice of *object.Object models for the user in
// the given request context. All of the relations for each object will be
// loaded into each model we have. If any of the objects have a bound namespace,
// then the namespace's user will be loaded too. A database.Paginator will also
// be returned if there are multiple pages of objects.
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

// StoreModel unmarshals the request's data into an object, validates it and
// stores it in the database. Upon success this will return the newly created
// object. This also returns the form for creating an object.
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

// ShowWithRelations retrieves the *object.Object model from the context of the
// given request. All of the relations for the object will be loaded into the
// model we have. If the object has a namespace bound to it, then the
// namespace's user will be loaded to the namespace.
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

// DeleteModel removes the object in the given request context from the
// database and the underlying block store.
func (h Object) DeleteModel(r *http.Request) error {
	o, ok := object.FromContext(r.Context())

	if !ok {
		return errors.New("failed to get object from context")
	}
	return errors.Err(h.Objects.Delete(o.ID, o.Hash))
}
