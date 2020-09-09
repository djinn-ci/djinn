package image

import (
	"bytes"
	sqldriver "database/sql/driver"
	"regexp"
	"testing"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/driver"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

type testQuery struct {
	query  string
	opts   []query.Option
	rows   *sqlmock.Rows
	args   []sqldriver.Value
	models []database.Model
}

var imageCols = []string{
	"user_id",
	"namespace_id",
	"driver",
	"hash",
	"name",
	"created_at",
}

func store(t *testing.T) (*Store, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_StoreAll(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM images",
			[]query.Option{},
			sqlmock.NewRows(imageCols),
			[]sqldriver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM images WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(imageCols),
			[]sqldriver.Value{1, 1, 1},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM images WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(imageCols),
			[]sqldriver.Value{1},
			[]database.Model{&namespace.Namespace{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.User = nil
		store.Namespace = nil
	}
}

func Test_StoreGet(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM images",
			[]query.Option{},
			sqlmock.NewRows(imageCols),
			[]sqldriver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM images WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(imageCols),
			[]sqldriver.Value{1, 1, 1},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM images WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(imageCols),
			[]sqldriver.Value{1},
			[]database.Model{&namespace.Namespace{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.User = nil
		store.Namespace = nil
	}
}

func Test_StoreCreate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	store.blockStore = block.NewNull()

	mock.ExpectQuery(
		"^INSERT INTO images \\([\\w+, ]+\\) VALUES \\([\\$\\d+, ]+\\) RETURNING id$",
	).WillReturnRows(mock.NewRows([]string{"id"}).AddRow(10))

	if _, err := store.Create("1a2b3c4d", "dev", driver.QEMU, bytes.NewBufferString("some image")); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}

func Test_StoreDelete(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	store.blockStore = block.NewNull()

	mock.ExpectExec(
		"^DELETE FROM images WHERE \\(id IN \\(\\$1\\)\\)$",
	).WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(1, driver.QEMU, "1a2b3c4d"); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
