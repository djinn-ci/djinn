package handler

import (
	"net/http"

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/image"
	"djinn-ci.com/namespace"
	"djinn-ci.com/user"
	"djinn-ci.com/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/schema"
)

// Image is the base handler that provides shared logic for the UI and API
// handlers for image creation, and management.
type Image struct {
	web.Handler

	loaders *database.Loaders
	hasher  *crypto.Hasher
	store   fs.Store
	limit   int64
}

func New(h web.Handler, hasher *crypto.Hasher, store fs.Store, limit int64) Image {
	loaders := database.NewLoaders()
	loaders.Put("user", h.Users)
	loaders.Put("author", h.Users)
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
	ctx := r.Context()

	f := image.Form{}

	u, ok := user.FromContext(ctx)

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	images := image.NewStore(h.DB, u)

	f.File = webutil.NewFile("file", h.limit, r)
	f.Resource = namespace.Resource{
		Author:     u,
		Namespaces: namespace.NewStore(h.DB, u),
	}
	f.Images = images

	var multierror schema.MultiError

	if r.Header.Get("Content-Type") == image.MimeTypeQEMU {
		q := r.URL.Query()

		f.Resource.Namespace = q.Get("namespace")
		f.Name = q.Get("name")
	} else {
		if err := webutil.Unmarshal(&f, r); err != nil {
			if errs, ok := err.(schema.MultiError); ok {
				multierror = errs
				goto validate
			}
			return nil, f, errors.Err(err)
		}
	}

validate:
	if err := f.Validate(); err != nil {
		if ferrs, ok := err.(*webutil.Errors); ok {
			// Merge in any errors we got from unmarshalling the form data.
			// This way when presenting the errors to the user, we can ensure
			// we get every field covered.
			for field, err := range multierror {
				if cerr, ok := err.(schema.ConversionError); ok {
					err = errors.Cause(cerr.Err)
				}
				ferrs.Put(field, err)
			}
			return nil, f, errors.Err(err)
		}
		return nil, f, errors.Err(err)
	}

	defer f.Remove()

	hash, err := h.hasher.HashNow()

	if err != nil {
		return nil, f, errors.Err(err)
	}

	store, err := h.store.Partition(f.Resource.Owner.ID)

	if err != nil {
		return nil, f, errors.Err(err)
	}

	images.SetBlockStore(store)

	// If being downloaded from a remote then f.File will be nil, and no
	// attempt will be made to actually upload an image file, this will be
	// handled as a queue job.
	i, err := images.Create(f.Author.ID, hash, f.Name, driver.QEMU, f.File.File)

	if err != nil {
		return nil, f, errors.Err(err)
	}

	if f.DownloadURL.URL != nil {
		j, err := image.NewDownloadJob(h.DB, i, f.DownloadURL)

		if err != nil {
			return nil, f, errors.Err(err)
		}

		if _, err := h.Queues.Produce(ctx, "jobs", j); err != nil {
			return nil, f, errors.Err(err)
		}
	}

	i.Author, err = user.NewStore(h.DB).Get(query.Where("id", "=", query.Arg(i.AuthorID)))

	if err != nil {
		return nil, f, errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &image.Event{
		Image:  i,
		Action: "created",
	})
	return i, f, nil
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

	d, err := image.NewDownloadStore(h.DB, i).Get()

	if err != nil {
		return i, errors.Err(err)
	}

	i.Download = d

	err = h.Users.Load(
		"id",
		[]interface{}{i.Namespace.Values()["user_id"]},
		database.Bind("user_id", "id", i.Namespace),
	)
	return i, errors.Err(err)
}

// DeleteModel removes the image in the given request context from the database
// and the underlying block store.
func (h Image) DeleteModel(r *http.Request) error {
	ctx := r.Context()

	i, ok := image.FromContext(ctx)

	if !ok {
		return errors.New("failed to get image from context")
	}

	store, err := h.store.Partition(i.UserID)

	if err != nil {
		return errors.Err(err)
	}

	if err = image.NewStoreWithBlockStore(h.DB, store).Delete(i.ID, i.Driver, i.Hash); err != nil {
		return errors.Err(err)
	}

	i.Author, err = user.NewStore(h.DB).Get(query.Where("id", "=", query.Arg(i.AuthorID)))

	if err != nil {
		return errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &image.Event{
		Image:  i,
		Action: "deleted",
	})
	return nil
}
