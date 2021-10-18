package handler

import (
	"net/http"

	"djinn-ci.com/build"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/user"
	"djinn-ci.com/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/mux"
)

// Tag is the base handler that provides shared logic for the UI and API
// handlers for working with build tags.
type Tag struct {
	web.Handler

	loaders *database.Loaders
}

func NewTag(h web.Handler) Tag {
	loaders := database.NewLoaders()
	loaders.Put("user", user.NewStore(h.DB))
	loaders.Put("namespace", namespace.NewStore(h.DB))
	loaders.Put("build_tag", build.NewTagStore(h.DB))
	loaders.Put("build_trigger", build.NewTriggerStore(h.DB))

	return Tag{
		Handler: h,
		loaders: loaders,
	}
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

	if err := webutil.Unmarshal(f, r); err != nil {
		return nil, errors.Err(err)
	}

	tt, err := build.NewTagStore(h.DB, b).Create(u.ID, f.Tags...)

	if err != nil {
		return nil, errors.Err(err)
	}

	b.Trigger, err = build.NewTriggerStore(h.DB, b).Get(query.Where("build_id", "=", query.Arg(b.ID)))

	if err != nil {
		return nil, errors.Err(err)
	}

	h.Queues.Produce(ctx, "events", &build.TagEvent{
		Build: b,
		User:  u,
		Tags:  tt,
	})
	return tt, nil
}

func (h Tag) DeleteModel(r *http.Request) error {
	b, ok := build.FromContext(r.Context())

	if !ok {
		return errors.New("failed to get build from context")
	}
	return errors.Err(build.NewTagStore(h.DB).Delete(b.ID, mux.Vars(r)["name"]))
}
