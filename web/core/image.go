package core

import (
	"database/sql"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/model/types"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/sessions"
)

type Image struct {
	web.Handler

	Namespaces model.NamespaceStore
	Images     model.ImageStore
	FileStore  filestore.FileStore
	Limit      int64
}

func (h Image) Image(r *http.Request) *model.Image {
	val := r.Context().Value("image")

	i, _ := val.(*model.Image)

	return i
}

func (h Image) Destroy(r *http.Request) error {
	i := h.Image(r)

	if err := h.Images.Delete(i); err != nil {
		return errors.Err(err)
	}

	err := h.FileStore.Remove(filepath.Join(i.Driver.String(), i.Name + "::" + i.Hash))

	return errors.Err(err)
}

func (h Image) Index(images model.ImageStore, r *http.Request, opts ...query.Option) ([]*model.Image, model.Paginator, error) {
	index := []query.Option{
		model.Search("name", r.URL.Query().Get("search")),
	}

	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	paginator, err := images.Paginate(page, append(index, opts...)...)

	if err != nil {
		return []*model.Image{}, paginator, errors.Err(err)
	}

	ii, err := images.All(append(index, opts...)...)

	if err != nil {
		return ii, paginator, errors.Err(err)
	}

	if err := images.LoadNamespaces(ii); err != nil {
		return ii, paginator, errors.Err(err)
	}

	nn := make([]*model.Namespace, 0, len(ii))

	for _, i := range ii {
		if i.Namespace != nil {
			nn = append(nn, i.Namespace)
		}
	}

	err = h.Namespaces.LoadUsers(nn)

	return ii, paginator, errors.Err(err)
}

func (h Image) Show(r *http.Request) (*model.Image, error) {
	i := h.Image(r)

	if err := i.LoadNamespace(); err != nil {
		return i, errors.Err(err)
	}

	err := i.Namespace.LoadUser()

	return i, errors.Err(err)
}

func (h Image) Store(w http.ResponseWriter, r *http.Request, sess *sessions.Session) (*model.Image, error) {
	u := h.User(r)

	images := u.ImageStore()

	f := &form.Image{
		Upload: form.Upload{
			Writer:  w,
			Request: r,
			Limit:   h.Limit,
		},
		User:   u,
		Images: images,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); ok {
			return &model.Image{}, form.ErrValidation
		}
		return &model.Image{}, errors.Err(err)
	}

	n, err := h.Namespaces.Get(query.Where("path", "=", f.Namespace))

	if err != nil {
		return &model.Image{}, errors.Err(err)
	}

	defer f.Upload.File.Close()

	hash, err := crypto.HashNow()

	if err != nil {
		return &model.Image{}, errors.Err(err)
	}

	dst, err := h.FileStore.OpenFile(
		filepath.Join("qemu", f.Name + "::" + hash),
		os.O_CREATE|os.O_WRONLY,
		os.FileMode(0755),
	)

	if err != nil {
		return &model.Image{}, errors.Err(err)
	}

	defer dst.Close()

	if _, err := io.Copy(dst, f.Upload.File); err != nil {
		return &model.Image{}, errors.Err(err)
	}

	i := images.New()
	i.NamespaceID = sql.NullInt64{
		Int64: n.ID,
		Valid: n.ID > 0,
	}
	i.Driver = types.Qemu
	i.Hash = hash
	i.Name = f.Name

	err = images.Create(i)

	return i, errors.Err(err)
}
