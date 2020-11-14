package handler

import (
	"net/http"
	"strings"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/driver"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/image"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/webutil"
)

// Image is the base handler that provides shared logic for the UI and API
// handlers for image creation, and management.
type Image struct {
	web.Handler

	loaders *database.Loaders
	hasher  *crypto.Hasher
	store   block.Store
	limit   int64
}

func New(h web.Handler, hasher *crypto.Hasher, store block.Store, limit int64) Image {
	loaders := database.NewLoaders()
	loaders.Put("user", h.Users)
	loaders.Put("namespace", namespace.NewStore(h.DB))

	return Image{
		Handler: h,
		loaders: loaders,
		hasher:  hasher,
		store:   store,
		limit:   limit,
	}
}

// IndexWithRelations retrieves a slice of *image.Image models for the user in
// the given request context. All of the relations for each image will be loaded
// into each model we have. If any of the images have a bound namespace, then
// the namespace's user will be loaded too. A database.Paginator will also be
// returned if there are multiple pages of images.
func (h Image) IndexWithRelations(r *http.Request) ([]*image.Image, database.Paginator, error) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, database.Paginator{}, errors.New("user not in request context")
	}

	ii, paginator, err := image.NewStore(h.DB, u).Index(r.URL.Query())

	if err != nil {
		return ii, paginator, errors.Err(err)
	}

	if err := image.LoadRelations(h.loaders, ii...); err != nil {
		return ii, paginator, errors.Err(err)
	}

	nn := make([]database.Model, 0, len(ii))

	for _, i := range ii {
		if i.Namespace != nil {
			nn = append(nn, i.Namespace)
		}
	}

	err = h.Users.Load("id", database.MapKey("user_id", nn), database.Bind("user_id", "id", nn...))
	return ii, paginator, errors.Err(err)
}

// StoreModel unmarshals the request's data into an image, validates it and
// stores it in the database. Upon success this will return the newly created
// image. This also returns the form for creating an image.
func (h Image) StoreModel(w http.ResponseWriter, r *http.Request) (*image.Image, image.Form, error) {
	f := image.Form{}

	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	images := image.NewStoreWithBlockStore(h.DB, h.store, u)

	f.File = webutil.NewFile("file", h.limit, w, r)
	f.Resource = namespace.Resource{
		User:       u,
		Namespaces: namespace.NewStore(h.DB, u),
	}
	f.Images = images

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

	hash, err := h.hasher.HashNow()

	if err != nil {
		return nil, f, errors.Err(err)
	}

	i, err := images.Create(hash, f.Name, driver.QEMU, f.File)
	return i, f, errors.Err(err)
}

// ShowWithRelations retrieves the *image.Image model from the context of the
// given request. All of the relations for the image will be loaded into the
// model we have. If the image has a namespace bound to it, then the
// namespace's user will be loaded to the namespace.
func (h Image) ShowWithRelations(r *http.Request) (*image.Image, error) {
	i, ok := image.FromContext(r.Context())

	if !ok {
		return nil, errors.New("image not in request context")
	}

	if err := image.LoadRelations(h.loaders, i); err != nil {
		return i, errors.Err(err)
	}

	err := h.Users.Load(
		"id",
		[]interface{}{i.Namespace.Values()["user_id"]},
		database.Bind("user_id", "id", i.Namespace),
	)
	return i, errors.Err(err)
}

// DeleteModel removes the image in the given request context from the database
// and the underlying block store.
func (h Image) DeleteModel(r *http.Request) error {
	i, ok := image.FromContext(r.Context())

	if !ok {
		return errors.New("failed to get image from context")
	}
	return errors.Err(image.NewStore(h.DB).Delete(i.ID, i.Driver, i.Hash))
}
