package handler

import (
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/block"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/image"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"
)

type Image struct {
	web.Handler

	Loaders    *database.Loaders
	Images     *image.Store
	Hasher     *crypto.Hasher
	BlockStore block.Store
	Limit      int64
}

func (h Image) IndexWithRelations(r *http.Request) ([]*image.Image, database.Paginator, error) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, database.Paginator{}, errors.New("user not in request context")
	}

	ii, paginator, err := image.NewStore(h.DB, u).Index(r.URL.Query())

	if err != nil {
		return ii, paginator, errors.Err(err)
	}

	if err := image.LoadRelations(h.Loaders, ii...); err != nil {
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

func (h Image) ShowWithRelations(r *http.Request) (*image.Image, error) {
	i, ok := image.FromContext(r.Context())

	if !ok {
		return nil, errors.New("image not in request context")
	}

	if err := image.LoadRelations(h.Loaders, i); err != nil {
		return i, errors.Err(err)
	}

	err := h.Users.Load(
		"id",
		[]interface{}{i.Namespace.Values()["user_id"]},
		database.Bind("user_id", "id", i.Namespace),
	)
	return i, errors.Err(err)
}

func (h Image) StoreModel(w http.ResponseWriter, r *http.Request) (*image.Image, image.Form, error) {
	f := image.Form{}

	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, f, errors.New("no user in request context")
	}

	images := image.NewStoreWithBlockStore(h.DB, h.BlockStore, u)

	f.File = form.File{
		Writer:  w,
		Request: r,
		Limit:   h.Limit,
	}
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

	i, err := images.Create(hash, f.Name, driver.QEMU, f.File)
	return i, f, errors.Err(err)
}

func (h Image) DeleteModel(r *http.Request) error {
	i, ok := image.FromContext(r.Context())

	if !ok {
		return errors.New("failed to get image from context")
	}
	return errors.Err(h.Images.Delete(i.ID, i.Driver, i.Hash))
}
