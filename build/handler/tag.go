package handler

import (
	"net/http"
	"strconv"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/gorilla/mux"
)

// Tag is the base handler that provides shared logic for the UI and API
// handlers for working with build tags.
type Tag struct {
	web.Handler

	// Loaders are the relationship loaders to use for loading the
	// relationships we need when working with build tags.
	Loaders *database.Loaders
}

// StoreModel unmarshals the request's data into build tags, validates it and
// stores it in the database. Upon success this will return the newly created
// build tags.
func (h Tag) StoreModel(r *http.Request) ([]*build.Tag, error) {
	ctx := r.Context()

	u, ok := user.FromContext(ctx)

	if !ok {
		return nil, errors.New("failed to get user from context")
	}

	b, ok := build.FromContext(ctx)

	if !ok {
		return nil, errors.New("failed to get build from context")
	}

	f := &build.TagForm{}

	if err := form.Unmarshal(f, r); err != nil {
		return nil, errors.Err(err)
	}

	tt, err := build.NewTagStore(h.DB, b).Create(u.ID, f.Tags...)
	return tt, errors.Err(err)
}

func (h Tag) DeleteModel(r *http.Request) error {
	id, _ := strconv.ParseInt(mux.Vars(r)["tag"], 10, 64)
	return errors.Err(build.NewTagStore(h.DB).Delete(id))
}
