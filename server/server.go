// Package server provides an HTTP server implementation that wraps the
// http.Server from the stdlib, and a Router interface for implementing routing.
package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/gob"
	"encoding/hex"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"djinn-ci.com/alert"
	"djinn-ci.com/auth"
	"djinn-ci.com/config"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/log"
	"djinn-ci.com/mail"
	"djinn-ci.com/provider"
	"djinn-ci.com/queue"
	"djinn-ci.com/template"
	"djinn-ci.com/template/form"
	"djinn-ci.com/user"
	"djinn-ci.com/version"

	"github.com/andrewpillar/fs"
	"github.com/andrewpillar/webutil/v2"

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

	Host  string
	Debug bool

	AESGCM *crypto.AESGCM // The mechanism used for encrypting data.
	Hasher *crypto.Hasher // The mechanism for generating secure hashes.

	DB    *database.Pool // The client connection to the PostgreSQL database.
	Redis *redis.Client  // The client connection to the Redis database.

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

	Artifacts fs.FS
	Images    fs.FS
	Objects   fs.FS

	Auths *auth.Registry

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

	authKey, block, hash, _ := cfg.Crypto()

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

	db := cfg.DB()
	securecookie := securecookie.New(hash, block)

	token := user.TokenAuth(db)
	cookie := user.CookieAuth(db, securecookie)
	form := user.FormAuth(db)

	auths := cfg.Auths()
	auths.Register(user.InternalProvider+":cookie", cookie)
	auths.Register(user.InternalProvider+":form", form)
	auths.Register(user.InternalProvider, auth.Fallback(token, cookie, form))

	srv := &Server{
		Host:         cfg.Host(),
		Debug:        cfg.Debug(),
		Server:       cfg.Server(),
		AESGCM:       cfg.AESGCM(),
		Hasher:       cfg.Hasher(),
		Log:          cfg.Log(),
		Router:       mux.NewRouter(),
		DB:           db,
		Redis:        redis,
		DriverQueues: cfg.DriverQueues(),
		Queues:       queue.NewSet(),
		SecureCookie: securecookie,
		SessionStore: store,
		CSRF: csrf.Protect(
			authKey,
			csrf.RequestHeader("X-CSRF-Token"),
			csrf.FieldName("csrf_token"),
		),
		Artifacts: cfg.Artifacts(),
		Images:    cfg.Images(),
		Objects:   cfg.Objects(),
		Auths:     auths,
		Providers: cfg.Providers(),
	}

	srv.Router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.NotFound(w, r)
	})

	srv.Router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		srv.Error(w, r, errors.New("Method Not Allowed"), http.StatusMethodNotAllowed)
	})

	srv.Router.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		if ExpectsJSON(r) {
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
		if !isAPI(r) {
			_, save := s.Session(r)
			save(r, w)
		}
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
func runtimeStack(encode bool) string {
	stack := debug.Stack()

	if !encode {
		return string(stack)
	}

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

func headersHandler(h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAPI(r) {
			w.Header().Set("X-Frame-Options", "deny")
		}
		h.ServeHTTP(w, r)
	})
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
	gob.Register(map[int64]struct{}{})

	gob.Register(alert.Alert{})
	gob.Register(time.Time{})
	gob.Register(webutil.ValidationErrors{})

	s.Server.Handler = s.recoverHandler(headersHandler(spoofHandler(s.Router)))
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

var ErrNotFound = errors.Benign("Not Found")

// NotFound replies to the request with a 404 Not found response.
func (s *Server) NotFound(w http.ResponseWriter, r *http.Request) {
	s.Error(w, r, ErrNotFound, http.StatusNotFound)
}

// InternalServerError replies to the request with 500 Internal server error,
// and logs the given error to the underlying logger.
func (s *Server) InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	s.Error(w, r, errors.Wrap(err, "Internal Server Error"), http.StatusInternalServerError)
}

func isAPI(r *http.Request) bool {
	url, _ := url.Parse(webutil.BaseAddress(r))
	api, _ := url.Parse(env.DJINN_API_SERVER)

	return api.Scheme == url.Scheme && api.Host == url.Host && strings.HasPrefix(r.URL.Path, api.Path)
}

func ExpectsJSON(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Accept"), "application/json") ||
		strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") ||
		isAPI(r)
}

// Error serves up the given error with the given HTTP status code. If err is
// nil, then the status text for the HTTP status code is used. If the error is
// of type errors.Benign, then it is not logged.
func (s *Server) Error(w http.ResponseWriter, r *http.Request, err error, code int) {
	if _, ok := err.(errors.Benign); !ok {
		s.Log.Error.Println(r.Method, r.URL, errors.Unwrap(err))
	}

	typ := r.Header.Get("Content-Type")

	// Form submission, so redirect back.
	if strings.HasPrefix(typ, "application/x-www-form-urlencoded") ||
		strings.HasPrefix(typ, "multipart/form-data") {
		sess, _ := s.Session(r)

		alert.Flash(sess, alert.Danger, err.Error())
		s.RedirectBack(w, r)
		return
	}

	if ExpectsJSON(r) {
		m := map[string]string{
			"message": err.Error(),
		}

		if s.Debug {
			m["stack"] = errors.Format(errors.Unwrap(err))
		}

		webutil.JSON(w, m, code)
		return
	}

	tmpl := template.Error{
		Code:    code,
		Message: err.Error(),
	}

	if s.Debug {
		tmpl.Error = errors.Unwrap(err)
	}
	webutil.HTML(w, template.Render(&tmpl), code)
}

// FormError checks if the given error implements webutil.ValidationErrors, if
// it does, then the form and errors are flashed to the session, and a redirect
// to the request Referrer is made. Otherwise it defers to Server.Error.
func (s *Server) FormError(w http.ResponseWriter, r *http.Request, f webutil.Form, err error) {
	var errs webutil.ValidationErrors

	if errors.As(err, &errs) {
		if ExpectsJSON(r) {
			webutil.JSON(w, errs, http.StatusBadRequest)
			return
		}

		sess, _ := s.Session(r)

		webutil.FlashFormWithErrors(sess, f, errs)
		s.RedirectBack(w, r)
		return
	}
	s.Error(w, r, err, http.StatusInternalServerError)
}

func (s *Server) recoverHandler(h http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if ExpectsJSON(r) {
					data := map[string]string{
						"message": "Something went wrong",
						"stack":   runtimeStack(!s.Debug),
					}

					if err, ok := v.(error); ok {
						s.Log.Error.Println(err)
					}
					s.Log.Error.Println(string(debug.Stack()))
					webutil.JSON(w, data, http.StatusInternalServerError)
					return
				}

				p := &template.FatalError{
					Error: template.Error{
						Code:    http.StatusInternalServerError,
						Message: "Fatal error, when submitting an issue please include the following",
					},
					Stack: runtimeStack(!s.Debug),
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

func (s *Server) Guest(a auth.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				// User is logging in, so bypass authentication check.
				if r.URL.Path == "/login" {
					next.ServeHTTP(w, r)
					return
				}
			}

			u, err := a.Auth(r)

			if err != nil {
				if !errors.Is(err, auth.ErrAuth) {
					s.InternalServerError(w, r, errors.Wrap(err, "Failed to authenticate request"))
					return
				}
				goto serve
			}

			if u.ID > 0 {
				if ExpectsJSON(r) {
					s.NotFound(w, r)
					return
				}

				s.Redirect(w, r, "/")
				return
			}

		serve:
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) Restrict(a auth.Authenticator, perms []string, fn auth.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := a.Auth(r)

		if err != nil {
			if errors.Is(err, database.ErrNoRows) || errors.Is(err, auth.ErrAuth) {
				if ExpectsJSON(r) {
					s.NotFound(w, r)
					return
				}

				s.Redirect(w, r, "/login")
				return
			}
			s.InternalServerError(w, r, err)
			return
		}

		for _, perm := range perms {
			if !u.Has(perm) {
				if ExpectsJSON(r) {
					s.NotFound(w, r)
					return
				}
				s.Redirect(w, r, "/login")
				return
			}
		}
		fn(u, w, r)
	})
}

func (s *Server) Optional(a auth.Authenticator, fn auth.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := a.Auth(r)

		if err != nil {
			if !errors.Is(err, auth.ErrAuth) {
				s.InternalServerError(w, r, err)
				return
			}
			u = &auth.User{}
		}
		fn(u, w, r)
	})
}

const (
	sudoTimestamp = "sudo_timestamp"
	sudoToken     = "sudo_token"
	sudoUrl       = "sudo_url"
	sudoReferer   = "sudo_referer"
)

func (s *Server) generateSudoToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

type sudoForm struct {
	User  *auth.User `schema:"-"`
	Token string     `schema:"-"`

	SudoToken   string `schema:"sudo_token"`
	SudoURL     string `schema:"sudo_url"`
	SudoReferer string `schema:"sudo_referer"`
	Password    string
}

var _ webutil.Form = (*sudoForm)(nil)

func (f sudoForm) Fields() map[string]string { return nil }

func (f sudoForm) Validate(ctx context.Context) error {
	var v webutil.Validator

	nametab := map[string]string{
		"password": "Password",
	}

	v.WrapError(func(name string, err error) error {
		if s, ok := nametab[name]; ok {
			name = s
		}
		return webutil.WrapFieldError(name, err)
	})

	v.Add("password", f.Password, webutil.FieldRequired)
	v.Add("sudo_token", f.SudoToken, webutil.FieldEquals(f.Token))

	errs := v.Validate(ctx)

	return errs.Err()
}

func (s *Server) sudoHandler(u *auth.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := s.Session(r)

	tok := sess.Values[sudoToken].(string)
	delete(sess.Values, sudoToken)

	f := sudoForm{
		User:  u,
		Token: tok,
	}

	if err := webutil.UnmarshalFormAndValidate(&f, r); err != nil {
		s.FormError(w, r, &f, err)
		return
	}

	r.PostForm.Set("handle", u.Email)

	form, _ := s.Auths.Get(user.InternalProvider + ":form")

	if _, err := form.Auth(r); err != nil {
		if !errors.Is(err, auth.ErrAuth) {
			s.InternalServerError(w, r, errors.Wrap(err, "Failed to authenticate request"))
			return
		}

		s.InternalServerError(w, r, errors.Benign("Authentication failed"))
		return
	}

	expires := time.Now().Add(time.Minute * 30)

	s.Log.Debug.Println(r.Method, r.URL, "authorizing sudo request")
	s.Log.Debug.Println(r.Method, r.URL, "sudo request authorized, expires at", expires)

	sess.Values[sudoTimestamp] = expires
	s.Redirect(w, r, f.SudoURL)
}

func (s *Server) Template(w http.ResponseWriter, r *http.Request, tmpl template.Template, code int) {
	_, save := s.Session(r)
	save(r, w)
	webutil.HTML(w, template.Render(tmpl), code)
}

func (s *Server) Sudo(fn auth.HandlerFunc) http.HandlerFunc {
	registered := false

	s.Router.Walk(func(r *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		if name := r.GetName(); name == "sudo" {
			registered = true
			return nil
		}
		return nil
	})

	cookie, _ := s.Auths.Get(user.InternalProvider + ":cookie")

	if !registered {
		s.Router.HandleFunc("/sudo", s.Restrict(cookie, nil, s.sudoHandler)).Name("sudo").Methods("POST")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if ExpectsJSON(r) {
			webutil.JSON(w, map[string]any{"message": "Unauthorized"}, http.StatusUnauthorized)
			return
		}

		u, err := cookie.Auth(r)

		if err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     "user",
				HttpOnly: true,
				Path:     "/",
				Expires:  time.Unix(0, 0),
			})

			s.InternalServerError(w, r, errors.Wrap(err, "Failed to authenticate request"))
			return
		}

		sess, save := s.Session(r)

		if v, ok := sess.Values[sudoTimestamp]; ok {
			t := v.(time.Time)

			if time.Now().Before(t) {
				s.Log.Debug.Println(r.Method, r.URL, "sudo still in session")

				if v, ok := sess.Values[sudoReferer]; ok {
					delete(sess.Values, sudoReferer)
					s.Log.Debug.Println(r.Method, r.URL, "sudo referer", v)
					r.Header.Set("Referer", v.(string))
				}

				fn(u, w, r)
				return
			}
			delete(sess.Values, sudoTimestamp)
		}

		s.Log.Debug.Println(r.Method, r.URL, "generating sudo token")

		tok := s.generateSudoToken()
		url := r.URL.String()
		ref := r.Header.Get("Referer")

		sess.Values[sudoToken] = tok
		sess.Values[sudoUrl] = url
		sess.Values[sudoReferer] = ref

		save(r, w)

		tmpl := template.SudoForm{
			Form:        form.New(sess, r),
			Alert:       alert.First(sess),
			Email:       u.Email,
			SudoURL:     url,
			SudoReferer: ref,
			SudoToken:   tok,
		}
		webutil.HTML(w, template.Render(&tmpl), http.StatusOK)
	}
}
