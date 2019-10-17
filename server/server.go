package server

import (
	nethttp "net/http"

	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/http"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/core"

	"github.com/jmoiron/sqlx"

	"github.com/gorilla/mux"

	"github.com/go-redis/redis"

	"github.com/RichardKnop/machinery/v1"
)

type Server struct {
	Http *http.Server

	router *mux.Router

	build        core.Build
	collaborator core.Collaborator
	image        core.Image
	invite       core.Invite
	job          core.Job
	key          core.Key
	namespace    core.Namespace
	object       core.Object
	tag          core.Tag
	variable     core.Variable

	DB    *sqlx.DB
	Redis *redis.Client

	Images    filestore.FileStore
	Objects   filestore.FileStore
	Artifacts filestore.FileStore

	ImageLimit  int64
	ObjectLimit int64

	Queues map[string]*machinery.Server

	Handler    web.Handler
	Middleware web.Middleware
}

func notFound(w nethttp.ResponseWriter, r *nethttp.Request) {
	web.HTMLError(w, "Not found", nethttp.StatusNotFound)
}

func methodNotAllowed(w nethttp.ResponseWriter, r *nethttp.Request) {
	web.HTMLError(w, "Method not allowed", nethttp.StatusMethodNotAllowed)
}

func (s *Server) Init() {
	store := model.Store{
		DB: s.DB,
	}

	s.namespace = core.Namespace{
		Handler:    s.Handler,
		Namespaces: model.NamespaceStore{
			Store: store,
		},
	}

	s.build = core.Build{
		Handler:    s.Handler,
		Namespace:  s.namespace,
		Namespaces: model.NamespaceStore{
			Store: store,
		},
		Builds:     model.BuildStore{
			Store: store,
		},
		Client:     s.Redis,
		Queues:     s.Queues,
	}

	s.collaborator = core.Collaborator{
		Handler: s.Handler,
		Invites: model.InviteStore{
			Store: store,
		},
	}

	s.image = core.Image{
		Handler:   s.Handler,
		Namespace: s.namespace,
		FileStore: s.Images,
		Limit:     s.ImageLimit,
		Images:    model.ImageStore{
			Store: store,
		},
	}

	s.invite = core.Invite{
		Handler: s.Handler,
	}

	s.job = core.Job{
		Handler: s.Handler,
	}

	s.key = core.Key{
		Handler:    s.Handler,
		Namespace:  s.namespace,
		Namespaces: model.NamespaceStore{
			Store: store,
		},
	}

	s.object = core.Object{
		Handler:    s.Handler,
		Namespace:  s.namespace,
		FileStore:  s.Objects,
		Limit:      s.ObjectLimit,
		Namespaces: model.NamespaceStore{
			Store: store,
		},
		Objects:   model.ObjectStore{
			Store: store,
		},
	}

	s.tag = core.Tag{
		Handler: s.Handler,
	}

	s.variable = core.Variable{
		Handler:    s.Handler,
		Namespaces: s.namespace,
		Namespaces: model.NamespaceStore{
			Store: store,
		},
		Variables:  model.VariableStore{
			Store: store,
		},
	}
}
