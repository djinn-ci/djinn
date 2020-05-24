package handler

import (
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
)

type Tag struct {
	web.Handler
}

func (h Tag) StoreModel(r *http.Request) ([]*build.Tag, error) {
	u := h.User(r)
	b := Model(r)

	f := &build.TagForm{}

	if err := form.Unmarshal(f, r); err != nil {
		return []*build.Tag{}, errors.Err(err)
	}

	if len(f.Tags) == 0 {
		return []*build.Tag{}, nil
	}

	tags := build.NewTagStore(h.DB, b)
	tt := make([]*build.Tag, 0, len(f.Tags))

	for _, name := range f.Tags {
		t := tags.New()
		t.UserID = u.ID
		t.Name = name

		tt = append(tt, t)
	}

	err := tags.Create(tt...)
	return tt, errors.Err(err)
}

func (h Tag) Delete(r *http.Request) error {
	u := h.User(r)

	vars := mux.Vars(r)

	buildId, _ := strconv.ParseInt(vars["build"], 10, 64)

	b, err := build.NewStore(h.DB, u).Get(query.Where("id", "=", buildId))

	if err != nil {
		return errors.Err(err)
	}

	tagId, _ := strconv.ParseInt(vars["tag"], 10, 64)

	tags := build.NewTagStore(h.DB, b)

	t, err := tags.Get(query.Where("id", "=", tagId))

	if err != nil {
		return errors.Err(err)
	}

	err = tags.Delete(t)
	return errors.Err(err)
}
