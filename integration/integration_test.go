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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/oauth2"
	"github.com/andrewpillar/djinn/serverutil"
	"github.com/andrewpillar/djinn/user"

	"github.com/jmoiron/sqlx"
)

type client struct {
	server *httptest.Server
}

type request struct {
	name        string
	method      string
	uri         string
	token       *oauth2.Token
	contentType string
	body        io.ReadCloser
	code        int
	check       func(*testing.T, string, *http.Request, *http.Response)
}

var (
	myTok   *oauth2.Token
	yourTok *oauth2.Token

	buildRWD     *oauth2.Token // build:read,write,delete
	namespaceRWD *oauth2.Token // namespace:read,write,delete
	objectRWD    *oauth2.Token // object:read,write,delete
	variableRWD  *oauth2.Token // variable:read,write,delete
	keyRWD       *oauth2.Token // key:read,write,delete

	server *httptest.Server
	db     *sqlx.DB

	imageStore  block.Store
	objectStore block.Store
)

func checkResponseJSONLen(l int) func(*testing.T, string, *http.Request, *http.Response) {
	return func(t *testing.T, name string, req *http.Request, resp *http.Response) {
		items := make([]interface{}, 0)

		json.NewDecoder(resp.Body).Decode(&items)

		if len(items) != l {
			t.Errorf("unexpected number of json items in response for %q, expected=%d, got=%d\n", name, l, len(items))
		}
	}
}

func checkFormErrors(field string, msgs ...string) func(*testing.T, string, *http.Request, *http.Response) {
	return func(t *testing.T, name string, req *http.Request, resp *http.Response) {
		errs := make(map[string][]string)

		if err := json.NewDecoder(resp.Body).Decode(&errs); err != nil {
			t.Errorf("request test failed: %q\n", name)
			t.Fatalf("unexpected json.Decode error: %s\n", err.Error())
		}

		msgs1, ok := errs[field]

		if !ok {
			t.Fatalf("expected field %s in errors json: %v\n", field, errs)
		}

		set := make(map[string]struct{})

		for _, msg := range msgs1 {
			set[msg] = struct{}{}
		}

		for _, msg := range msgs {
			if _, ok := set[msg]; !ok {
				dumpRequest(t, req)
				t.Errorf("request test failed: %q\n", name)
				t.Fatalf("could not find error message\nmessage = %q\nerrors  = %v", msg, errs)
			}
		}
	}
}

func drain(t *testing.T, rc io.ReadCloser) (io.ReadCloser, io.ReadCloser) {
	if rc == nil {
		return nil, nil
	}

	var buf bytes.Buffer

	if _, err := buf.ReadFrom(rc); err != nil {
		t.Fatalf("unexpected bytes.Buffer.ReadFrom error: %s\n", err)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("unexpected io.ReadCloser.Close error: %s\n", err)
	}
	return ioutil.NopCloser(&buf), ioutil.NopCloser(bytes.NewReader(buf.Bytes()))
}

func dumpRequest(t *testing.T, r *http.Request) {
	t.Logf("%s %s", r.Method, r.URL.String())

	for k, v := range r.Header {
		t.Logf("%s: %s\n", k, strings.Join(v, "; "))
	}

	if r.Body != nil {
		defer r.Body.Close()

		var buf bytes.Buffer

		if _, err := io.Copy(&buf, r.Body); err != nil {
			t.Fatalf("unexpected io.Copy error: %s\n", err)
		}
		t.Logf("%s\n", buf.String())
	}
}

func dumpResponse(t *testing.T, r *http.Response) {
	t.Logf("%s\n", r.Status)

	for k, v := range r.Header {
		t.Logf("%s: %s\n", k, strings.Join(v, "; "))
	}

	if r.Body != nil {
		defer r.Body.Close()

		var buf bytes.Buffer

		if _, err := io.Copy(&buf, r.Body); err != nil {
			t.Fatalf("unexpected io.Copy error: %s\n", err)
		}
		t.Logf("%s\n", buf.String())
	}
}

func jsonBody(v interface{}) io.ReadCloser {
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(v)
	return ioutil.NopCloser(&buf)
}

func (c client) do(t *testing.T, r request) *http.Response {
	req, err := http.NewRequest(r.method, c.server.URL+r.uri, r.body)

	if err != nil {
		t.Fatalf("unexpected http.NewRequest error: %s\n", err)
	}

	if r.token != nil {
		req.Header.Set("Authorization", "Bearer "+hex.EncodeToString(r.token.Token))
	}

	req.Header.Set("Content-Type", r.contentType)

	reqbody1, reqbody2 := drain(t, req.Body)

	req.Body = reqbody1

	resp, err := c.server.Client().Do(req)

	if err != nil {
		t.Fatalf("unexpected http.Client.Do error: %s\n", err)
	}

	req.Body = reqbody2

	if resp.StatusCode != r.code {
		t.Errorf("request test failed: %q\n", r.name)
		t.Errorf("unexpected http response status, expected=%d, got=%q\n", r.code, resp.Status)
		dumpRequest(t, req)
		dumpResponse(t, resp)
	}

	respbody1, respbody2 := drain(t, resp.Body)

	resp.Body = respbody1

	if r.check != nil {
		r.check(t, r.name, req, resp)
	}

	resp.Body = respbody2
	return resp
}

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

func readFile(t *testing.T, name string) []byte {
	b, err := ioutil.ReadFile(filepath.Join("testdata", name))

	if err != nil {
		t.Fatalf("unexpected ReadFile error: %s\n", err)
	}
	return b
}

func openFile(t *testing.T, name string) *os.File {
	f, err := os.Open(filepath.Join("testdata", name))

	if err != nil {
		t.Fatalf("unexpected Open error: %s\n", err)
	}
	return f
}

func readAll(t *testing.T, r io.Reader) []byte {
	b, err := ioutil.ReadAll(r)

	if err != nil {
		t.Fatalf("unexpected ReadAll error: %s\n", err)
	}
	return b
}

func newClient(server *httptest.Server) client {
	return client{
		server: server,
	}
}

func TestMain(m *testing.M) {
	if err := os.MkdirAll(filepath.Join(os.TempDir(), "qemu"), os.FileMode(0755)); err != nil {
		fatalf("failed to create qemu tempdir: %s\n", err)
	}

	args := []string{
		"djinn-server",
		"-config",
		filepath.Join("testdata", "server.toml"),
	}

	api, config, ui, _ := serverutil.ParseFlags(args)

	srv, cfg, close_, err := serverutil.Init(config)

	if err != nil {
		fatalf("failed to initialize server: %s\n", err)
	}

	defer close_()

	db = cfg.DB()

	imageStore = cfg.Images().Store
	objectStore = cfg.Objects().Store

	users := user.NewStore(db)

	me, _, err := users.Create("me@example.com", "me", []byte("secret"))

	if err != nil {
		fatalf("failed to create user: %s\n", err)
	}

	you, _, err := users.Create("you@example.com", "you", []byte("secret"))

	if err != nil {
		fatalf("failed to create user: %s\n", err)
	}

	scope := oauth2.NewScope()

	for _, res := range oauth2.Resources {
		for _, perm := range oauth2.Permissions {
			scope.Add(res, perm)
		}
	}

	buildScope := oauth2.NewScope()
	buildScope.Add(oauth2.Build, oauth2.Read|oauth2.Write|oauth2.Delete)

	namespaceScope := oauth2.NewScope()
	namespaceScope.Add(oauth2.Namespace, oauth2.Read|oauth2.Write|oauth2.Delete)

	objectScope := oauth2.NewScope()
	objectScope.Add(oauth2.Object, oauth2.Read|oauth2.Write|oauth2.Delete)

	variableScope := oauth2.NewScope()
	variableScope.Add(oauth2.Variable, oauth2.Read|oauth2.Write|oauth2.Delete)

	keyScope := oauth2.NewScope()
	keyScope.Add(oauth2.Key, oauth2.Read|oauth2.Write|oauth2.Delete)

	myTok, err = oauth2.NewTokenStore(db, me).Create("My Token", scope)

	if err != nil {
		fatalf("failed to create token: %s\n", err)
	}

	buildRWD, err = oauth2.NewTokenStore(db, me).Create("Build Token", buildScope)

	if err != nil {
		fatalf("failed to create token: %s\n", err)
	}

	namespaceRWD, err = oauth2.NewTokenStore(db, me).Create("Namespace Token", namespaceScope)

	if err != nil {
		fatalf("failed to create token: %s\n", err)
	}

	objectRWD, err = oauth2.NewTokenStore(db, me).Create("Object Token", objectScope)

	if err != nil {
		fatalf("failed to create token: %s\n", err)
	}

	variableRWD, err = oauth2.NewTokenStore(db, me).Create("Variable Token", variableScope)

	if err != nil {
		fatalf("failed to create token: %s\n", err)
	}

	keyRWD, err = oauth2.NewTokenStore(db, me).Create("Key Token", keyScope)

	if err != nil {
		fatalf("failed to create token: %s\n", err)
	}

	yourTok, err = oauth2.NewTokenStore(db, you).Create("Your Token", scope)

	if err != nil {
		fatalf("failed to create token: %s\n", err)
	}

	serverutil.RegisterRoutes(cfg, api, ui, srv)

	server = httptest.NewServer(srv.Server.Handler)

	srv.Server = server.Config

	serverutil.Start(srv)

	os.Exit(m.Run())
}
