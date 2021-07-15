package handler

import (
	"net/http"
	"os"
	"strings"

	"djinn-ci.com/build"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/namespace"
	"djinn-ci.com/object"
	"djinn-ci.com/user"
	"djinn-ci.com/web"

	"github.com/andrewpillar/webutil"
)

// Object is the base handler that provides shared logic for the UI and API
// handlers for object creation, and management.
type Object struct {
	web.Handler

	loaders *database.Loaders
	hasher  *crypto.Hasher
	store   fs.Store
	limit   int64
}

func New(h web.Handler, hasher *crypto.Hasher, store fs.Store, limit int64) Object {
	loaders := database.NewLoaders()
	loaders.Put("author", h.Users)
	loaders.Put("user", h.Users)
	loaders.Put("namespace", namespace.NewStore(h.DB))
	loaders.Put("build_tag", build.NewTagStore(h.DB))
	loaders.Put("build_trigger", build.NewTriggerStore(h.DB))

	return Object{
		Handler: h,
		loaders: loaders,
		hasher:  hasher,
		store:   store,
		limit:   limit,
	}
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

	if err := object.LoadRelations(h.loaders, oo...); err != nil {
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
	ctx := r.Context()

	f := object.Form{}

	u, ok := user.FromContext(ctx)

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	store, err := h.store.Partition(u.ID)

	if err != nil {
		return nil, f, errors.Err(err)
	}

	objects := object.NewStoreWithBlockStore(h.DB, store, u)

	f.File = webutil.NewFile("file", h.limit, r)
	f.Resource = namespace.Resource{
		Author:     u,
		Namespaces: namespace.NewStore(h.DB, u),
	}
	f.Objects = objects

	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		q := r.URL.Query()

		f.Resource.Namespace = q.Get("namespace")
		f.Name = q.Get("name")
	} else {
		if err := webutil.Unmarshal(&f, r); err != nil {
			return nil, f, errors.Err(err)
		}
	}

	if err := f.Validate(); err != nil {
		return nil, f, errors.Err(err)
	}

	defer func() {
		// File was written to disk because it was too big for memory, so make
		// sure it is removed to free up space.
		if v, ok := f.File.File.(*os.File); ok {
			os.RemoveAll(v.Name())
		}
	}()

	hash, err := h.hasher.HashNow()

	if err != nil {
		return nil, f, errors.Err(err)
	}

	o, err := objects.Create(f.Author.ID, f.Name, hash, f.File)

	if err != nil {
		return nil, f, errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &object.Event{
		Object: o,
		Action: "created",
	})
	return o, f, nil
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

	if err := object.LoadRelations(h.loaders, o); err != nil {
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
	ctx := r.Context()

	o, ok := object.FromContext(ctx)

	if !ok {
		return errors.New("failed to get object from context")
	}

	store, err := h.store.Partition(o.UserID)

	if err != nil {
		return errors.Err(err)
	}

	if err := object.NewStoreWithBlockStore(h.DB, store).Delete(o.ID, o.Hash); err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &object.Event{
		Object: o,
		Action: "deleted",
	})
	return nil
}
