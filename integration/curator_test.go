package integration

import (
	"database/sql"
	"io"
	"testing"
	"time"

	"djinn-ci.com/build"
	"djinn-ci.com/fs"
	"djinn-ci.com/log"
	"djinn-ci.com/integration/djinn"

	"github.com/andrewpillar/query"
)

type discardCloser struct {
	io.Writer
}

func (w discardCloser) Close() error {
	return nil
}

func discard() io.WriteCloser {
	return discardCloser{
		Writer: io.Discard,
	}
}

func Test_CurationOfPinnedBuild(t *testing.T) {
	u := users.get("gordon.freeman@black-mesa.com")

	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	b, err := djinn.SubmitBuild(cli, djinn.BuildParams{
		Manifest: djinn.Manifest{
			Driver: map[string]string{
				"type": "os",
			},
			Stages: []string{"collect-artifact"},
			Jobs: []djinn.ManifestJob{
				{
					Stage:     "collect-artifact",
					Commands:  []string{
						"/bin/sh -c pwd",
						`/bin/sh -c "head -c 1048576 /dev/urandom > rand.dat"`,
					},
					Artifacts: djinn.ManifestPassthrough{
						"rand.dat": "",
					},
				},
			},
		},
		Comment: "Test_CurationOfPinnedBuild",
		Tags:    []string{"os"},
	})

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

	if b.Status != djinn.Passed {
		t.Fatalf("unexpected status, expected=%q, got=%q\n", djinn.Passed, b.Status)
	}

	if err := b.Pin(cli); err != nil {
		t.Fatal(err)
	}

	q := query.Update(
		"users",
		query.Set("cleanup", query.Arg(1024)),
		query.Where("id", "=", query.Arg(u.ID)),
	)

	if _, err := db.Exec(q.Build(), q.Args()...); err != nil {
		t.Fatal(err)
	}

	cur := build.NewCurator(log.New(discard()), db, fs.NewNull())

	if err := cur.Invoke(); err != nil {
		t.Fatal(err)
	}

	q = query.Select(
		query.Columns("deleted_at"),
		query.From("build_artifacts"),
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("name", "=", query.Arg("rand.dat")),
	)

	var deletedAt sql.NullTime

	if err := db.QueryRow(q.Build(), q.Args()...).Scan(&deletedAt); err != nil {
		t.Fatal(err)
	}

	// Make sure artifact has not been deleted as it belongs to a pinned build.
	if deletedAt.Valid {
		t.Fatal("expected artifact of pinned build to not be deleted")
	}
}
