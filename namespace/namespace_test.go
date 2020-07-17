package namespace

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var (
	namespaceCols = []string{
		"user_id",
		"root_id",
		"parent_id",
		"name",
		"path",
		"description",
		"level",
		"visibility",
	}

	userCols = []string{
		"email",
		"username",
		"password",
		"updated_at",
		"deleted_at",
	}
)

func store(t *testing.T) (*Store, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_StoreIndex(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM namespaces WHERE (LOWER(path) LIKE $1)",
			[]query.Option{},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{"%blackmesa%"},
			[]database.Model{},
		},
	}

	vals := []url.Values{
		url.Values(map[string][]string{"search": []string{"blackmesa"}}),
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

func Test_StoreAll(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM namespaces",
			[]query.Option{},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM namespaces WHERE (parent_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1},
			[]database.Model{&Namespace{ID: 1}},
		},
		{
			"SELECT * FROM namespaces WHERE (LOWER(path) LIKE $1)",
			[]query.Option{database.Search("path", "example_path")},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{"%example_path%"},
			[]database.Model{},
		},
		{
			"SELECT * FROM namespaces WHERE (user_id = $1 OR root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $2))) AND (LOWER(path) LIKE $3)",
			[]query.Option{database.Search("path", "example_path")},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1, 1, "%example_path%"},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM namespaces WHERE (user_id = $1 OR root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $2)))",
			[]query.Option{SharedWith(&user.User{ID: 1})},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1, 1},
			[]database.Model{},
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

func Test_StoreGet(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM namespaces",
			[]query.Option{},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM namespaces WHERE (user_id = $1 OR root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $2)))",
			[]query.Option{},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1, 1},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM namespaces WHERE (parent_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1},
			[]database.Model{&Namespace{ID: 1}},
		},
		{
			"SELECT * FROM namespaces WHERE (LOWER(path) LIKE $1)",
			[]query.Option{database.Search("path", "blackmesa")},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{"%blackmesa%"},
			[]database.Model{},
		},
		{
			"SELECT * FROM namespaces WHERE (user_id = $1 OR root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $2))) AND (LOWER(path) LIKE $3)",
			[]query.Option{database.Search("path", "blackmesa")},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1, 1, "%blackmesa%"},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM namespaces WHERE (root_id = (SELECT root_id FROM namespaces WHERE (id = $1)))",
			[]query.Option{query.WhereQuery("root_id", "=", SelectRootID(1))},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1},
			[]database.Model{},
		},
		{
			"SELECT * FROM namespaces WHERE (root_id = (SELECT root_id FROM namespaces WHERE (id = $1)) AND id = (SELECT root_id FROM namespaces WHERE (id = $2)))",
			[]query.Option{
				query.WhereQuery("root_id", "=", SelectRootID(1)),
				query.WhereQuery("id", "=", SelectRootID(1)),
			},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1, 1},
			[]database.Model{},
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

func Test_StoreGetByPath(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT * FROM namespaces WHERE (path = $1)"),
	).WithArgs(
		[]driver.Value{"blackmesa"}...,
	).WillReturnRows(
		sqlmock.NewRows(namespaceCols),
	)

	mock.ExpectQuery(fmt.Sprintf(insertFmt, table)).WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectExec(fmt.Sprintf(updateFmt, table)).WillReturnResult(sqlmock.NewResult(1, 1))

	if _, err := store.GetByPath("blackmesa"); err != nil {
		t.Fatal(errors.Cause(err))
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT * FROM users WHERE (username = $1)"),
	).WithArgs(
		[]driver.Value{"freemang"}...,
	).WillReturnRows(sqlmock.NewRows(userCols))

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT * FROM namespaces WHERE (path = $1)"),
	).WithArgs(
		[]driver.Value{"blackmesa"}...,
	).WillReturnRows(sqlmock.NewRows(namespaceCols))

	mock.ExpectQuery(
		fmt.Sprintf(insertFmt, table),
	).WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	mock.ExpectExec(fmt.Sprintf(updateFmt, table)).WillReturnResult(sqlmock.NewResult(1, 1))

	if _, err := store.GetByPath("blackmesa@freemang"); err != nil {
		t.Fatal(errors.Cause(err))
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT * FROM namespaces WHERE (path = $1)"),
	).WithArgs(
		[]driver.Value{"blackmesa/blueshift"}...,
	).WillReturnRows(sqlmock.NewRows(namespaceCols))

	mock.ExpectQuery(fmt.Sprintf(insertFmt, table)).WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectExec(fmt.Sprintf(updateFmt, table)).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(fmt.Sprintf(insertFmt, table)).WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))
	mock.ExpectExec(fmt.Sprintf(updateFmt, table)).WillReturnResult(sqlmock.NewResult(1, 1))

	if _, err := store.GetByPath("blackmesa/blueshift"); err != nil {
		t.Fatal(errors.Cause(err))
	}
}

func Test_StoreCreate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectQuery(
		"^SELECT \\* FROM namespaces WHERE \\(path = \\$1\\)$",
	).WillReturnRows(sqlmock.NewRows(namespaceCols))

	mock.ExpectQuery(
		"^INSERT INTO namespaces \\((.+)\\) VALUES \\((.+)\\) RETURNING id$",
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	mock.ExpectExec(
		"^UPDATE namespaces SET root_id = \\$1 WHERE \\(id = \\$2\\)$",
	).WithArgs(1, 1).WillReturnResult(sqlmock.NewResult(0, 1))

	if _, err := store.Create("", "project", "", Internal); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}

func Test_StoreUpdate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectQuery(
		"^SELECT \\* FROM namespaces WHERE \\(id = \\(SELECT parent_id FROM namespaces WHERE \\(id = \\$1\\)\\)\\)$",
	).WithArgs(1).WillReturnRows(sqlmock.NewRows(namespaceCols))

	mock.ExpectExec(
		"^UPDATE namespaces SET visibility = \\$1 WHERE \\(root_id = \\$2\\)$",
	).WithArgs(Public, 1).WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(
		"^UPDATE namespaces SET name = \\$1, description = \\$2, visibility = \\$3 WHERE \\(id = \\$4\\)$",
	).WithArgs("project", "", Public, 1).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.Update(1, "project", "", Public); err != nil {
		t.Fatal(errors.Cause(err))
	}
}

func Test_StoreDelete(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectExec(
		regexp.QuoteMeta("DELETE FROM namespaces WHERE (root_id IN ($1, $2, $3))"),
	).WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(1, 2, 3); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
