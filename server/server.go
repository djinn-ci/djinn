// Package server provides an HTTP server implementation that wraps the
// http.Server from the stdlib, and a Router interface for implementing routing.
package server

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"net/http"
	"runtime/debug"
	"strings"

	"djinn-ci.com/errors"
	"djinn-ci.com/log"
	"djinn-ci.com/template"
	"djinn-ci.com/web"

	"github.com/andrewpillar/webutil"

	"github.com/gorilla/mux"
)

type MiddlewareFunc func(http.Handler) http.Handler

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

// encodeStack returns the base64 encoded string of the formatted stack trace
// for the goroutine that called it. The base64 encoded string has each line
// folded at 47 characters in length for formatting. This is used from the
// recover handler to gracefully display fatal internal errors for reporting.
func encodeStack() string {
	base64 := base64.StdEncoding.EncodeToString(debug.Stack())

	var buf bytes.Buffer

	prev := 0

	for i := range base64 {
		if i % 47 == 0 {
			buf.WriteString(base64[prev:i] + "\n")
			prev = i
			continue
		}
	}
	buf.WriteString(base64[prev:] + "\n")
	return buf.String()
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

// Init will initialize the server, and apply the given list of middleware
// functions to all of the routes added to the server.
func (s *Server) Init(middleware ...MiddlewareFunc) {
	if s.Router == nil {
		panic("initializing server with nil router")
	}

	var handler http.Handler = s.Router

	for _, mw := range middleware {
		handler = mw(handler)
	}

	s.Server.Handler = s.recoverHandler(spoofHandler(handler))
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
				if strings.HasPrefix(r.Header.Get("Accept"), "application/json") ||
					strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
					data := map[string]string{
						"message": "Something went wrong",
						"stack": encodeStack(),
					}

					webutil.JSON(w, data, http.StatusInternalServerError)
					return
				}

				p := &template.InternalError{
					Error: template.Error{
						Code:    http.StatusInternalServerError,
						Message: "Fatal error, when submitting an issue please include the following",
					},
					Stack: encodeStack(),
				}
				s.Log.Error.Println(string(debug.Stack()))
				webutil.HTML(w, template.Render(p), p.Code)
			}
		}()
		h.ServeHTTP(w, r)
	})
}
