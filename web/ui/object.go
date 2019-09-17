package ui

import (
	"crypto/md5"
	"crypto/sha256"
	"database/sql"
	"io"
	"net/http"
	"os"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/object"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type Object struct {
	web.Handler

	Build     Build
	Limit     int64
	Objects   model.ObjectStore
	FileStore filestore.FileStore
}

func (h Object) Object(r *http.Request) *model.Object {
	val := r.Context().Value("object")

	o, _ := val.(*model.Object)

	return o
}

func (h Object) indexPage(objects model.ObjectStore, r *http.Request) (object.IndexPage, error) {
	u := h.User(r)

	search := r.URL.Query().Get("search")

	oo, err := objects.Index(model.Search("name", search))

	if err != nil {
		return object.IndexPage{}, errors.Err(err)
	}

	p := object.IndexPage{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:    string(csrf.TemplateField(r)),
		Objects: oo,
		Search:  search,
	}

	return p, nil
}

func (h Object) Index(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	p, err := h.indexPage(u.ObjectStore(), r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	d := template.NewDashboard(&p, r.URL, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Object) Show(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)
	o := h.Object(r)

	if err := o.LoadNamespace(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := o.Namespace.LoadUser(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	bp, err := h.Build.indexPage(u.BuildStore(), r, query.WhereQuery("id", "IN",
		query.Select(
			query.Columns("build_id"),
			query.From("build_objects"),
			query.Where("object_id", "=", o.ID),
		),
	))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &object.ShowPage{
		BasePage: template.BasePage{
			URL: r.URL,
		},
		Object: o,
		Index:  bp,
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Object) Create(w http.ResponseWriter, r *http.Request) {
	p := &object.CreatePage{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Errors(w, r),
			Fields: h.Form(w, r),
		},
	}

	d := template.NewDashboard(p, r.URL, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Object) Store(w http.ResponseWriter, r *http.Request) {
	var err error

	u := h.User(r)

	objects := u.ObjectStore()
	namespaces := u.NamespaceStore()

	f := &form.Object{
		Writer:  w,
		Request: r,
		Limit:   h.Limit,
		Objects: objects,
	}

	if err := form.Unmarshal(f, r); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create object: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	n := &model.Namespace{}

	if f.Namespace != "" {
		n, err = namespaces.FindOrCreate(f.Namespace)

		if !n.CanAdd(u) {
			log.Error.Println(errors.Err(err))
			h.FlashAlert(w, r, template.Danger("Failed to create object: Namespace does not exist"))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		if err != nil {
			log.Error.Println(errors.Err(err))
			h.FlashAlert(w, r, template.Danger("Failed to create object: " + errors.Cause(err).Error()))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		f.Objects = n.ObjectStore()
	}

	if err := f.Validate(); err != nil {
		if _, ok := err.(form.Errors); ok {
			h.FlashErrors(w, r, err.(form.Errors))
			h.FlashForm(w, r, f)
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create object: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	defer f.File.Close()

	hash, err := crypto.HashNow()

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create object: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	hmd5 := md5.New()
	hsha256 := sha256.New()

	tee := io.TeeReader(f.File, io.MultiWriter(hmd5, hsha256))

	dst, err := h.FileStore.OpenFile(hash, os.O_CREATE|os.O_WRONLY, os.FileMode(0755))

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create object: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	defer dst.Close()

	size, err := io.Copy(dst, tee)

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create object: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
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
	o.MD5 = hmd5.Sum(nil)
	o.SHA256 = hsha256.Sum(nil)

	if err := objects.Create(o); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create object: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Object has been added: " + o.Name))

	http.Redirect(w, r, "/objects", http.StatusSeeOther)
}

func (h Object) Download(w http.ResponseWriter, r *http.Request) {
	o := h.Object(r)

	vars := mux.Vars(r)

	if o.Name != vars["name"] {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	f, err := h.FileStore.Open(o.Hash)

	if err != nil {
		if os.IsNotExist(errors.Cause(err)) {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	defer f.Close()

	http.ServeContent(w, r, o.Name, o.UpdatedAt, f)
}

func (h Object) Destroy(w http.ResponseWriter, r *http.Request) {
	o := h.Object(r)

	if err := h.Objects.Delete(o); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to delete object: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if err := h.FileStore.Remove(o.Hash); err != nil {
		if !os.IsNotExist(errors.Cause(err)) {
			log.Error.Println(errors.Err(err))
		}
	}

	h.FlashAlert(w, r, template.Success("Object has been deleted: " + o.Name))

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
