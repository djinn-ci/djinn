package handler

import (
	"net/http"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/object"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/variable"
	"github.com/andrewpillar/thrall/web"

	"github.com/go-redis/redis"

	"github.com/RichardKnop/machinery/v1"
)

// Build is the base handler for handling the incoming HTTP requests for each
// route. This will perform the basic handling of requests for validating,
// storing, and retrieving of data.
type Build struct {
	web.Handler

	Block     *crypto.Block
	Loaders   *database.Loaders
	Objects   *object.Store
	Variables *variable.Store
	Client    *redis.Client
	Hasher    *crypto.Hasher
	Queues    map[string]*machinery.Server
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

	err = h.Objects.Load("id", database.MapKey("object_id", mm), database.Bind("object_id", "id", mm...))
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

	err = h.Variables.Load("id", database.MapKey("variable_id", mm), database.Bind("variable_id", "id", mm...))
	return vv, errors.Err(err)
}

// ShowWithRelations will retrieve the *build.Build model from the given
// request, if any, and load in all of the relationships for that model, and
// the nested relationships for those relationships, such as the stage jobs,
// namespace user.
func (h Build) ShowWithRelations(r *http.Request) (*build.Build, error) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		return nil, errors.New("failed to get build from context")
	}

	if err := build.LoadRelations(h.Loaders, b); err != nil {
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

// IndexWithRelations returns a paginated slice of Builds for the given request
// with all of the Build relations loaded. This uses the user from the given
// request to bind to a new build.Store, along with the request URL values to
// get the Builds.
func (h Build) IndexWithRelations(r *http.Request) ([]*build.Build, database.Paginator, error) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, database.Paginator{}, errors.New("no user in request context")
	}

	bb, paginator, err := build.NewStore(h.DB, u).Index(r.URL.Query())

	if err != nil {
		return bb, paginator, errors.Err(err)
	}

	if err := build.LoadRelations(h.Loaders, bb...); err != nil {
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

// StoreModel creates a new build in the database and returns it. The given
// http.Request is decoded to retrieve information about the build being stored.
func (h Build) StoreModel(r *http.Request) (*build.Build, build.Form, error) {
	f := build.Form{}

	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, f, errors.New("failed to get user from request context")
	}

	if err := form.UnmarshalAndValidate(&f, r); err != nil {
		return nil, f, errors.Err(err)
	}

	if _, ok := h.Queues[f.Manifest.Driver["type"]]; !ok {
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

// Kill the build from the given request.
func (h Build) Kill(r *http.Request) error {
	b, ok := build.FromContext(r.Context())

	if !ok {
		return errors.New("failed to get build from context")
	}
	return errors.Err(b.Kill(h.Client))
}
