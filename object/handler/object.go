package handler

import (
	"crypto/md5"
	"crypto/sha256"
	"io"
	"os"
	"net/http"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/object"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/sessions"
)

type Object struct {
	web.Handler

	Loaders    model.Loaders
	Objects    object.Store
	Builds     build.Store
	FileStore  filestore.FileStore
	Limit      int64
}

func (h Object) Model(r *http.Request) *object.Object {
	val := r.Context().Value("object")
	o, _ := val.(*object.Object)
	return o
}

func (h Object) Delete(r *http.Request) error {
	o := h.Model(r)

	if err := h.Objects.Delete(o); err != nil {
		return errors.Err(err)
	}
	return errors.Err(h.FileStore.Remove(o.Hash))
}

func (h Object) IndexWithRelations(s object.Store, r *http.Request) ([]*object.Object, model.Paginator, error) {
	oo, paginator, err := s.Index(r)

	if err != nil {
		return oo, paginator, errors.Err(err)
	}

	if err := object.LoadRelations(h.Loaders, oo...); err != nil {
		return oo, paginator, errors.Err(err)
	}

	nn := make([]model.Model, 0, len(oo))

	for _, o := range oo {
		if o.Namespace != nil {
			nn = append(nn, o.Namespace)
		}
	}

	err = h.Users.Load("id", model.MapKey("user_id", nn), model.Bind("user_id", "id", nn...))
	return oo, paginator, errors.Err(err)
}

func (h Object) ShowWithRelations(r *http.Request) (*object.Object, error) {
	o := h.Model(r)

	if err := object.LoadRelations(h.Loaders, o); err != nil {
		return o, errors.Err(err)
	}

	err := h.Users.Load(
		"id",
		[]interface{}{o.Namespace.Values()["user_id"]},
		model.Bind("user_id", "id", o.Namespace),
	)
	return o, errors.Err(err)
}

func (h Object) realStore(s object.Store, f namespace.ResourceForm, name, typ string, r io.Reader) (*object.Object, error) {
	if f.Namespace != "" {
		n, err := f.Namespaces.GetByPath(f.Namespace)

		if err != nil {
			return &object.Object{}, errors.Err(err)
		}

		if !n.CanAdd(f.User) {
			return &object.Object{}, namespace.ErrPermission
		}
		s.Bind(n)
	}

	hash, err := crypto.HashNow()

	if err != nil {
		return &object.Object{}, errors.Err(err)
	}

	md5 := md5.New()
	sha256 := sha256.New()

	tee := io.TeeReader(r, io.MultiWriter(md5, sha256))

	dst, err := h.FileStore.OpenFile(hash, os.O_CREATE|os.O_WRONLY, os.FileMode(0755))

	if err != nil {
		return &object.Object{}, errors.Err(err)
	}

	defer dst.Close()

	size, err := io.Copy(dst, tee)

	if err != nil {
		return &object.Object{}, errors.Err(err)
	}

	o := s.New()
	o.Name = name
	o.Hash = hash
	o.Type = typ
	o.Size = size
	o.MD5 = md5.Sum(nil)
	o.SHA256 = sha256.Sum(nil)

	err = s.Create(o)
	return o, errors.Err(err)
}

func (h Object) StoreModel(w http.ResponseWriter, r *http.Request, sess *sessions.Session) (*object.Object, error) {
	u := h.User(r)

	objects := object.NewStore(h.DB, u)

	rf := namespace.ResourceForm{
		User:          u,
		Namespaces:    namespace.NewStore(h.DB, u),
		Collaborators: namespace.NewCollaboratorStore(h.DB),
	}
	f := &object.Form{
		File: form.File{
			Writer:  w,
			Request: r,
			Limit:   h.Limit,
		},
		ResourceForm: rf,
		Objects:      objects,
	}

	if err := h.ValidateForm(f, r, sess); err != nil {
		return &object.Object{}, errors.Err(err)
	}

	defer f.File.Close()

	o, err := h.realStore(objects, rf, f.Name, f.Info.Header.Get("Content-Type"), f.File)
	return o, errors.Err(err)
}
