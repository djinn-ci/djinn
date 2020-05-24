package handler

import (
	"net/http"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/object"
	"github.com/andrewpillar/thrall/web"
)

type Object struct {
	web.Handler

	Objects *object.Store
}

func (h Object) IndexWithRelations(r *http.Request) ([]*build.Object, error) {
	b := Model(r)

	oo, err := build.NewObjectStore(h.DB, b).All()

	if err != nil {
		return oo, errors.Err(err)
	}

	mm := model.Slice(len(oo), build.ObjectModel(oo))

	err = h.Objects.Load("id", model.MapKey("object_id", mm), model.Bind("object_id", "id", mm...))
	return oo, errors.Err(err)
}
