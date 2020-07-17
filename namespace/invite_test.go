package namespace

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var inviteCols = []string{
	"namespace_id",
	"invitee_id",
	"inviter_id",
}

func inviteStore(t *testing.T) (*InviteStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewInviteStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_InviteStoreAll(t *testing.T) {
	store, mock, close_ := inviteStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM namespace_invites",
			[]query.Option{},
			sqlmock.NewRows(inviteCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM namespace_invites WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(inviteCols),
			[]driver.Value{1},
			[]database.Model{&Namespace{ID: 1}},
		},
		{
			"SELECT * FROM namespace_invites WHERE (invitee_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(inviteCols),
			[]driver.Value{1},
			[]database.Model{&user.User{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.Namespace = nil
	}
}

func Test_InviteStoreGet(t *testing.T) {
	store, mock, close_ := inviteStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM namespace_invites",
			[]query.Option{},
			sqlmock.NewRows(inviteCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM namespace_invites WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(inviteCols),
			[]driver.Value{1},
			[]database.Model{&Namespace{ID: 1}},
		},
		{
			"SELECT * FROM namespace_invites WHERE (invitee_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(inviteCols),
			[]driver.Value{1},
			[]database.Model{&user.User{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.Namespace = nil
	}
}
