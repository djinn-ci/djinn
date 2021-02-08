package object

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/andrewpillar/djinn/fs"
	"github.com/andrewpillar/djinn/database"
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
	args   []driver.Value
	models []database.Model
}

var (
	objectCols = []string{
		"user_id",
		"namespace_id",
		"hash",
		"name",
		"type",
		"size",
		"md5",
		"sha256",
		"deleted_at",
	}

	userModel = &user.User{
		ID:        1,
		Email:     "freemang@black-mesa.com",
		Username:  "freemang",
		Password:  []byte("secret"),
		CreatedAt: time.Now(),
	}

	namespaceModel = &namespace.Namespace{
		ID:     1,
		UserID: 1,
		RootID: sql.NullInt64{
			Int64: 1,
			Valid: true,
		},
		Name:       "blackmesa",
		Path:       "blackmesa",
		Level:      1,
		Visibility: namespace.Private,
		CreatedAt:  time.Now(),
	}
)

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
			"SELECT * FROM objects",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM objects WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{1, 1, 1},
			[]database.Model{userModel},
		},
		{
			"SELECT * FROM objects WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{1},
			[]database.Model{namespaceModel},
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

func Test_StoreIndex(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM objects WHERE (LOWER(name) LIKE $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{"%aperture%"},
			[]database.Model{},
		},
	}

	vals := []url.Values{
		url.Values(map[string][]string{"search": {"aperture"}}),
	}

	for i, test := range tests {
		paginate := strings.Replace(test.query, "*", "COUNT(*)", 1)
		paginateRows := sqlmock.NewRows([]string{"*"}).AddRow(1)

		mock.ExpectQuery(regexp.QuoteMeta(paginate)).WillReturnRows(paginateRows)
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, _, err := store.Index(vals[i], test.opts...); err != nil {
			t.Errorf("tests[%d] - %s\n", i, errors.Cause(err))
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
			"SELECT * FROM objects",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM objects WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{1, 1, 1},
			[]database.Model{userModel},
		},
		{
			"SELECT * FROM objects WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{1},
			[]database.Model{namespaceModel},
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

	store.blockStore = fs.NewNull()

	mock.ExpectQuery(
		"^INSERT INTO objects \\((.+)\\) VALUES \\((.+)\\) RETURNING id$",
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

	if _, err := store.Create(1, "some-script", "1a2b3c4d", bytes.NewReader([]byte("#!/bin/sh"))); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}

func Test_StoreDelete(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	store.blockStore = fs.NewNull()

	mock.ExpectExec(
		"^DELETE FROM objects WHERE \\(id IN \\((.+)\\)\\)$",
	).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.Delete(1, "1a2b3c4d"); err != nil {
		t.Errorf("unexpected Delete error: %s\n", errors.Cause(err))
	}
}
