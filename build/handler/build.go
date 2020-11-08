package handler

import (
	"net/http"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/object"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/variable"
	"github.com/andrewpillar/djinn/web"

	"github.com/go-redis/redis"

	"github.com/mcmathja/curlyq"
)

// Build is the base handler that provides shared logic for the UI and API
// handlers for build creation, submission, and retrieval.
type Build struct {
	web.Handler

	artifacts block.Store
	loaders   *database.Loaders
	redis     *redis.Client
	hasher    *crypto.Hasher
	producers map[string]*curlyq.Producer
}

func New(h web.Handler, artifacts block.Store, redis *redis.Client, hasher *crypto.Hasher, producers map[string]*curlyq.Producer) Build {
	loaders := database.NewLoaders()
	loaders.Put("user", h.Users)
	loaders.Put("namespace", namespace.NewStore(h.DB))
	loaders.Put("build_tag", build.NewTagStore(h.DB))
	loaders.Put("build_trigger", build.NewTriggerStore(h.DB))
	loaders.Put("build_stage", build.NewStageStore(h.DB))

	return Build{
		Handler:   h,
		artifacts: artifacts,
		loaders:   loaders,
		redis:     redis,
		hasher:    hasher,
		producers: producers,
	}
}

func (h Build) objectsWithRelations(b *build.Build) ([]*build.Object, error) {
	oo, err := build.NewObjectStore(h.DB, b).All()

	if err != nil {
		return oo, errors.Err(err)
	}

	if len(oo) == 0 {
		return oo, nil
	}

	mm := database.ModelSlice(len(oo), build.ObjectModel(oo))

	err = object.NewStore(h.DB).Load(
		"id", database.MapKey("object_id", mm), database.Bind("object_id", "id", mm...),
	)
	return oo, errors.Err(err)
}

func (h Build) variablesWithRelations(b *build.Build) ([]*build.Variable, error) {
	vv, err := build.NewVariableStore(h.DB, b).All()

	if err != nil {
		return vv, errors.Err(err)
	}

	if len(vv) == 0 {
		return vv, nil
	}

	mm := database.ModelSlice(len(vv), build.VariableModel(vv))

	err = variable.NewStore(h.DB).Load(
		"id", database.MapKey("variable_id", mm), database.Bind("variable_id", "id", mm...),
	)
	return vv, errors.Err(err)
}

// ShowWithRelations retrieves the *build.Build model from the context of the
// given request. All of the relations for the build will be loaded into the
// model we have. If the build has a namespace bound to it, then the
// namespace's user will be loaded to the namespace.
func (h Build) ShowWithRelations(r *http.Request) (*build.Build, error) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		return nil, errors.New("failed to get build from context")
	}

	if err := build.LoadRelations(h.loaders, b); err != nil {
		return nil, errors.Err(err)
	}

	ss := make([]database.Model, 0, len(b.Stages))

	for _, s := range b.Stages {
		ss = append(ss, s)
	}

	err := build.NewJobStore(h.DB, b).Load(
		"stage_id", database.MapKey("id", ss), database.Bind("id", "stage_id", ss...),
	)

	if err != nil {
		return nil, errors.Err(err)
	}

	if b.Namespace != nil {
		err = h.Users.Load(
			"id", []interface{}{b.Namespace.Values()["user_id"]}, database.Bind("user_id", "id", b.Namespace),
		)
	}
	return b, errors.Err(err)
}

// IndexWithRelations retreives a slice of *build.Build models for the user in
// the given request context. All of the relations for each build will be
// loaded into each model we have. If any of the builds have a bound namespace,
// then the namespace's user will be loaded too. A database.Paginator will also
// be returned if there are multiple pages of builds.
func (h Build) IndexWithRelations(r *http.Request) ([]*build.Build, database.Paginator, error) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, database.Paginator{}, errors.New("no user in request context")
	}

	bb, paginator, err := build.NewStore(h.DB, u).Index(r.URL.Query())

	if err != nil {
		return bb, paginator, errors.Err(err)
	}

	if err := build.LoadRelations(h.loaders, bb...); err != nil {
		return bb, paginator, errors.Err(err)
	}

	nn := make([]database.Model, 0, len(bb))

	for _, b := range bb {
		if b.Namespace != nil {
			nn = append(nn, b.Namespace)
		}
	}

	if len(nn) > 0 {
		err = h.Users.Load("id", database.MapKey("user_id", nn), database.Bind("user_id", "id", nn...))
	}
	return bb, paginator, errors.Err(err)
}

// StoreModel unmarshals the request's data into a build, validates it and
// stores it in the database. Upon success this will return the newly created
// build. This also returns the form for creating a build.
func (h Build) StoreModel(r *http.Request) (*build.Build, build.Form, error) {
	f := build.Form{}

	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, f, errors.New("failed to get user from request context")
	}

	if err := form.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	if _, ok := h.producers[f.Manifest.Driver["type"]]; !ok {
		return nil, f, build.ErrDriver
	}

	t := &build.Trigger{
		Type:    build.Manual,
		Comment: f.Comment,
		Data: map[string]string{
			"email":    u.Email,
			"username": u.Username,
		},
	}

	tags := make([]*build.Tag, 0, len(f.Tags))

	for _, name := range f.Tags {
		tags = append(tags, &build.Tag{UserID: u.ID, Name: name})
	}

	b, err := build.NewStore(h.DB, u).Create(f.Manifest, t, f.Tags...)
	return b, f, errors.Err(err)
}

// Kill will kill the build in the given request context.
func (h Build) Kill(r *http.Request) error {
	b, ok := build.FromContext(r.Context())

	if !ok {
		return errors.New("failed to get build from context")
	}
	return errors.Err(b.Kill(h.redis))
}
