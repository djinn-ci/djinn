package server

import (
	"net/http"

	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"
	"github.com/andrewpillar/thrall/web/core"

	"github.com/jmoiron/sqlx"

	"github.com/gorilla/mux"

	"github.com/go-redis/redis"

	"github.com/RichardKnop/machinery/v1"
)

type Server struct {
	*http.Server

	Cert string
	Key  string

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

	CSRFToken []byte

	Images    filestore.FileStore
	Objects   filestore.FileStore
	Artifacts filestore.FileStore

	ImageLimit  int64
	ObjectLimit int64

	Queues map[string]*machinery.Server

	Handler    web.Handler
	Middleware web.Middleware
}

func notFound(w http.ResponseWriter, r *http.Request) {
	web.HTMLError(w, "Not found", http.StatusNotFound)
}

func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	web.HTMLError(w, "Method not allowed", http.StatusMethodNotAllowed)
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
		Images:     model.ImageStore{
			Store: store,
		},
		Variables:  model.VariableStore{
			Store: store,
		},
		Keys:       model.KeyStore{
			Store: store,
		},
		Objects:    model.ObjectStore{
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
		Keys:       model.KeyStore{
			Store: store,
		},
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
		Namespace:  s.namespace,
		Namespaces: model.NamespaceStore{
			Store: store,
		},
		Variables:  model.VariableStore{
			Store: store,
		},
	}
}

func (s *Server) Serve() error {
	if s.Cert != "" && s.Key != "" {
		return s.ListenAndServeTLS(s.Cert, s.Key)
	}

	return s.ListenAndServe()
}
