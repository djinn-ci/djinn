// Package server provides an HTTP server implementation that wraps the
// http.Server from the stdlib, and a Router interface for implementing routing.
package server

import (
	"encoding/gob"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/log"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/webutil"

	"github.com/gorilla/mux"
)

// Router defines how a router should be implemented to be used for the Server.
// It is perfectly valid for the RegisterUI or RegisterAPI methods to be simple
// stubs to statisfy the interface is a Router doesn't need to expose either via
// the Server.
type Router interface {
	// RegisterUI will register the router's UI routes with the given
	// mux.Router. It will also pass through the CSRF middleware function, and
	// a variadic list of gates to apply to the routes being registered.
	RegisterUI(*mux.Router, func(http.Handler) http.Handler, ...web.Gate)

	// RegisterAPI will register the router's API routes with the given
	// mux.Router. Unlike RegisterUI, this does not take a CSRF middleware
	// function, and only the variadic list of gates.
	RegisterAPI(string, *mux.Router, ...web.Gate)
}

// Server is a wrapper around the stdlib http.Server. It provides a simple
// mechanism of adding Routers for routing requests.
type Server struct {
	*http.Server

	Log     *log.Logger       // Log is the logger to use for application logging.
	Router  *mux.Router       // Router is the mux.Router to use for registering routes.
	Routers map[string]Router // Routers defines the routers for the server, along with their name.

	// Cert and Key define the paths to the certificate and key to use for
	// serving over TLS.
	Cert string
	Key  string
}

// API wraps the Server struct, and uses a separate mux.Router for serving
// routes for the API.
type API struct {
	*Server

	apiRouter *mux.Router

	// Prefix defines the router prefix to register the API routes against.
	Prefix string
}

// UI wraps the Server struct, and provides CSRF middleware for each route.
type UI struct {
	*Server

	// CSRF defines the middleware function to use for protecting form submissions
	// from CSRF attacks.
	CSRF func(http.Handler) http.Handler
}

func spoofHandler(h http.Handler) http.HandlerFunc {
	methods := map[string]struct{}{
		"PATCH":  {},
		"PUT":    {},
		"DELETE": {},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			typ := r.Header.Get("Content-Type")

			if strings.HasPrefix(typ, "application/x-www-form-urlencoded") ||
				strings.HasPrefix(typ, "multipart/form-data") {
				method := r.PostFormValue("_method")

				if _, ok := methods[method]; ok {
					r.Method = method
				}
			}
		}
		h.ServeHTTP(w, r)
	})
}

// Init will initialize the UI server, and register the necessary types with
// gob for encoding session data, such as form errors, form fields, and alerts.
// This will also wrap the underlying Router with a handler for spoofing HTTP
// methods, such as PATCH, and DELETE.
func (s *UI) Init() {
	if s.Router == nil {
		panic("initializing ui server with nil router")
	}

	gob.Register(webutil.NewErrors())
	gob.Register(template.Alert{})
	gob.Register(make(map[string]string))
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
		r.RegisterAPI(s.Prefix, s.apiRouter, gates...)
	}
}

func (s *Server) Init() {
	if s.Router == nil {
		panic("initializing server with nil router")
	}
	s.Server.Handler = s.recoverHandler(spoofHandler(s.Router))
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

func (s *Server) recoverHandler(h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				errh := web.HTMLError

				if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
					errh = web.JSONError
				}

				if err, ok := v.(error); ok {
					s.Log.Error.Println(r.Method, r.URL, err)
				}

				s.Log.Error.Println(r.Method, r.URL, string(debug.Stack()))
				errh(w, "Something went wrong", http.StatusInternalServerError)
			}
		}()
		h.ServeHTTP(w, r)
	})
}
