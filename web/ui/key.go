package ui

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/key"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type Key struct {
	web.Handler
}

func (h Key) Index(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	search := r.URL.Query().Get("search")

	kk, err := u.KeyStore().Index(model.Search("name", search))

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &key.IndexPage{
		BasePage: template.BasePage{
			URI:  r.URL.Path,
			User: u,
		},
		CSRF:   string(csrf.TemplateField(r)),
		Search: search,
		Keys:   kk,
	}

	d := template.NewDashboard(p, r.URL.Path, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Key) Create(w http.ResponseWriter, r *http.Request) {
	p := &key.Form{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Errors(w, r),
			Fields: h.Form(w, r),
		},
	}

	d := template.NewDashboard(p, r.URL.Path, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Key) Store(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	keys := u.KeyStore()
	namespaces := u.NamespaceStore()

	f := &form.Key{
		Keys: keys,
	}

	if err := form.Unmarshal(f, r); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create SSH key: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	n := &model.Namespace{}

	if f.Namespace != "" {
		n, err := namespaces.FindOrCreate(f.Namespace)

		if err != nil {
			log.Error.Println(errors.Err(err))
			h.FlashAlert(w, r, template.Danger("Failed to create SSH key: " + errors.Cause(err).Error()))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		f.Keys = n.KeyStore()
	}

	if err := f.Validate(); err != nil {
		if _, ok := err.(form.Errors); ok {
			h.FlashErrors(w, r, err.(form.Errors))
			h.FlashForm(w, r, f)
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create SSH key: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	enc, err := crypto.Encrypt([]byte(f.Priv))

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create SSH key: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	k := keys.New()
	k.NamespaceID = sql.NullInt64{
		Int64: n.ID,
		Valid: n.ID > 0,
	}
	k.Name = f.Name
	k.Key = []byte(enc)
	k.Config = f.Config

	if err := keys.Create(k); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to create SSH key: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Key has been added: " + k.Name))

	http.Redirect(w, r, "/keys", http.StatusSeeOther)
}

func (h Key) Edit(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars["key"], 10, 64)

	k, err := u.KeyStore().Find(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := k.LoadNamespace(); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &key.Form{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: h.Errors(w, r),
			Fields: h.Form(w, r),
		},
		Key: k,
	}

	d := template.NewDashboard(p, r.URL.Path, h.Alert(w, r))

	web.HTML(w, template.Render(d), http.StatusOK)
}

func (h Key) Update(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars["key"], 10, 64)

	keys := u.KeyStore()
	namespaces := u.NamespaceStore()

	k, err := keys.Find(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to update SSH key: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	f := &form.Key{
		Keys: keys,
	}

	if err := form.Unmarshal(f, r); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to update SSH key: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if f.Namespace != "" {
		n, err := namespaces.FindOrCreate(f.Namespace)

		if err != nil {
			log.Error.Println(errors.Err(err))
			h.FlashAlert(w, r, template.Danger("Failed to create SSH key: " + errors.Cause(err).Error()))
			http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
			return
		}

		k.NamespaceID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}
	}

	k.Config = f.Config

	if err := keys.Update(k); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to update SSH key: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Key changes saved for: " + k.Name))

	http.Redirect(w, r, "/keys", http.StatusSeeOther)
}

func (h Key) Destroy(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars["key"], 10, 64)

	keys := u.KeyStore()

	k, err := keys.Find(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to delete SSH key: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if err := keys.Delete(k); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to delete SSH key: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Key has been deleted: " + k.Name))

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
