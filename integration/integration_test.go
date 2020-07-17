// +build integration

package integration

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/andrewpillar/thrall/block"
	buildweb "github.com/andrewpillar/thrall/build/web"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	imageweb "github.com/andrewpillar/thrall/image/web"
	keyweb "github.com/andrewpillar/thrall/key/web"
	"github.com/andrewpillar/thrall/log"
	namespaceweb "github.com/andrewpillar/thrall/namespace/web"
	objectweb "github.com/andrewpillar/thrall/object/web"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/user"
	variableweb "github.com/andrewpillar/thrall/variable/web"
	"github.com/andrewpillar/thrall/web"

	"github.com/gorilla/mux"

	goredis "github.com/go-redis/redis"

	"github.com/RichardKnop/machinery/v1"
	qconfig "github.com/RichardKnop/machinery/v1/config"
)

type Flow struct {
	requests []*http.Request
	codes    []int
	handlers []func(*testing.T, *http.Request, []byte)
	done     map[int]struct{}
}

var (
	myTok   *oauth2.Token
	yourTok *oauth2.Token

	server *httptest.Server
)

func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

func getenv(key string) string {
	val := os.Getenv(key)

	if val == "" {
		fatalf("environment variable %s not set\n", key)
	}
	return val
}

func NewFlow() *Flow {
	return &Flow{
		requests: make([]*http.Request, 0),
		codes:    make([]int, 0),
		handlers: make([]func(*testing.T, *http.Request, []byte), 0),
		done:     make(map[int]struct{}),
	}
}

func checkJSONResponseSize(l int) func(*testing.T, *http.Request, []byte) {
	return func(t *testing.T, r *http.Request, b []byte) {
		data := make([]interface{}, 0)

		if err := json.Unmarshal(b, &data); err != nil {
			t.Errorf("%q %q - unexpected Unmarshal error: %s\n", r.Method, r.URL, err)
		}

		if len(data) != l {
			t.Errorf("%q %q - expected json array to have %d items, got=%d\n", r.Method, r.URL, l, len(data))
		}
	}
}

func checkJSONResponseSizeApprox(l int) func(*testing.T, *http.Request, []byte) {
	return func(t *testing.T, r *http.Request, b []byte) {
		data := make([]interface{}, 0)

		if err := json.Unmarshal(b, &data); err != nil {
			t.Errorf("%q %q - unexpected Unmarshal error: %s\n", r.Method, r.URL, err)
		}

		if len(data) < l {
			t.Errorf("%q %q - expected json array to be >= %d, got=%d\n", r.Method, r.URL, l, len(data))
		}
	}
}

func ReadFile(t *testing.T, name string) []byte {
	b, err := ioutil.ReadFile(filepath.Join("testdata", name))

	if err != nil {
		t.Fatalf("unexpected ReadFile error: %s\n", err)
	}
	return b
}

func JSON(t *testing.T, m map[string]interface{}) *bytes.Buffer {
	buf := &bytes.Buffer{}

	if err := json.NewEncoder(buf).Encode(m); err != nil {
		t.Fatalf("unexpected Encode error: %s\n", err)
	}
	return buf
}

// ApiGet returns a new HTTP GET request to the given path with the given
// token set in the Authorization header.
func ApiGet(t *testing.T, path string, tok *oauth2.Token) *http.Request {
	return NewApiRequest(t, "GET", path, tok, nil)
}

// ApiPost returns a new HTTP POST request to the given path, with the given
// io.Reader as the body and with the given token set in the Authorization
// header.
func ApiPost(t *testing.T, path string, tok *oauth2.Token, r io.Reader) *http.Request {
	return NewApiRequest(t, "POST", path, tok, r)
}

// ApiPatch returns a new HTTP PATCH request to the given path, with the given
// io.Reader as the body and with the given token set in the Authorization
// header.
func ApiPatch(t *testing.T, path string, tok *oauth2.Token, r io.Reader) *http.Request {
	return NewApiRequest(t, "PATCH", path, tok, r)
}

// ApiDelete returns a new HTTP DELETE request to the given path with the given
// token set in the Authorization header.
func ApiDelete(t *testing.T, path string, tok *oauth2.Token) *http.Request {
	return NewApiRequest(t, "DELETE", path, tok, nil)
}

// NewApiRequest returns an http.Request with the given method, and path. This
// will set the Authorization header to the given token, and set the
// Content-Type header to application/json. If the creation of the request fails
// then Fatalf is called on the given testing.T.
func NewApiRequest(t *testing.T, method, path string, tok *oauth2.Token, r io.Reader) *http.Request {
	req, err := http.NewRequest(method, server.URL + path, r)

	if err != nil {
		t.Fatalf("unexpected NewRequest error: %s\n", err)
	}

	req.Header.Set("Authorization", "Bearer " + hex.EncodeToString(tok.Token))
	req.Header.Set("Content-Type", "application/json")

	return req
}

// OpenFile opens a file of the given name from within the testdata directory
// and returns it. If any error occurs then Fatalf is called on the given
// testing.T.
func OpenFile(t *testing.T, name string) *os.File {
	f, err := os.Open(filepath.Join("testdata", name))

	if err != nil {
		t.Fatalf("unexpected Open error: %s\n", err)
	}
	return f
}

// ReadAll returns all of the read bytes from the given io.Reader. If any errors
// occur then Fatalf is called on the given testing.T.
func ReadAll(t *testing.T, r io.Reader) []byte {
	b, err := ioutil.ReadAll(r)

	if err != nil {
		t.Fatalf("unexpected ReadAll error: %s\n", err)
	}
	return b
}

func (f *Flow) Add(r *http.Request, code int, handler func(*testing.T, *http.Request, []byte)) {
	f.requests = append(f.requests, r)
	f.codes = append(f.codes, code)
	f.handlers = append(f.handlers, handler)
}

func (f *Flow) fatal(t *testing.T, i int, req []byte, resp []byte, err error) {
	buf := bytes.Buffer{}
	buf.WriteString("requests[" + strconv.FormatInt(int64(i), 10) + "] - " + err.Error() + "\n")
	buf.Write(req)
	buf.WriteString("\n")
	buf.Write(resp)

	t.Fatal(buf.String())
}

func (f *Flow) Do(t *testing.T, cli *http.Client) {
	for i, r := range f.requests {
		if _, ok := f.done[i]; ok {
			continue
		}

		func(i int, r *http.Request) {
			reqBytes, err := httputil.DumpRequest(r, true)

			if err != nil {
				f.fatal(t, i, reqBytes, nil, errors.New("unexpected DumpRequest error: " + err.Error()))
				return
			}

			resp, err := cli.Do(r)

			if err != nil {
				f.fatal(t, i, reqBytes, nil, errors.New("unexpected Do error: " + err.Error()))
				return
			}

			respBytes, err := httputil.DumpResponse(resp, true)

			if err != nil {
				f.fatal(t, i, reqBytes, respBytes, errors.New("unexpected DumpResponse error: " + err.Error()))
				return
			}

			defer resp.Body.Close()

			if resp.StatusCode != f.codes[i] {
				f.fatal(t, i, reqBytes, respBytes, fmt.Errorf("expected http status %d got=%d", f.codes[i], resp.StatusCode))
				return
			}

			b, err := ioutil.ReadAll(resp.Body)

			if err != nil {
				f.fatal(t, i, reqBytes, respBytes, errors.New("unexpected ReadAll error: " + err.Error()))
				return
			}

			if handler := f.handlers[i]; handler != nil {
				handler(t, r, b)
			}

			f.done[i] = struct{}{}
		}(i, r)
	}
}

func TestMain(m *testing.M) {
	var err error

	pgaddr := getenv("PGADDR")
	redisaddr := getenv("RDADDR")

	db, err := database.Connect(pgaddr)

	if err != nil {
		fatalf("failed to connect to database: %s\n", err)
	}

	redis := goredis.NewClient(&goredis.Options{
		Addr: redisaddr,
	})

	if _, err := redis.Ping().Result(); err != nil {
		fatalf("failed to connect to redis: %s\n", err)
	}

	users := user.NewStore(db)

	me, err := users.Create("me@example.com", "me", []byte("secret"))

	if err != nil {
		fatalf("failed to create user: %s\n", err)
	}

	you, err := users.Create("you@example.com", "you", []byte("secret"))

	if err != nil {
		fatalf("failed to create user: %s\n", err)
	}

	scope := oauth2.NewScope()

	for _, res := range oauth2.Resources {
		for _, perm := range oauth2.Permissions {
			scope.Add(res, perm)
		}
	}

	myTok, err = oauth2.NewTokenStore(db, me).Create("My Token", scope)

	if err != nil {
		fatalf("failed to create token: %s\n", err)
	}

	yourTok, err = oauth2.NewTokenStore(db, you).Create("Your Token", scope)

	if err != nil {
		fatalf("failed to create token: %s\n", err)
	}

	logf, err := os.Create("server.log")

	if err != nil {
		fatalf("failed to create server.log file: %s\n", err)
	}

	defer logf.Close()

	log := log.New(logf)
	log.SetLevel("debug")

	blockStore := block.NewNull()

	hasher := &crypto.Hasher{
		Salt:   "123456",
		Length: 8,
	}

	if err := hasher.Init(); err != nil {
		fatalf("failed to initialize hashing: %s\n", err)
	}

	blockCipher, err := crypto.NewBlock([]byte("some-supersecret"))

	if err != nil {
		fatalf("failed to create block cipher: %s\n", err)
	}

	queues := make(map[string]*machinery.Server, 0)

	for _, d := range []string{"qemu", "docker"} {
		queue, err := machinery.NewServer(&qconfig.Config{
			Broker:        "redis://" + redisaddr,
			DefaultQueue:  "builds",
			ResultBackend: "redis://" + redisaddr,
		})

		if err != nil {
			fatalf("failed to setup queue %s: %s\n", d, err)
		}
		queues[d] = queue
	}

	webHandler := web.Handler{
		DB:    db,
		Log:   log,
		Users: users,
	}

	middleware := web.Middleware{
		Handler: webHandler,
		Users:   users,
		Tokens:  oauth2.NewTokenStore(db),
	}

	router := mux.NewRouter()
	subrouter := router.PathPrefix("/api").Subrouter()

	buildRouter := buildweb.Router{
		Block:      blockCipher,
		Middleware: middleware,
		Artifacts:  blockStore,
		Redis:      redis,
		Hasher:     hasher,
		Queues:     queues,
	}
	buildRouter.Init(webHandler)
	buildRouter.RegisterAPI("/api", subrouter, buildweb.Gate(db))

	namespaceRouter := namespaceweb.Router{
		Middleware: middleware,
	}
	namespaceRouter.Init(webHandler)
	namespaceRouter.RegisterAPI("/api", subrouter, namespaceweb.Gate(db))

	imageRouter := imageweb.Router{
		Middleware: middleware,
		Hasher:     hasher,
		BlockStore: blockStore,
	}
	imageRouter.Init(webHandler)
	imageRouter.RegisterAPI("/api", subrouter, imageweb.Gate(db))

	objectRouter := objectweb.Router{
		Middleware: middleware,
		Hasher:     hasher,
		BlockStore: blockStore,
	}
	objectRouter.Init(webHandler)
	objectRouter.RegisterAPI("/api", subrouter, objectweb.Gate(db))

	variableRouter := variableweb.Router{
		Middleware: middleware,
	}
	variableRouter.Init(webHandler)
	variableRouter.RegisterAPI("/api", subrouter, variableweb.Gate(db))

	keyRouter := keyweb.Router{
		Block:      blockCipher,
		Middleware: middleware,
	}
	keyRouter.Init(webHandler)
	keyRouter.RegisterAPI("/api", subrouter, keyweb.Gate(db))

	server = httptest.NewServer(router)

	log.Debug.Println("serving integration test server on", server.Listener.Addr())

	defer server.Close()

	os.Exit(m.Run())
}
