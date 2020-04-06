package server

import (
	"encoding/gob"
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/mux"
)

type Router interface {
	// Init will initialize the router with the base web.Handler.
	Init(web.Handler)

	// RegisterUI will register the router's UI routes with the given
	// mux.Router. It will also pass through the CSRF middleware function, and
	// a variadic list of gates to apply to the routes being registered.
	RegisterUI(*mux.Router, func(http.Handler) http.Handler, ...web.Gate)

	// RegisterAPI will register the router's API routes with the given
	// mux.Router. Unlike RegisterUI, this does not take a CSRF middleware
	// function, and only the variadic list of gates.
	RegisterAPI(*mux.Router, ...web.Gate)
}

type Server struct {
	*http.Server

	// Router is the mux.Router to use for registering routes.
	Router  *mux.Router

	// Routers defines the routers for the server, along with their name,
	Routers map[string]Router

	// Cert and Key define the paths to the certificate and key to use for
	// serving over TLS.
	Cert    string
	Key     string
}

type API struct {
	Server

	apiRouter *mux.Router

	// Prefix defines the router prefix to register the API routes against.
	Prefix string
}

type UI struct {
	Server

	// CSRF defines the middleware function to use for protecting form submissions
	// from CSRF attacks.
	CSRF func(http.Handler) http.Handler
}

// Init will initialize the API server, and register the necessary types with
// gob for encoding session data, such as form errors, form fields, and alerts.
// This will also wrap the underlying Router with a handler for spoofing HTTP
// methods, such as PATCH, and DELETE.
func (s *UI) Init() {
	gob.Register(form.NewErrors())
	gob.Register(template.Alert{})
	gob.Register(make(map[string]string))

	if s.Router == nil {
		panic("initializing ui server with nil router")
	}

	s.Server.Server.Handler = web.NewSpoof(s.Router)
}

// Register will register the UI routers of the given name with the given
// gates.
func (s *UI) Register(name string, gates ...web.Gate) {
	if r, ok := s.Routers[name]; ok {
		r.RegisterUI(s.Router, s.CSRF, gates...)
	}
}

// Init will initialize the API server. This will create a new subrouter to
// register the routes on if the Prefix for the API server was set.
func (s *API) Init() {
	if s.Router == nil {
		panic("initializing api server with nil router")
	}

	s.apiRouter = s.Router

	if s.Prefix != "" {
		s.apiRouter = s.Router.PathPrefix(s.Prefix).Subrouter()
	}
}

// Register will register the API routers of the given name with the given
// gates.
func (s *API) Register(name string, gates ...web.Gate) {
	if r, ok := s.Routers[name]; ok {
		r.RegisterAPI(s.apiRouter, gates...)
	}
}

// Init initializes all of the routers given to the server with the given base
// web.Handler.
func (s *Server) Init(h web.Handler) {
	if s.Router == nil {
		panic("initializing server with nil router")
	}

	s.Server.Handler = s.Router

	for _, r := range s.Routers {
		r.Init(h)
	}
}

// AddRouter adds the given router to the server with the given name. If the
// router already exists, then it will be replaced.
func (s *Server) AddRouter(name string, r Router) {
	if s.Routers == nil {
		s.Routers = make(map[string]Router)
	}
	s.Routers[name] = r
}

// Serve will bind the server to the given address. If a certificate and key
// were given, then the server will be served over TLS.
func (s *Server) Serve() error {
	if s.Cert != "" && s.Key != "" {
		return errors.Err(s.ListenAndServeTLS(s.Cert, s.Key))
	}
	return errors.Err(s.ListenAndServe())
}
