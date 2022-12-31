package integration

import (
	"bufio"
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/fs"
	"djinn-ci.com/integration/djinn"
	"djinn-ci.com/oauth2"
	"djinn-ci.com/serverutil"
	"djinn-ci.com/user"
	"djinn-ci.com/workerutil"

	goredis "github.com/go-redis/redis"
)

var (
	users  *usermap
	tokens *tokenmap
)

type usermap struct {
	mu *sync.RWMutex
	m  map[string]*user.User
}

func (m *usermap) put(u *user.User) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.m == nil {
		m.m = make(map[string]*user.User)
	}
	m.m[u.Email] = u
}

func (m *usermap) get(email string) *user.User {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.m[email]
}

type tokenmap struct {
	mu *sync.RWMutex
	m  map[string]*oauth2.Token
}

func (m *tokenmap) put(t *oauth2.Token) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.m == nil {
		m.m = make(map[string]*oauth2.Token)
	}
	m.m[t.Name] = t
}

func (m *tokenmap) get(name string) *oauth2.Token {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.m[name]
}

var (
	db     database.Pool
	redis  *goredis.Client
	server *httptest.Server
	aesgcm *crypto.AESGCM

	imagestore  fs.Store
	objectstore fs.Store

	apiEndpoint string
)

func fatalf(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

func mkuser(db database.Pool, email, username string, password []byte) *user.User {
	users := user.Store{Pool: db}

	u, _, err := users.Create(user.Params{
		Email:    email,
		Username: username,
		Password: string(password),
	})

	if err != nil {
		fatalf("mkuser error: %s\n", err)
	}
	return u
}

func mktoken(db database.Pool, u *user.User, name, scope string) *oauth2.Token {
	sc, err := oauth2.UnmarshalScope(scope)

	if err != nil {
		fatalf("mktoken error: %s\n", err)
	}

	tokens := oauth2.TokenStore{Pool: db}

	tok, err := tokens.Create(oauth2.TokenParams{
		UserID: u.ID,
		Name:   name,
		Scope:  sc,
	})

	if err != nil {
		fatalf("mktoken error: %s\n", err)
	}
	return tok
}

func submitBuildAndWait(t *testing.T, cli *djinn.Client, status djinn.Status, p djinn.BuildParams) *djinn.Build {
	b, err := djinn.SubmitBuild(cli, p)

	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(time.Second):
				if err := b.Get(cli); err != nil {
					t.Fatal(err)
				}

				if b.Status != djinn.Queued && b.Status != djinn.Running {
					done <- struct{}{}
					return
				}
			}
		}
	}()

	select {
	case <-time.After(time.Second * 5):
		done <- struct{}{}
	case <-done:
	}

	if b.Status != status {
		t.Fatalf("unexpected status, expected=%q, got=%q\n", status, b.Status)
	}
	return b
}

// setupServer sets up the server for integration testing. This returns a
// callback for cleaning up the resources used by the server.
func setupServer(cfgdir string) func() {
	cfgfile := filepath.Join(cfgdir, "server.conf")

	api, config, ui, _ := serverutil.ParseFlags([]string{"djinn-server", "-config", cfgfile})

	bgctx := context.Background()

	qctx, qcancel := context.WithCancel(bgctx)

	srv, close, err := serverutil.Init(qctx, config)

	if err != nil {
		fatalf("failed to initialize server: %s\n", err)
	}

	db = srv.DB
	redis = srv.Redis

	users = &usermap{
		mu: &sync.RWMutex{},
	}

	uu := []*user.User{
		{Email: "gordon.freeman@black-mesa.com", Username: "gordon.freeman", Password: []byte("secret")},
		{Email: "eli.vance@black-mesa.com", Username: "eli.vance", Password: []byte("secret")},
		{Email: "wallace.breen@black-mesa.com", Username: "wallace.breen", Password: []byte("secret")},
		{Email: "doug.rattman@aperturescience.com", Username: "doug.rattman", Password: []byte("secret")},
		{Email: "henry.bloggs@aperturescience.com", Username: "henry.bloggs", Password: []byte("secret")},
	}

	for i, u0 := range uu {
		u := mkuser(db, u0.Email, u0.Username, u0.Password)

		users.put(u)
		uu[i] = u
	}

	tokens = &tokenmap{
		mu: &sync.RWMutex{},
	}

	for _, u := range uu {
		sc := oauth2.NewScope()

		for _, res := range oauth2.Resources {
			for _, perm := range oauth2.Permissions {
				sc.Add(res, perm)
			}
		}
		tokens.put(mktoken(db, u, u.Username, sc.String()))
	}

	ch := make(chan os.Signal, 1)

	serverutil.RegisterRoutes(api, ui, srv)
	serverutil.Start(srv, ch)

	server = httptest.NewServer(srv.Server.Handler)
	aesgcm = srv.AESGCM

	imagestore = srv.Images
	objectstore = srv.Objects

	apiEndpoint = server.URL + "/api"
	env.DJINN_API_SERVER = apiEndpoint

	ctx, cancel := context.WithTimeout(bgctx, time.Duration(time.Second*15))

	return func() {
		defer qcancel()
		defer close()
		defer cancel()
		defer server.Close()
		defer srv.Shutdown(ctx)
	}
}

func setupWorker(cfgdir string) func() {
	cfgfile := filepath.Join(cfgdir, "worker.conf")

	config, driver, _ := workerutil.ParseFlags([]string{
		"djinn-worker", "-config", cfgfile, "-driver", filepath.Join(cfgdir, "driver.conf"),
	})

	w, close, err := workerutil.Init(config, driver)

	if err != nil {
		fatalf("failed to initialize worker: %s\n", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	workerutil.Start(ctx, w)

	return func() {
		defer close()
		defer cancel()
	}
}

func TestMain(m *testing.M) {
	cfgdir := os.Getenv("INTEGRATION_CONFIG_DIR")

	if cfgdir == "" {
		fmt.Println("skipping integration tests, INTEGRATION_CONFIGR_DIR not set")
		return
	}

	tmp := os.TempDir()

	for _, dir := range []string{"artifacts", "images", "objects"} {
		if err := os.MkdirAll(filepath.Join(tmp, dir), os.FileMode(0755)); err != nil {
			fatalf("failed to create directory %s: %s\n", dir, err)
		}
	}

	logs := []string{
		filepath.Join("testdata", "log", "server.log"),
		filepath.Join("testdata", "log", "worker.log"),
	}

	for _, log := range logs {
		if _, err := os.Stat(log); err == nil {
			if err := os.Truncate(log, 0); err != nil {
				fatalf("failed to truncate %s log: %s\n", log, err)
			}
		}
	}

	srvcleanup := setupServer(cfgdir)
	wrkcleanup := setupWorker(cfgdir)

	defer srvcleanup()
	defer wrkcleanup()

	code := m.Run()

	for _, log := range logs {
		func(log string) {
			f, err := os.Open(log)

			if err != nil {
				fatalf("failed to open log: %s\n", err)
			}

			defer f.Close()

			sc := bufio.NewScanner(f)

			for sc.Scan() {
				line := sc.Text()

				parts := strings.SplitN(line, " ", 4)

				if len(parts) >= 3 {
					if parts[2] == "ERROR" {
						fmt.Fprintf(os.Stderr, "error in %s: %s\n", log, line)
						code = 1
					}
				}
			}

			if err := sc.Err(); err != nil {
				fatalf("failed to scan log: %s\n", err)
			}
		}(log)
	}

	if code != 0 {
		os.Exit(code)
	}
}
