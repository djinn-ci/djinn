package server

import (
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web/api"
)

type API struct {
	Server

	Prefix string

	build api.Build
	gate  gate
}

func (s *API) Init() {
	store := model.Store{DB: s.DB}

	if s.Prefix != "" {
		s.router = s.router.PathPrefix(s.Prefix).Subrouter()
	}

	s.gate = gate{
		users:      model.UserStore{Store: store},
		namespaces: model.NamespaceStore{Store: store},
		images:     model.ImageStore{Store: store},
		objects:    model.ObjectStore{Store: store},
		variables:  model.VariableStore{Store: store},
		keys:       model.KeyStore{Store: store},
	}

	s.build = api.Build{
		Core: s.Server.build,
	}
}

func (s *API) Build() {
	r := s.router.PathPrefix("/b/{username}/{build:[0-9]+}").Subrouter()
	r.HandleFunc("", s.build.Show).Methods("GET")
	r.HandleFunc("", s.build.Kill).Methods("DELETE")
	r.HandleFunc("/objects", s.build.Show).Methods("GET")
	r.HandleFunc("/variables", s.build.Show).Methods("GET")
	r.HandleFunc("/keys", s.build.Show).Methods("GET")
}
