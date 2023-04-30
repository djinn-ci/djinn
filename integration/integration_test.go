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

	"djinn-ci.com/auth"
	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/integration/djinn"
	"djinn-ci.com/oauth2"
	"djinn-ci.com/serverutil"
	"djinn-ci.com/user"
	"djinn-ci.com/workerutil"

	"github.com/andrewpillar/fs"

	goredis "github.com/go-redis/redis"
)

type syncMap[T any] struct {
	mu sync.RWMutex
	m  map[string]T
}

func (m *syncMap[T]) put(key string, val T) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.m == nil {
		m.m = make(map[string]T)
	}
	m.m[key] = val
}

func (m *syncMap[T]) get(key string) T {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.m[key]
}

var (
	users  *syncMap[*auth.User]
	tokens *syncMap[*oauth2.Token]

	server *httptest.Server

	db       *database.Pool
	redis    *goredis.Client
	aesgcm   *crypto.AESGCM
	imageFS  fs.FS
	objectFS fs.FS
)

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

func mkuser(ctx context.Context, db *database.Pool, email, username string, password []byte) *auth.User {
	users := user.Store{
		Store: user.NewStore(db),
	}

	u, err := users.Create(ctx, &user.Params{
		Email:    email,
		Username: username,
		Password: string(password),
	})

	if err != nil {
		fatalf("mkuser: %s\n", err)
	}
	return u
}

func mktoken(ctx context.Context, db *database.Pool, u *auth.User, name string, sc oauth2.Scope) *oauth2.Token {
	tokens := oauth2.TokenStore{
		Store: oauth2.NewTokenStore(db),
	}

	tok, err := tokens.Create(ctx, &oauth2.TokenParams{
		User:  u,
		Name:  name,
		Scope: sc,
	})

	if err != nil {
		fatalf("mktoken: %s\n", err)
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

func setupServer(ctx context.Context, cfgdir string) func(context.Context) {
	args := []string{
		"djinn-server",
		"-config", filepath.Join(cfgdir, "server.conf"),
	}

	api, config, ui, _ := serverutil.ParseFlags(args)

	qctx, qcancel := context.WithCancel(ctx)

	srv, close, err := serverutil.Init(qctx, config)

	if err != nil {
		fatalf("setupServer: %s\n", err)
	}

	db = srv.DB
	redis = srv.Redis
	aesgcm = srv.AESGCM
	imageFS = srv.Images
	objectFS = srv.Objects

	ch := make(chan os.Signal, 1)

	serverutil.RegisterRoutes(api, ui, srv)
	serverutil.Start(srv, ch)

	server = httptest.NewServer(srv.Server.Handler)

	env.DJINN_API_SERVER = server.URL + "/api"

	return func(ctx context.Context) {
		qcancel()
		close()
		server.Close()
		srv.Shutdown(ctx)
	}
}

func setupWorker(ctx context.Context, cfgdir string) func() {
	args := []string{
		"djinn-worker",
		"-config", filepath.Join(cfgdir, "worker.conf"),
		"-driver", filepath.Join(cfgdir, "driver.conf"),
	}

	config, driver, _ := workerutil.ParseFlags(args)

	wrk, close, err := workerutil.Init(config, driver)

	if err != nil {
		fatalf("setupWorker: %s\n", err)
	}

	workerutil.Start(ctx, wrk)

	return close
}

func TestMain(m *testing.M) {
	cfgdir := os.Getenv("INTEGRATION_CONFIG_DIR")

	if cfgdir == "" {
		fmt.Println("skipping integration tests, INTEGRATION_CONFIGR_DIR not set")
		return
	}

	tmp := os.TempDir()

	for _, dir := range []string{"artifacts", "images", "objects"} {
		if err := os.MkdirAll(filepath.Join(tmp, dir), fs.FileMode(0750)); err != nil {
			fatalf("%s\n", err)
		}
	}

	logs := []string{
		filepath.Join("testdata", "log", "server.log"),
		filepath.Join("testdata", "log", "worker.log"),
	}

	for _, log := range logs {
		if _, err := os.Stat(log); err == nil {
			if err := os.Truncate(log, 0); err != nil {
				fatalf("%s\n", err)
			}
		}
	}

	ctx := context.Background()

	srvCleanup := setupServer(ctx, cfgdir)
	wrkCleanup := setupWorker(ctx, cfgdir)

	defer srvCleanup(ctx)
	defer wrkCleanup()

	users = &syncMap[*auth.User]{}
	tokens = &syncMap[*oauth2.Token]{}

	uu := []*auth.User{
		{Email: "gordon.freeman@black-mesa.com", Username: "gordon.freeman"},
		{Email: "eli.vance@black-mesa.com", Username: "eli.vance"},
		{Email: "wallace.breen@black-mesa.com", Username: "wallace.breen"},
		{Email: "doug.rattman@aperturescience.com", Username: "doug.rattman"},
		{Email: "henry.bloggs@aperturescience.com", Username: "henry.bloggs"},
	}

	for _, u := range uu {
		u = mkuser(ctx, db, u.Email, u.Username, []byte("secret"))
		users.put(u.Username, u)

		sc := oauth2.NewScope()

		for _, res := range oauth2.Resources {
			for _, perm := range oauth2.Permissions {
				sc.Add(res, perm)
			}
		}
		tokens.put(u.Username, mktoken(ctx, db, u, u.Username, sc))
	}

	code := m.Run()

	for _, log := range logs {
		func(log string) {
			f, err := os.Open(log)

			if err != nil {
				fatalf("%s\n", err)
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
