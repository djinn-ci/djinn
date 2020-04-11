package object

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

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
			"SELECT * FROM objects",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM objects WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{1, 1, 1},
			[]model.Model{userModel},
		},
		{
			"SELECT * FROM objects WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{1},
			[]model.Model{namespaceModel},
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
			"SELECT * FROM objects WHERE (name LIKE $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{"%aperture%"},
			[]model.Model{},
		},
	}

	vals := []url.Values{
		url.Values(map[string][]string{"search": []string{"aperture"}}),
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
			"SELECT * FROM objects",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM objects WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{1, 1, 1},
			[]model.Model{userModel},
		},
		{
			"SELECT * FROM objects WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(objectCols),
			[]driver.Value{1},
			[]model.Model{namespaceModel},
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

	o := &Object{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, table)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(o); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if o.ID != id {
		t.Fatalf("object id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, o.ID)
	}
}

func Test_StoreUpdate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	o := &Object{ID: 10}

	expected := fmt.Sprintf(updateFmt, table)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(o.ID, 1))

	if err := store.Update(o); err != nil {
		t.Fatal(errors.Cause(err))
	}
}

func Test_StoreDelete(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	oo := []*Object{
		&Object{ID: 1},
		&Object{ID: 2},
		&Object{ID: 3},
	}

	expected := fmt.Sprintf(deleteFmt, table)

	mock.ExpectPrepare(expected).ExpectExec().WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(oo...); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
