package integration

import (
	"os"
	"path/filepath"
	"testing"

	"djinn-ci.com/build"
	"djinn-ci.com/fs"
	"djinn-ci.com/integration/djinn"
	"djinn-ci.com/log"

	"github.com/andrewpillar/query"
)

func Test_BuildCuration(t *testing.T) {
	u := users.get("gordon.freeman@black-mesa.com")

	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	b := submitBuildAndWait(t, cli, djinn.Passed, djinn.BuildParams{
		Manifest: djinn.Manifest{
			Driver: map[string]string{
				"type": "os",
			},
			Stages: []string{"collect-artifact"},
			Jobs: []djinn.ManifestJob{
				{
					Stage: "collect-artifact",
					Commands: []string{
						"/bin/sh -c pwd",
						`/bin/sh -c "head -c 1048576 /dev/urandom > /tmp/mb0.dat"`,
						`/bin/sh -c "head -c 1048576 /dev/urandom > /tmp/mb1.dat"`,
						`/bin/sh -c "head -c 1048576 /dev/urandom > /tmp/mb2.dat"`,
						`/bin/sh -c "head -c 1048576 /dev/urandom > /tmp/mb3.dat"`,
						`/bin/sh -c "head -c 1048576 /dev/urandom > /tmp/mb4.dat"`,
					},
					Artifacts: djinn.ManifestPassthrough{
						"/tmp/mb0.dat": "",
						"/tmp/mb1.dat": "",
						"/tmp/mb2.dat": "",
						"/tmp/mb3.dat": "",
						"/tmp/mb4.dat": "",
					},
				},
			},
		},
		Comment: "Test_BuildCuration",
		Tags:    []string{"os"},
	})

	if err := b.Pin(cli); err != nil {
		t.Fatal(err)
	}

	q := query.Update(
		"users",
		query.Set("cleanup", query.Arg(3<<20)),
		query.Where("id", "=", query.Arg(u.ID)),
	)

	if _, err := db.Exec(q.Build(), q.Args()...); err != nil {
		t.Fatal(err)
	}

	f, err := os.Create(filepath.Join("testdata", "log", "curator.log"))

	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	log := log.New(f)
	log.SetLevel("debug")

	cur := build.NewCurator(log, db, fs.NewNull())

	if err := cur.Invoke(); err != nil {
		t.Fatal(err)
	}

	// Make sure only the artifacts we want remain.
	q = query.Select(
		query.Count("id"),
		query.From("build_artifacts"),
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("deleted_at", "IS NOT", query.Lit("NULL")),
	)

	var count int64

	if err := db.QueryRow(q.Build(), q.Args()...).Scan(&count); err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Fatalf("unexpected artifacts after curation, expected=%d, got=%d\n", 0, count)
	}

	if err := b.Unpin(cli); err != nil {
		t.Fatal(err)
	}

	if err := cur.Invoke(); err != nil {
		t.Fatal(err)
	}

	// Check to see if the artifacts have been deleted.
	q = query.Select(
		query.Count("id"),
		query.From("build_artifacts"),
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("deleted_at", "IS NOT", query.Lit("NULL")),
	)

	if err := db.QueryRow(q.Build(), q.Args()...).Scan(&count); err != nil {
		t.Fatal(err)
	}

	if count != 2 {
		t.Fatalf("unexpected artifacts after curation, expected=%d, got=%d\n", 2, count)
	}
}
