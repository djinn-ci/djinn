package namespace

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var collaboratorCols = []string{
	"namespace_id",
	"user_id",
}

func collaboratorStore(t *testing.T) (*CollaboratorStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewCollaboratorStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_CollaboratorStoreAll(t *testing.T) {
	store, mock, close_ := collaboratorStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM namespace_collaborators",
			[]query.Option{},
			sqlmock.NewRows(collaboratorCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM namespace_collaborators WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(collaboratorCols),
			[]driver.Value{1},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM namespace_collaborators WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(collaboratorCols),
			[]driver.Value{1},
			[]database.Model{&Namespace{ID: 1, RootID: sql.NullInt64{Int64: 1, Valid: true}}},
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

func Test_CollaboratorStoreGet(t *testing.T) {
	store, mock, close_ := collaboratorStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM namespace_collaborators",
			[]query.Option{},
			sqlmock.NewRows(collaboratorCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM namespace_collaborators WHERE (user_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(collaboratorCols),
			[]driver.Value{1},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM namespace_collaborators WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(collaboratorCols),
			[]driver.Value{1},
			[]database.Model{&Namespace{ID: 1, RootID: sql.NullInt64{Int64: 1, Valid: true}}},
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

func Test_CollaboratorStoreCreate(t *testing.T) {
	store, mock, close_ := collaboratorStore(t)
	defer close_()

	c := &Collaborator{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, collaboratorTable)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectQuery(expected).WillReturnRows(rows)

	if err := store.Create(c); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if c.ID != id {
		t.Fatalf("expected = '%d' actual = '%d'\n", id, c.ID)
	}
}

func Test_CollaboratorStoreUpdate(t *testing.T) {
	store, mock, close_ := collaboratorStore(t)
	defer close_()

	cc := []*Collaborator{
		&Collaborator{ID: 1},
		&Collaborator{ID: 2},
		&Collaborator{ID: 3},
	}

	expected := fmt.Sprintf(deleteFmt, collaboratorTable)

	mock.ExpectExec(expected).WillReturnResult(sqlmock.NewResult(0, 3))

	if err := store.Delete(cc...); err != nil {
		t.Fatal(errors.Cause(err))
	}
}
