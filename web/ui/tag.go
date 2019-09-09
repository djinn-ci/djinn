package ui

import (
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/template"

	"github.com/gorilla/mux"
)

type Tag struct {
	web.Handler
}

func (h Tag) Store(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars["build"], 10, 64)

	b, err := u.BuildStore().Find(id)

	if err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	f := &form.Tag{}

	if err := form.Unmarshal(f, r); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if len(f.Tags) == 0 {
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	tags := b.TagStore()
	tt := make([]*model.Tag, len(f.Tags), len(f.Tags))

	for i, name := range f.Tags {
		t := tags.New()
		t.UserID = u.ID
		t.Name = name

		tt[i] = t
	}

	if err := tags.Create(tt...); err != nil {
		log.Error.Println(errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func (h Tag) Destroy(w http.ResponseWriter, r *http.Request) {
	u := h.User(r)

	vars := mux.Vars(r)

	buildId, _ := strconv.ParseInt(vars["build"], 10, 64)

	b, err := u.BuildStore().Find(buildId)

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to delete tag: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	tagId, err := strconv.ParseInt(vars["tag"], 10, 64)

	if err != nil {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	tags := b.TagStore()

	t, err := tags.Find(tagId)

	if err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to delete tag: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	if t.IsZero() {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if err := tags.Delete(t); err != nil {
		log.Error.Println(errors.Err(err))
		h.FlashAlert(w, r, template.Danger("Failed to delete tag: " + errors.Cause(err).Error()))
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
		return
	}

	h.FlashAlert(w, r, template.Success("Tag has been deleted: " + t.Name))

	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}
