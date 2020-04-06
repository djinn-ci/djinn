package handler

import (
	"net/http"
	"io"
	"os"
	"path/filepath"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/driver"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/image"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/sessions"
)

type Image struct {
	web.Handler

	Loaders    model.Loaders
	Images     image.Store
	FileStore  filestore.FileStore
	Limit      int64
}

func (h Image) Model(r *http.Request) *image.Image {
	val := r.Context().Value("image")
	i, _ := val.(*image.Image)
	return i
}

func (h Image) Delete(r *http.Request) error {
	i := h.Model(r)

	if err := h.Images.Delete(i); err != nil {
		return errors.Err(err)
	}
	err := h.FileStore.Remove(filepath.Join(i.Driver.String(), i.Name+"::"+i.Hash))
	return errors.Err(err)
}

func (h Image) IndexWithRelations(s image.Store, r *http.Request, opts ...query.Option) ([]*image.Image, model.Paginator, error) {
	ii, paginator, err := s.Index(r, opts...)

	if err != nil {
		return ii, paginator, errors.Err(err)
	}

	if err := image.LoadRelations(h.Loaders, ii...); err != nil {
		return ii, paginator, errors.Err(err)
	}

	nn := make([]model.Model, 0, len(ii))

	for _, i := range ii {
		if i.Namespace != nil {
			nn = append(nn, i.Namespace)
		}
	}

	err = h.Users.Load("id", model.MapKey("user_id", nn), model.Bind("user_id", "id", nn...))
	return ii, paginator, errors.Err(err)
}

func (h Image) Get(r *http.Request) (*image.Image, error) {
	i := h.Model(r)

	if err := image.LoadRelations(h.Loaders, i); err != nil {
		return i, errors.Err(err)
	}

	err := h.Users.Load(
		"id",
		[]interface{}{i.Namespace.Values()["user_id"]},
		model.Bind("user_id", "id", i.Namespace),
	)
	return i, errors.Err(err)
}

func (h Image) realStore(s image.Store, f namespace.ResourceForm, name string, r io.Reader) (*image.Image, error) {
	if f.Namespace != "" {
		n, err := f.Namespaces.GetByPath(f.Namespace)

		if err != nil {
			return &image.Image{}, errors.Err(err)
		}

		if !n.CanAdd(f.User) {
			return &image.Image{}, namespace.ErrPermission
		}
		s.Bind(n)
	}

	hash, err := crypto.HashNow()

	if err != nil {
		return &image.Image{}, errors.Err(err)
	}

	dst, err := h.FileStore.OpenFile(
		filepath.Join("qemu", name + "::" + hash),
		os.O_CREATE|os.O_WRONLY,
		os.FileMode(0755),
	)

	if err != nil {
		return &image.Image{}, errors.Err(err)
	}

	defer dst.Close()

	if _, err := io.Copy(dst, r); err != nil {
		return &image.Image{}, errors.Err(err)
	}

	i := s.New()
	i.Driver = driver.TypeQEMU
	i.Hash = hash
	i.Name = name

	err = s.Create(i)
	return i, errors.Err(err)
}

func (h Image) StoreModel(w http.ResponseWriter, r *http.Request, sess *sessions.Session) (*image.Image, error) {
	u := h.User(r)

	images := image.NewStore(h.DB, u)

	rf := namespace.ResourceForm{
		User:          u,
		Namespaces:    namespace.NewStore(h.DB, u),
		Collaborators: namespace.NewCollaboratorStore(h.DB),
	}
	f := &image.Form{
		File: form.File{
			Writer:  w,
			Request: r,
			Limit:   h.Limit,
		},
		ResourceForm: rf,
		Images:       images,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		return &image.Image{}, errors.Err(err)
	}

	defer f.File.Close()

	i, err := h.realStore(images, rf, f.Name, f.File)
	return i, errors.Err(err)
}
