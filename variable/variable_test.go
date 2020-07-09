package variable

import (
	"database/sql"
	"database/sql/driver"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

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

var (
	variableCols = []string{
		"id",
		"user_id",
		"namespace_id",
		"key",
		"value",
		"created_at",
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
			"SELECT * FROM variables",
			[]query.Option{},
			sqlmock.NewRows(variableCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM variables WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(variableCols),
			[]driver.Value{1, 1, 1},
			[]database.Model{userModel},
		},
		{
			"SELECT * FROM variables WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(variableCols),
			[]driver.Value{1},
			[]database.Model{namespaceModel},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Errorf("tests[%d] - %s\n", i, errors.Cause(err))
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
			"SELECT * FROM variables WHERE (LOWER(key) LIKE $1)",
			[]query.Option{},
			sqlmock.NewRows(variableCols),
			[]driver.Value{"%gman%"},
			[]database.Model{},
		},
	}

	vals := []url.Values{
		url.Values(map[string][]string{"search": []string{"gman"}}),
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
			"SELECT * FROM variables",
			[]query.Option{},
			sqlmock.NewRows(variableCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM variables WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(variableCols),
			[]driver.Value{1, 1, 1},
			[]database.Model{userModel},
		},
		{
			"SELECT * FROM variables WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(variableCols),
			[]driver.Value{1},
			[]database.Model{namespaceModel},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Errorf("tests[%d] - %s\n", i ,errors.Cause(err))
		}

		store.User = nil
		store.Namespace = nil
	}
}

func Test_StoreCreate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectQuery(
		"^INSERT INTO variables \\((.+)\\) VALUES \\((.+)\\) RETURNING id$",
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

	if _, err := store.Create("PGADDR", "host=localhost port=5432"); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}

func Test_StoreDelete(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectExec(
		"^DELETE FROM variables WHERE \\(id IN \\(\\$1, \\$2, \\$3\\)\\)$",
	).WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(1, 2, 3); err != nil {
		t.Error(errors.Cause(err))
	}
}
