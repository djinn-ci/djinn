package build

import (
	"database/sql/driver"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var buildCols = []string{
	"id",
	"user_id",
	"namespace_id",
	"manifest",
	"status",
	"output",
	"secret",
	"created_at",
	"started_at",
	"finished_at",
}

func store(t *testing.T) (Store, sqlmock.Sqlmock, func() error) {
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
			"SELECT * FROM builds",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM builds WHERE (id IN (SELECT build_id FROM build_tags WHERE (name LIKE $1)))",
			[]query.Option{WhereSearch("borealis")},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"%borealis%"},
			[]model.Model{},
		},
		{
			"SELECT * FROM builds WHERE (status IN ($1, $2))",
			[]query.Option{WhereStatus("passed")},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"passed", "passed_with_failures"},
			[]model.Model{},
		},
		{
			"SELECT * FROM builds WHERE (status IN ($1))",
			[]query.Option{WhereStatus("queued")},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"queued"},
			[]model.Model{},
		},
		{
			"SELECT * FROM builds WHERE (id IN (SELECT build_id FROM build_tags WHERE (name = $1)))",
			[]query.Option{WhereTag("borealis")},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"borealis"},
			[]model.Model{},
		},
		{
			"SELECT * FROM builds WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{1},
			[]model.Model{&namespace.Namespace{ID: 1}},
		},
		{
			"SELECT * FROM builds WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{1, 1, 1},
			[]model.Model{&user.User{ID: 1}},
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

func Test_StoreIndex(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM builds WHERE (status IN ($1))",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"running"},
			[]model.Model{},
		},
		{
			"SELECT * FROM builds WHERE (id IN (SELECT build_id FROM build_tags WHERE (name LIKE $1))",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"%ravenholm%"},
			[]model.Model{},
		},
		{
			"SELECT * FROM builds WHERE (id IN (SELECT build_id FROM build_tags WHERE (name = $1))",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"ravenholm"},
			[]model.Model{},
		},
	}

	vals := []url.Values{
		url.Values(map[string][]string{"status": []string{"running"}}),
		url.Values(map[string][]string{"search": []string{"ravenholm"}}),
		url.Values(map[string][]string{"tag": []string{"ravenholm"}}),
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

func Test_StoreGet(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM builds",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM builds WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{1, 1, 1},
			[]model.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM builds WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{1},
			[]model.Model{&namespace.Namespace{ID: 1}},
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

func Test_StoreCreate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	b := &Build{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, table)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(b); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if b.ID != id {
		t.Fatalf("build id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, b.ID)
	}
}

func Test_StoreUpdate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	b := &Build{ID: 10}

	expected := fmt.Sprintf(updateFmt, table)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(b.ID, 1))

	if err := store.Update(b); err != nil {
		t.Fatal(errors.Cause(err))
	}
}

func Test_StorePaginate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT COUNT(*) FROM builds",
			[]query.Option{},
			sqlmock.NewRows([]string{"*"}).AddRow(1),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT COUNT(*) FROM builds WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows([]string{"*"}).AddRow(1),
			[]driver.Value{},
			[]model.Model{&user.User{ID: 1}},
		},
		{
			"SELECT COUNT(*) FROM builds WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows([]string{"*"}).AddRow(1),
			[]driver.Value{},
			[]model.Model{&namespace.Namespace{ID: 1}},
		},
	}

	for _, test := range tests {
		mock.ExpectPrepare(regexp.QuoteMeta(test.query)).ExpectQuery().WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Paginate(1); err != nil {
			t.Fatal(errors.Cause(err))
		}

		store.User = nil
		store.Namespace = nil
	}
}
