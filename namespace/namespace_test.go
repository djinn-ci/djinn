package namespace

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
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

func store(t *testing.T) (Store, sqlmock.Sqlmock, func() error) {
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
			"SELECT * FROM namespaces WHERE (path LIKE $1)",
			[]query.Option{},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{"%blackmesa%"},
			[]model.Model{},
		},
	}

	vals := []url.Values{
		url.Values(map[string][]string{"search": []string{"blackmesa"}}),
	}

	for i, test := range tests {
		paginate := strings.Replace(test.query, "*", "COUNT(*)", 1)
		paginateRows := sqlmock.NewRows([]string{"*"}).AddRow(1)

		mock.ExpectPrepare(regexp.QuoteMeta(paginate)).ExpectQuery().WillReturnRows(paginateRows)
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, _, err := store.Index(vals[i], test.opts...); err != nil {
			t.Fatal(errors.Cause(err))
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
			[]model.Model{},
		},
		{
			"SELECT * FROM namespaces WHERE (parent_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1},
			[]model.Model{&Namespace{ID: 1}},
		},
		{
			"SELECT * FROM namespaces WHERE (path LIKE $1)",
			[]query.Option{model.Search("path", "example_path")},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{"%example_path%"},
			[]model.Model{},
		},
		{
			"SELECT * FROM namespaces WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3) AND (path LIKE $4)",
			[]query.Option{model.Search("path", "example_path")},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1, 1, 1, "%example_path%"},
			[]model.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM namespaces WHERE (user_id = $1 OR root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $2)))",
			[]query.Option{SharedWith(&user.User{ID: 1})},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1, 1},
			[]model.Model{},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatal(errors.Cause(err))
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
			[]model.Model{},
		},
		{

			"SELECT * FROM namespaces WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1, 1, 1},
			[]model.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM namespaces WHERE (parent_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1},
			[]model.Model{&Namespace{ID: 1}},
		},
		{
			"SELECT * FROM namespaces WHERE (path LIKE $1)",
			[]query.Option{model.Search("path", "blackmesa")},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{"%blackmesa%"},
			[]model.Model{},
		},
		{

			"SELECT * FROM namespaces WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3) AND (path LIKE $4)",
			[]query.Option{model.Search("path", "blackmesa")},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1, 1, 1, "%blackmesa%"},
			[]model.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM namespaces WHERE (root_id = (SELECT root_id FROM namespaces WHERE (id = $1)))",
			[]query.Option{query.WhereQuery("root_id", "=", SelectRootID(1))},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1},
			[]model.Model{},
		},
		{
			"SELECT * FROM namespaces WHERE (root_id = (SELECT root_id FROM namespaces WHERE (id = $1)) AND id = (SELECT root_id FROM namespaces WHERE (id = $2)))",
			[]query.Option{
				query.WhereQuery("root_id", "=", SelectRootID(1)),
				query.WhereQuery("id", "=", SelectRootID(1)),
			},
			sqlmock.NewRows(namespaceCols),
			[]driver.Value{1, 1},
			[]model.Model{},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatal(errors.Cause(err))
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

	mock.ExpectPrepare(
		fmt.Sprintf(insertFmt, table),
	).ExpectQuery().WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	mock.ExpectPrepare(
		fmt.Sprintf(updateFmt, table),
	).ExpectExec().WillReturnResult(sqlmock.NewResult(1, 1))

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

	mock.ExpectPrepare(
		fmt.Sprintf(insertFmt, table),
	).ExpectQuery().WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	mock.ExpectPrepare(
		fmt.Sprintf(updateFmt, table),
	).ExpectExec().WillReturnResult(sqlmock.NewResult(1, 1))

	if _, err := store.GetByPath("blackmesa@freemang"); err != nil {
		t.Fatal(errors.Cause(err))
	}

	mock.ExpectQuery(
		regexp.QuoteMeta("SELECT * FROM namespaces WHERE (path = $1)"),
	).WithArgs(
		[]driver.Value{"blackmesa/blueshift"}...,
	).WillReturnRows(sqlmock.NewRows(namespaceCols))

	mock.ExpectPrepare(
		fmt.Sprintf(insertFmt, table),
	).ExpectQuery().WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	mock.ExpectPrepare(
		fmt.Sprintf(updateFmt, table),
	).ExpectExec().WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectPrepare(
		fmt.Sprintf(insertFmt, table),
	).ExpectQuery().WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	mock.ExpectPrepare(
		fmt.Sprintf(updateFmt, table),
	).ExpectExec().WillReturnResult(sqlmock.NewResult(1, 1))

	if _, err := store.GetByPath("blackmesa/blueshift"); err != nil {
		t.Fatal(errors.Cause(err))
	}
}

func Test_StoreCreate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	n := &Namespace{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, table)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(n); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if n.ID != id {
		t.Fatalf("namespace id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, n.ID)
	}
}

func Test_StoreUpdate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	n := &Namespace{ID: 10}

	expected := fmt.Sprintf(updateFmt, table)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(n.ID, 1))

	if err := store.Update(n); err != nil {
		t.Fatal(errors.Cause(err))
	}
}

func Test_StoreDelete(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	nn := []*Namespace{
		&Namespace{ID: 1},
		&Namespace{ID: 2},
		&Namespace{ID: 3},
	}

	mock.ExpectPrepare(
		regexp.QuoteMeta("DELETE FROM namespaces WHERE (root_id IN ($1, $2, $3))"),
	).ExpectExec().WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(nn...); err != nil {
		t.Fatal(errors.Cause(err))
	}
}

func Test_NamespaceCascadeVisiblity(t *testing.T) {
	n := &Namespace{
		ID:         1,
		RootID:     sql.NullInt64{Int64: 1, Valid: true},
		Visibility: Private,
	}

	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectPrepare(
		regexp.QuoteMeta("UPDATE namespaces SET visibility = $1 WHERE (root_id = $2)"),
	).ExpectExec().WillReturnResult(sqlmock.NewResult(n.ID, 1))

	if err := n.CascadeVisibility(sqlx.NewDb(db, "sqlmock")); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
