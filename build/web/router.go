package web

import (
	"context"
	"net/http"
	"strconv"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/build/handler"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/image"
	"github.com/andrewpillar/thrall/key"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/object"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/server"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/variable"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"

	"github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"

	"github.com/RichardKnop/machinery/v1"
)

type Router struct {
	build handler.Build

	Middleware web.Middleware
	Artifacts  filestore.FileStore
	Redis      *redis.Client
	Queues     map[string]*machinery.Server
	Providers  map[string]oauth2.Provider
}

var _ server.Router = (*Router)(nil)

func Gate(db *sqlx.DB) web.Gate {
	namespaces := namespace.NewStore(db)
	users := user.NewStore(db)

	return func(u *user.User, r *http.Request) (*http.Request, bool, error) {
		vars := mux.Vars(r)

		owner, err := users.Get(query.Where("username", "=", vars["username"]))

		if err != nil {
			return r, false, errors.Err(err)
		}

		id, _ := strconv.ParseInt(vars["build"], 10, 64)

		b, err := build.NewStore(db, owner).Get(query.Where("id", "=", id))

		if err != nil {
			return r, false, errors.Err(err)
		}

		if b.IsZero() {
			return r, false, nil
		}

		r = r.WithContext(context.WithValue(r.Context(), "build", b))

		if !b.NamespaceID.Valid {
			return r, u.ID == b.UserID, nil
		}

		root, err := namespaces.Get(
			query.WhereQuery("root_id", "=", namespace.SelectRootID(b.NamespaceID.Int64)),
			query.WhereQuery("id", "=", namespace.SelectRootID(b.NamespaceID.Int64)),
		)

		if err != nil {
			return r, false, errors.Err(err)
		}

		cc, err := namespace.NewCollaboratorStore(db, root).All()

		if err != nil {
			return r, false, errors.Err(err)
		}

		root.LoadCollaborators(cc)
		return r, root.AccessibleBy(u), nil
	}
}

func (r *Router) Init(h web.Handler) {
	users := user.NewStore(h.DB)
	namespaces := namespace.NewStore(h.DB)
	tags := build.NewTagStore(h.DB)
	triggers := build.NewTriggerStore(h.DB)

	loaders := model.NewLoaders()
	loaders.Put("user", users)
	loaders.Put("namespace", namespaces)
	loaders.Put("build_tag", tags)
	loaders.Put("build_trigger", triggers)

	r.build = handler.Build{
		Handler:         h,
		Loaders:         loaders,
		Builds:          build.NewStore(h.DB),
		Tags:            tags,
		Triggers:        triggers,
		Stages:          build.NewStageStore(h.DB),
		Artifacts:       build.NewArtifactStore(h.DB),
		Keys:            key.NewStore(h.DB),
		Namespaces:      namespaces,
		Objects:         object.NewStore(h.DB),
		Providers:       provider.NewStore(h.DB),
		Images:          image.NewStore(h.DB),
		Variables:       variable.NewStore(h.DB),
		FileStore:       r.Artifacts,
		Client:          r.Redis,
		Queues:          r.Queues,
		Oauth2Providers: r.Providers,
	}
}

func (r *Router) RegisterUI(mux *mux.Router, csrf func(http.Handler) http.Handler, gates ...web.Gate) {
	build := handler.UI{Build: r.build}
	hook := handler.Hook{Build: r.build}

	mux.HandleFunc("/hook/github", hook.Github).Methods("POST")
	mux.HandleFunc("/hook/gitlab", hook.Gitlab).Methods("POST")

	auth := mux.PathPrefix("/").Subrouter()
	auth.HandleFunc("/", build.Index).Methods("GET")
	auth.HandleFunc("/builds/create", build.Create).Methods("GET")
	auth.HandleFunc("/builds", build.Store).Methods("POST")
	auth.Use(r.Middleware.Auth, csrf)

	sr := mux.PathPrefix("/b/{username}/{build:[0-9]+}").Subrouter()
	sr.HandleFunc("", build.Show).Methods("GET")
	sr.HandleFunc("", build.Kill).Methods("DELETE")
	sr.HandleFunc("/manifest", build.Show).Methods("GET")
	sr.HandleFunc("/manifest/raw", build.Show).Methods("GET")
	sr.HandleFunc("/output/raw", build.Show).Methods("GET")
	sr.HandleFunc("/objects", build.Show).Methods("GET")
	sr.HandleFunc("/variables", build.Show).Methods("GET")
	sr.HandleFunc("/keys", build.Show).Methods("GET")
	sr.HandleFunc("/jobs/{job:[0-9]+}", build.JobShow).Methods("GET")
	sr.HandleFunc("/jobs/{job:[0-9]+}/output/raw", build.JobShow).Methods("GET")
	sr.HandleFunc("/artifacts", build.Show).Methods("GET")
	sr.HandleFunc("/artifacts/{artifact:[0-9]+}/download/{name}", build.ArtifactShow).Methods("GET")
	sr.HandleFunc("/tags", build.Show).Methods("GET")
	sr.HandleFunc("/tags", build.TagStore).Methods("POST")
	sr.HandleFunc("/tags/{tag:[0-9]+}", build.TagDestroy).Methods("DELETE")
	sr.Use(r.Middleware.Gate(gates...), csrf)
}

func (r *Router) RegisterAPI(mux *mux.Router, gates ...web.Gate) {}
