package build

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var artifactCols = []string{
	"build_id",
	"job_id",
	"hash",
	"source",
	"name",
	"size",
	"md5",
	"sha256",
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
			"SELECT * FROM build_artifacts",
			[]query.Option{},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM build_artifacts WHERE (LOWER(name) LIKE $1)",
			[]query.Option{model.Search("name", "example_artifact")},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{"%example_artifact%"},
			[]model.Model{},
		},
		{
			"SELECT * FROM build_artifacts WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{10},
			[]model.Model{&Build{ID: 10}},
		},
		{
			"SELECT * FROM build_artifacts WHERE (job_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{10},
			[]model.Model{&Job{ID: 10}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

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
			"SELECT * FROM build_artifacts",
			[]query.Option{},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM build_artifacts WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{10},
			[]model.Model{&Build{ID: 10}},
		},
		{
			"SELECT * FROM build_artifacts WHERE (job_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(artifactCols),
			[]driver.Value{10},
			[]model.Model{&Job{ID: 10}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

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

	a := &Artifact{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, artifactTable)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(a); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if a.ID != id {
		t.Fatalf("artifact id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, a.ID)
	}
}

func Test_ArtifactStoreUpdate(t *testing.T) {
	store, mock, close_ := artifactStore(t)
	defer close_()

	a := &Artifact{}

	expected := fmt.Sprintf(updateFmt, artifactTable)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(a.ID, 1))

	if err := store.Update(a); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
