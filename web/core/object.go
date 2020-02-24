package core

import (
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/sessions"
)

type Object struct {
	web.Handler

	Namespaces model.NamespaceStore
	Objects    model.ObjectStore
	FileStore  filestore.FileStore
	Limit      int64
}

func (h Object) Object(r *http.Request) *model.Object {
	val := r.Context().Value("object")

	o, _ := val.(*model.Object)

	return o
}

func (h Object) Destroy(r *http.Request) error {
	o := h.Object(r)

	if err := h.Objects.Delete(o); err != nil {
		return errors.Err(err)
	}

	err := h.FileStore.Remove(o.Hash)

	return errors.Err(err)
}

func (h Object) Index(objects model.ObjectStore, r *http.Request, opts ...query.Option) ([]*model.Object, model.Paginator, error) {
	index := []query.Option{
		model.Search("name", r.URL.Query().Get("search")),
	}

	page, err := strconv.ParseInt(r.URL.Query().Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	paginator, err := objects.Paginate(page, append(index, opts...)...)

	if err != nil {
		return []*model.Object{}, paginator, errors.Err(err)
	}

	oo, err := objects.All(append(index, opts...)...)

	if err != nil {
		return oo, paginator, errors.Err(err)
	}

	if err := objects.LoadNamespaces(oo); err != nil {
		return oo, paginator, errors.Err(err)
	}

	nn := make([]*model.Namespace, 0, len(oo))

	for _, o := range oo {
		if o.Namespace != nil {
			nn = append(nn, o.Namespace)
		}
	}

	err = h.Namespaces.LoadUsers(nn)

	return oo, paginator, errors.Err(err)
}

func (h Object) Show(r *http.Request) (*model.Object, error) {
	o := h.Object(r)

	if err := o.LoadNamespace(); err != nil {
		return o, errors.Err(err)
	}

	err := o.Namespace.LoadUser()

	return o, errors.Err(err)
}

func (h Object) Store(w http.ResponseWriter, r *http.Request, sess *sessions.Session) (*model.Object, error) {
	u := h.User(r)

	objects := u.ObjectStore()

	f := &form.Object{
		Upload: form.Upload{
			Writer:     w,
			Request:    r,
			Limit:      h.Limit,
			Disallowed: []string{
				"application/gzip",
				"application/zip",
				"application/x-zip-compressed",
				"application/x-rar-compressed",
				"multipart/x-zip",
			},
		},
		User:    u,
		Objects: objects,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		if _, ok := err.(form.Errors); ok {
			return &model.Object{}, form.ErrValidation
		}
		return &model.Object{}, errors.Err(err)
	}

	n, err := h.Namespaces.Get(query.Where("path", "=", f.Namespace))

	if err != nil {
		return &model.Object{}, errors.Err(err)
	}

	defer f.Upload.File.Close()

	hash, err := crypto.HashNow()

	if err != nil {
		return &model.Object{}, errors.Err(err)
	}

	md5_ := md5.New()
	sha256_ := sha256.New()

	tee := io.TeeReader(f.Upload.File, io.MultiWriter(md5_, sha256_))

	dst, err := h.FileStore.OpenFile(hash, os.O_CREATE|os.O_WRONLY, os.FileMode(0755))

	if err != nil {
		return &model.Object{}, errors.Err(err)
	}

	defer dst.Close()

	size, err := io.Copy(dst, tee)

	if err != nil {
		return &model.Object{}, errors.Err(err)
	}

	o := objects.New()
	o.NamespaceID = sql.NullInt64{
		Int64: n.ID,
		Valid: n.ID > 0,
	}
	o.Name = f.Name
	o.Hash = hash
	o.Type = f.Info.Header.Get("Content-Type")
	o.Size = size
	o.MD5 = md5_.Sum(nil)
	o.SHA256 = sha256_.Sum(nil)

	err = objects.Create(o)

	return o, errors.Err(err)
}
