package key

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

type testQuery struct {
	query  string
	opts   []query.Option
	rows   *sqlmock.Rows
	args   []driver.Value
	models []database.Model
}

var keyCols = []string{
	"user_id",
	"namespace_id",
	"name",
	"key",
	"config",
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
			"SELECT * FROM keys",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM keys WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{1, 1, 1},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM keys WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{1},
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
			"SELECT * FROM keys",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM keys WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{1, 1, 1},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM keys WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{1},
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

	block, err := crypto.NewBlock([]byte("some-supersecret"))

	if err != nil {
		t.Fatal(errors.Cause(err))
	}

	store.block = block

	mock.ExpectQuery(
		"^INSERT INTO keys \\([\\w+, ]+\\) VALUES \\([\\$\\d+, ]+\\) RETURNING id$",
	).WillReturnRows(mock.NewRows([]string{"id"}).AddRow(10))

	if _, err := store.Create("id_rsa", "=AAAAAA", ""); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}

func Test_StoreUpdate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectExec(
		"^UPDATE keys SET namespace_id = \\$1, config = \\$2 WHERE \\(id = \\$3\\)$",
	).WithArgs(1, "", 10).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.Update(10, 1, ""); err != nil {
		t.Errorf("unexpected Update error: %s\n", errors.Cause(err))
	}
}

func Test_StoreDelete(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectExec(
		"^DELETE FROM keys WHERE \\(id IN \\(\\$1\\)\\)$",
	).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.Delete(10); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
