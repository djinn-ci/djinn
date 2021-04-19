package build

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"database/sql/driver"
	"io"
	"io/ioutil"
	"testing"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var artifactCols = []string{
	"id",
	"build_id",
	"job_id",
	"hash",
	"source",
	"name",
	"size",
	"md5",
	"sha256",
}

type discardCollector struct {
	w io.Writer
}

func newDiscardCollector() *discardCollector {
	return &discardCollector{
		w: ioutil.Discard,
	}
}

func (c *discardCollector) Collect(_ string, r io.Reader) (int64, error) {
	return io.Copy(c.w, r)
}

func artifactStore(t *testing.T) (*ArtifactStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewArtifactStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_ArtifactStoreAll(t *testing.T) {
	store, mock, close_ := artifactStore(t)
	defer close_()

	tests := []testQuery{
		{
			"^SELECT \\* FROM build_artifacts$",
			[]query.Option{},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT \\* FROM build_artifacts WHERE \\(LOWER\\(name\\) LIKE \\$1\\)$",
			[]query.Option{database.Search("name", "example_artifact")},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{"%example_artifact%"},
			[]database.Model{},
		},
		{
			"SELECT \\* FROM build_artifacts WHERE \\(build_id = \\$1\\)$",
			[]query.Option{},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{10},
			[]database.Model{&Build{ID: 10}},
		},
		{
			"SELECT \\* FROM build_artifacts WHERE \\(job_id = \\$1\\)$",
			[]query.Option{},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{10},
			[]database.Model{&Job{ID: 10}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(test.query).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.Build = nil
		store.Job = nil
	}
}

func Test_ArtifactStoreGet(t *testing.T) {
	store, mock, close_ := artifactStore(t)
	defer close_()

	tests := []testQuery{
		{
			"^SELECT \\* FROM build_artifacts$",
			[]query.Option{},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"^SELECT \\* FROM build_artifacts WHERE \\(build_id = \\$1\\)$",
			[]query.Option{},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{10},
			[]database.Model{&Build{ID: 10}},
		},
		{
			"^SELECT \\* FROM build_artifacts WHERE \\(job_id = \\$1\\)$",
			[]query.Option{},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{10},
			[]database.Model{&Job{ID: 10}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(test.query).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.Build = nil
		store.Job = nil
	}
}

func Test_ArtifactStoreCreate(t *testing.T) {
	store, mock, close_ := artifactStore(t)
	defer close_()

	mock.ExpectQuery(
		"^INSERT INTO build_artifacts \\([\\w+, ]+\\) VALUES \\([\\$\\d+, ]+\\) RETURNING id$",
	).WillReturnRows(mock.NewRows([]string{"id"}).AddRow(10))

	if _, err := store.Create("1a2b3c4d", "build.out", "build.out"); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}

func Test_ArtifactStoreCollect(t *testing.T) {
	store, mock, close_ := artifactStore(t)
	defer close_()

	store.Bind(&Build{ID: 1})
	store.collector = newDiscardCollector()

	tmp := bytes.NewBufferString("some artifact")
	buf := &bytes.Buffer{}

	mock.ExpectQuery(
		"^SELECT \\* FROM build_artifacts WHERE \\(build_id = \\$1 AND name = \\$2\\)$",
	).WillReturnRows(sqlmock.NewRows(artifactCols).AddRow(1, 1, 1, "1a2b3c4d", "build.out", "build.out", nil, nil, nil))

	md5 := md5.New()
	sha256 := sha256.New()

	if _, err := io.Copy(io.MultiWriter(md5, sha256, buf), tmp); err != nil {
		t.Fatalf("unexpected io.Copy error: %s\n", err)
	}

	mock.ExpectExec(
		"^UPDATE build_artifacts SET size = \\$1, md5 = \\$2, sha256 = \\$3 WHERE \\(id = \\$4\\)$",
	).WithArgs(buf.Len(), md5.Sum(nil), sha256.Sum(nil), 1).WillReturnResult(
		sqlmock.NewResult(0, 1),
	)

	if _, err := store.Collect("build.out", buf); err != nil {
		t.Errorf("unexpected Collect error: %s\n", errors.Cause(err))
	}
}
