package ui

import (
	"crypto/md5"
	"crypto/sha256"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/object"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type Object struct {
	web.Handler

	Limit     int64
	FileStore filestore.FileStore
}

func (h Object) object(r *http.Request) (*model.Object, map[string]string, error) {
	vars := mux.Vars(r)

	u, err := h.User(r)

	if err != nil {
		return &model.Object{}, vars, errors.Err(err)
	}

	id, err := strconv.ParseInt(vars["object"], 10, 64)

	if err != nil {
		return &model.Object{}, vars, nil
	}

	o, err := u.ObjectStore().Find(id)

	if err != nil {
		return &model.Object{}, vars, errors.Err(err)
	}

	if o.DeletedAt != nil && o.DeletedAt.Valid {
		return &model.Object{}, vars, nil
	}

	return o, vars, nil
}

func (h Object) Index(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	search := r.URL.Query().Get("search")

	oo, err := u.ObjectStore().Index(model.Search("name", search))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &object.IndexPage{
		Page: template.Page{
			URI: r.URL.Path,
		},
		CSRF:    csrf.TemplateField(r),
		Objects: oo,
		Search:  search,
	}

	d := template.NewDashboard(p, r.URL.Path)

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Object) Show(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)

	id, err := strconv.ParseInt(vars["object"], 10, 64)

	if err != nil {
		web.HTML(w, "Not found", http.StatusNotFound)
		return
	}

	status := r.URL.Query().Get("status")

	o, err := u.ObjectStore().Show(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	bb, err := u.BuildStore().All(
		model.WhereInQuery("id",
				model.Select(
				model.Columns("build_id"),
				model.Table("build_objects"),
				model.WhereEq("object_id", o.ID),
			),
		),
	)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &object.ShowPage{
		Page:   template.Page{
			URI: r.URL.Path,
		},
		Object: o,
		Status: status,
		Builds: bb,
	}

	d := template.NewDashboard(p, r.URL.Path)

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Object) Create(w http.ResponseWriter, r *http.Request) {
	p := &object.CreatePage{
		Form: template.Form{
			CSRF:   csrf.TemplateField(r),
			Errors: h.Errors(w, r),
			Form:   h.Form(w, r),
		},
	}

	d := template.NewDashboard(p, r.URL.Path)

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Object) Store(w http.ResponseWriter, r *http.Request) {
	u, err := h.User(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	f := &form.Object{
		Writer:  w,
		Request: r,
		Limit:   h.Limit,
	}

	if err := h.ValidateForm(f, w, r); err != nil {
		if _, ok := err.(form.Errors); ok {
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	defer f.File.Close()

	hash, err := crypto.HashNow()

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	hmd5 := md5.New()
	hsha256 := sha256.New()

	tee := io.TeeReader(f.File, io.MultiWriter(hmd5, hsha256))

	dst, err := h.FileStore.OpenFile(hash, os.O_CREATE|os.O_WRONLY, os.FileMode(0755))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	defer dst.Close()

	n, err := io.Copy(dst, tee)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	o := u.ObjectStore().New()
	o.Name = f.Name
	o.Hash = hash
	o.Type = f.Info.Header.Get("Content-Type")
	o.Size = n
	o.MD5 = hmd5.Sum(nil)
	o.SHA256 = hsha256.Sum(nil)

	if err := o.Create(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/objects", http.StatusSeeOther)
}

func (h Object) Download(w http.ResponseWriter, r *http.Request) {
	o, vars, err := h.object(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if o.IsZero() || o.Name != vars["name"] {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	f, err := h.FileStore.Open(o.Hash)

	if err != nil {
		if os.IsNotExist(err) {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	defer f.Close()

	http.ServeContent(w, r, o.Name, *o.UpdatedAt, f)
}

func (h Object) Destroy(w http.ResponseWriter, r *http.Request) {
	o, _, err := h.object(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := o.Destroy(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := h.FileStore.Remove(o.Hash); err != nil {
		log.Error.Println(errors.Err(err))
	}

	http.Redirect(w, r, "/objects", http.StatusSeeOther)
}
