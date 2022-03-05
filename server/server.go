// Package server provides an HTTP server implementation that wraps the
// http.Server from the stdlib, and a Router interface for implementing routing.
package server

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/gob"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"

	"djinn-ci.com/alert"
	"djinn-ci.com/config"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/log"
	"djinn-ci.com/mail"
	"djinn-ci.com/provider"
	"djinn-ci.com/queue"
	"djinn-ci.com/template"
	"djinn-ci.com/version"

	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"

	"github.com/go-redis/redis"

	"github.com/mcmathja/curlyq"

	"github.com/rbcervilla/redisstore"
)

type Server struct {
	*http.Server

	AESGCM *crypto.AESGCM // The mechanism used for encrypting data.
	Hasher *crypto.Hasher // The mechanism for generating secure hashes.

	DB    database.Pool // The client connection to the PostgreSQL database.
	Redis *redis.Client // The client connection to the Redis database.

	// SMTP provides both the SMTP client for sending emails, and the address
	// from which the emails would be sent, typically the admin of the server.
	SMTP struct {
		Client *mail.Client
		From   string
	}

	// DriverQueues is a map holding the different producers that would be
	// used for submitting a build to a driver specific queue for running.
	DriverQueues map[string]*curlyq.Producer

	// Queues is a set of different queues that jobs can be produced to for
	// background processing. These are distinct from the DriverQueues used
	// for running builds.
	Queues *queue.Set

	SecureCookie *securecookie.SecureCookie
	SessionStore sessions.Store
	CSRF         func(http.Handler) http.Handler

	Artifacts fs.Store
	Images    fs.Store
	Objects   fs.Store

	// Providers contains the configured providers that the server can connect
	// to for 3rd party integration.
	Providers *provider.Registry

	// Log for application logging.
	Log *log.Logger

	// Router used for registering routes.
	Router *mux.Router
}

func New(cfg *config.Server) (*Server, error) {
	smtp, smtpadmin := cfg.SMTP()
	redis := cfg.Redis()

	auth, block, hash, _ := cfg.Crypto()

	store, err := redisstore.NewRedisStore(redis)

	if err != nil {
		return nil, errors.Err(err)
	}

	url, err := url.Parse(cfg.Host())

	if err != nil {
		return nil, errors.Err(err)
	}

	store.KeyPrefix("session_")
	store.Options(sessions.Options{
		Path:   "/",
		Domain: url.Hostname(),
		MaxAge: 86400 * 60,
	})

	srv := &Server{
		Server:       cfg.Server(),
		AESGCM:       cfg.AESGCM(),
		Hasher:       cfg.Hasher(),
		Log:          cfg.Log(),
		Router:       mux.NewRouter(),
		DB:           cfg.DB(),
		Redis:        redis,
		DriverQueues: cfg.DriverQueues(),
		Queues:       queue.NewSet(),
		SecureCookie: securecookie.New(hash, block),
		SessionStore: store,
		CSRF: csrf.Protect(
			auth,
			csrf.RequestHeader("X-CSRF-Token"),
			csrf.FieldName("csrf_token"),
		),
		Artifacts: cfg.Artifacts(),
		Images:    cfg.Images(),
		Objects:   cfg.Objects(),
		Providers: cfg.Providers(),
	}

	srv.Router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.NotFound(w, r)
	})

	srv.Router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.Error(w, r, "Method not allowed", http.StatusMethodNotAllowed)
	})

	srv.Router.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		if IsJSON(r) {
			webutil.JSON(w, map[string]string{"build": version.Build}, http.StatusOK)
			return
		}
		webutil.Text(w, version.Build, http.StatusOK)
	})

	srv.SMTP.Client = smtp
	srv.SMTP.From = smtpadmin

	return srv, nil
}

// Redirect replies to the request with a redirect to the URL. This will save
// the session before replying to the request.
func (s *Server) Redirect(w http.ResponseWriter, r *http.Request, url string) {
	_, save := s.Session(r)
	save(r, w)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

// RedirectBack replies to the request with a redirect to the Referer in the
// request. The Referer is then given to Redirect as the url.
func (s *Server) RedirectBack(w http.ResponseWriter, r *http.Request) {
	s.Redirect(w, r, r.Header.Get("Referer"))
}

// Save is a middleware function that will save all session data before serving
// the next request.
func (s *Server) Save(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, save := s.Session(r)
		save(r, w)
		next.ServeHTTP(w, r)
	})
}

const sessionName = "session"

// Session retrieves the session for the current request and a callback for
// saving that session.
func (s *Server) Session(r *http.Request) (*sessions.Session, func(*http.Request, http.ResponseWriter)) {
	sess, _ := s.SessionStore.Get(r, sessionName)

	return sess, func(r *http.Request, w http.ResponseWriter) {
		if err := sess.Save(r, w); err != nil {
			s.Log.Error.Println(r.Method, r.URL, "failed to save session", errors.Err(err))
		}
	}
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
		if i%47 == 0 {
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

func (s *Server) Init() {
	gob.Register(map[string]string{})

	gob.Register(alert.Alert{})
	gob.Register(webutil.ValidationErrors{})

	s.Server.Handler = s.recoverHandler(spoofHandler(s.Router))
}

// Serve will bind the server to the given address. If the server has a
// TLSConfig, then this will be used to serve over TLS.
func (s *Server) Serve() error {
	ln, err := net.Listen("tcp", s.Addr)

	if err != nil {
		return errors.Err(err)
	}

	if s.TLSConfig != nil {
		ln = tls.NewListener(ln, s.TLSConfig)
	}

	if err := s.Server.Serve(ln); err != nil {
		return errors.Err(err)
	}
	return nil
}

// NotFound replies to the request with a 404 Not found response.
func (s *Server) NotFound(w http.ResponseWriter, r *http.Request) {
	s.Error(w, r, "Not found", http.StatusNotFound)
}

// InternalServerError replies to the request with 500 Internal server error,
// and logs the given error to the underlying logger.
func (s *Server) InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	s.Log.Error.Println(r.Method, r.URL, err)
	s.Error(w, r, "Something went wrong", http.StatusInternalServerError)
}

func IsJSON(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Accept"), "application/json") ||
		strings.HasPrefix(r.Header.Get("Content-Type"), "application/json")
}

// Error replies to the request with the given error with the given HTTP code.
// If the given request was JSON, then the response will be JSON, otherwise the
// response will be the HTML error page.
func (s *Server) Error(w http.ResponseWriter, r *http.Request, err string, code int) {
	if IsJSON(r) {
		webutil.JSON(w, map[string]string{"message": err}, code)
		return
	}

	webutil.HTML(w, template.Render(&template.Error{
		Code:    code,
		Message: err,
	}), code)
}

func (s *Server) recoverHandler(h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if IsJSON(r) {
					data := map[string]string{
						"message": "Something went wrong",
						"stack":   encodeStack(),
					}

					if err, ok := v.(error); ok {
						s.Log.Error.Println(err)
					}
					s.Log.Error.Println(string(debug.Stack()))
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

				if err, ok := v.(error); ok {
					s.Log.Error.Println(err)
				}
				s.Log.Error.Println(string(debug.Stack()))
				webutil.HTML(w, template.Render(p), p.Code)
			}
		}()
		h.ServeHTTP(w, r)
	})
}
