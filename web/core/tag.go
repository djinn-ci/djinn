package core

import (
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/mux"
)

type Tag struct {
	web.Handler
}

func (h Tag) Store(w http.ResponseWriter, r *http.Request) ([]*model.Tag, error) {
	u := h.User(r)

	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars["build"], 10, 64)

	b, err := u.BuildStore().Find(id)

	if err != nil {
		return []*model.Tag{}, errors.Err(err)
	}

	f := &form.Tag{}

	if err := form.Unmarshal(f, r); err != nil {
		return []*model.Tag{}, errors.Err(err)
	}

	if len(f.Tags) == 0 {
		return []*model.Tag{}, nil
	}

	tags := b.TagStore()
	tt := make([]*model.Tag, 0, len(f.Tags))

	for _, name := range f.Tags {
		t := tags.New()
		t.UserID = u.ID
		t.Name = name

		tt = append(tt, t)
	}

	err = tags.Create(tt...)

	return tt, errors.Err(err)
}

func (h Tag) Destroy(r *http.Request) error {
	u := h.User(r)

	vars := mux.Vars(r)

	buildId, _ := strconv.ParseInt(vars["build"], 10, 64)

	b, err := u.BuildStore().Find(buildId)

	if err != nil {
		return errors.Err(err)
	}

	tagId, _ := strconv.ParseInt(vars["tag"], 10, 64)

	tags := b.TagStore()

	t, err := tags.Find(tagId)

	if err != nil {
		return errors.Err(err)
	}

	err = tags.Delete(t)

	return errors.Err(err)
}
