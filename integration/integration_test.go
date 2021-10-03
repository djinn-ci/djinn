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
	"djinn-ci.com/fs"
	"djinn-ci.com/oauth2"
	"djinn-ci.com/serverutil"
	"djinn-ci.com/user"

	goredis "github.com/go-redis/redis"

	"github.com/jmoiron/sqlx"
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
	db     *sqlx.DB
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

func mkuser(db *sqlx.DB, username, email string, password []byte) *user.User {
	u, _, err := user.NewStore(db).Create(username, email, password)

	if err != nil {
		fatalf("mkuser error: %s\n", err)
	}
	return u
}

func mktoken(db *sqlx.DB, u *user.User, name, scope string) *oauth2.Token {
	sc, err := oauth2.UnmarshalScope(scope)

	if err != nil {
		fatalf("mktoken error: %s\n", err)
	}

	tok, err := oauth2.NewTokenStore(db, u).Create(name, sc)

	if err != nil {
		fatalf("mktoken error: %s\n", err)
	}
	return tok
}

func TestMain(m *testing.M) {
	cfgfile := os.Getenv("INTEGRATION_CONFIG")

	if cfgfile == "" {
		fmt.Println("skipping integration tests, INTEGRATION_CONFIG not set")
		return
	}

	tmp := os.TempDir()

	for _, dir := range []string{"artifacts", "images", "objects"} {
		if err := os.MkdirAll(filepath.Join(tmp, dir), os.FileMode(0755)); err != nil {
			fatalf("failed to create directory %s: %s\n", dir, err)
		}
	}

	logname := filepath.Join("testdata", "server.log")

	if _, err := os.Stat(logname); err == nil {
		if err := os.Truncate(logname, 0); err != nil {
			fatalf("failed to truncate server log: %s\n", err)
		}
	}

	api, config, ui, _ := serverutil.ParseFlags([]string{"djinn-server", "-config", cfgfile})

	bgctx := context.Background()

	qctx, qcancel := context.WithCancel(bgctx)
	defer qcancel()

	srv, cfg, close_, err := serverutil.Init(qctx, config)

	if err != nil {
		fatalf("failed to initialize server: %s\n", err)
	}

	defer close_()

	db = cfg.DB()
	redis = cfg.Redis()
	server = httptest.NewServer(srv.Server.Handler)
	aesgcm = cfg.BlockCipher()
	defer server.Close()

	imagestore = cfg.Images().Store
	objectstore = cfg.Objects().Store

	apiEndpoint = server.URL + "/api"

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

	serverutil.RegisterRoutes(cfg, api, ui, srv)
	serverutil.Start(srv, ch)

	ctx, cancel := context.WithTimeout(bgctx, time.Duration(time.Second*15))
	defer cancel()

	code := m.Run()

	srv.Shutdown(ctx)

	f, err := os.Open(filepath.Join("testdata", "server.log"))

	if err != nil {
		fatalf("failed to open server log: %s\n", err)
	}

	sc := bufio.NewScanner(f)

	for sc.Scan() {
		line := sc.Text()

		parts := strings.SplitN(line, " ", 4)

		if len(parts) >= 3 {
			if parts[2] == "ERROR" {
				fmt.Fprintf(os.Stderr, line)
				code = 1
			}
		}
	}

	if err := sc.Err(); err != nil {
		fatalf("failed to scan log: %s\n", err)
	}
	os.Exit(code)
}
