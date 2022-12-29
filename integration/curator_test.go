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
						`/bin/sh -c "head -c 1048576 /dev/urandom > mb0.dat"`,
						`/bin/sh -c "head -c 1048576 /dev/urandom > mb1.dat"`,
						`/bin/sh -c "head -c 1048576 /dev/urandom > mb2.dat"`,
						`/bin/sh -c "head -c 1048576 /dev/urandom > mb3.dat"`,
						`/bin/sh -c "head -c 1048576 /dev/urandom > mb4.dat"`,
					},
					Artifacts: djinn.ManifestPassthrough{
						"mb0.dat": "",
						"mb1.dat": "",
						"mb2.dat": "",
						"mb3.dat": "",
						"mb4.dat": "",
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
		query.Set("cleanup", query.Arg(3*1<<20)),
		query.Where("id", "=", query.Arg(u.ID)),
	)

	if _, err := db.Exec(q.Build(), q.Args()...); err != nil {
		t.Fatal(err)
	}

	cur := build.NewCurator(log.New(discard()), db, fs.NewNull())

	if err := cur.Invoke(); err != nil {
		t.Fatal(err)
	}

	// Make sure only the artifacts we want remain.
	q = query.Select(
		query.Columns("id"),
		query.From("build_artifacts"),
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("deleted_at", "IS NOT", query.Lit("NULL")),
	)

	ids := make([]string, 0)

	rows, err := db.QueryRow(q.Build(), q.Args()

	if err != nil {
		t.Fatal(err)
	}

	var id string

	for rows.Next() {
		if err := row.Scan(&id); err != nil {
			ids = append(ids, id)
		}
	}

	if len(ids) != 2 {
		t.Fatalf("unexpected artifacts after curation, expected=%d, got=%d\n", 2, len(ids))
	}
}
