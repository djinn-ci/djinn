package user

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"

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
	userCols = []string{
		"id",
		"email",
		"username",
		"password",
		"created_at",
		"updated_at",
		"deleted_at",
	}

	bcryptPassword = []byte{36, 50, 97, 36, 49, 48, 36, 54, 82, 100, 70, 83, 47, 83, 102, 67, 87, 99, 50, 106, 102, 121, 72, 66, 51, 97, 100, 47, 117, 101, 98, 84, 119, 115, 82, 47, 65, 97, 103, 50, 88, 85, 86, 121, 76, 84, 69, 76, 82, 48, 69, 47, 53, 90, 99, 111, 113, 109, 65, 101}
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
			"SELECT * FROM users",
			[]query.Option{},
			sqlmock.NewRows(userCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM users WHERE (username = $1)",
			[]query.Option{query.Where("username", "=", query.Arg("freemang"))},
			sqlmock.NewRows(userCols),
			[]driver.Value{"freemang"},
			[]database.Model{},
		},
		{
			"SELECT * FROM users WHERE (email = $1 OR username = $2)",
			[]query.Option{WhereHandle("freemang@black-mesa.com")},
			sqlmock.NewRows(userCols),
			[]driver.Value{"freemang@black-mesa.com", "freemang@black-mesa.com"},
			[]database.Model{},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}
	}
}

func Test_StoreGet(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM users",
			[]query.Option{},
			sqlmock.NewRows(userCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM users WHERE (username = $1)",
			[]query.Option{query.Where("username", "=", query.Arg("freemang"))},
			sqlmock.NewRows(userCols),
			[]driver.Value{"freemang"},
			[]database.Model{},
		},
		{
			"SELECT * FROM users WHERE (email = $1 OR username = $2)",
			[]query.Option{WhereHandle("freemang@black-mesa.com")},
			sqlmock.NewRows(userCols),
			[]driver.Value{"freemang@black-mesa.com", "freemang@black-mesa.com"},
			[]database.Model{},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		if _, err := store.Get(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}
	}
}

func Test_StoreAuth(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []struct {
		user       *User
		password   string
		shouldAuth bool
	}{
		{
			&User{
				Email:    "freemang@black-mesa.com",
				Username: "freemang",
				Password: bcryptPassword,
			},
			"secret",
			true,
		},
		{
			&User{
				Email:    "freemang@black-mesa.com",
				Username: "freemang",
				Password: bcryptPassword,
			},
			"foo",
			false,
		},
	}

	for i, test := range tests {
		rows := mock.NewRows(
			[]string{"email", "username", "password"},
		).AddRow(test.user.Email, test.user.Username, test.user.Password)

		mock.ExpectQuery(
			regexp.QuoteMeta("SELECT * FROM users WHERE (email = $1 OR username = $2)"),
		).WithArgs(test.user.Username, test.user.Username).WillReturnRows(rows)

		if _, err := store.Auth(test.user.Username, test.password); err != nil {
			if !test.shouldAuth {
				continue
			}
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}
	}
}

func Test_StoreCreate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectQuery(
		"^INSERT INTO users \\((.+)\\) VALUES \\((.+)\\) RETURNING id$",
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(13))

	mock.ExpectQuery(
		"^SELECT COUNT\\(\\*\\) FROM account_tokens WHERE \\(user_id = \\$1 AND purpose = \\$2\\)$",
	).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectExec(
		"^INSERT INTO account_tokens \\((.+)\\) VALUES \\((.+)\\)$",
	).WillReturnResult(sqlmock.NewResult(0, 1))

	if _, _, err := store.Create("me@example.com", "me", []byte("secret")); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}

func Test_StoreUpdate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectExec(
		"^UPDATE users SET email = \\$1, cleanup = \\$2, updated_at = \\$3, password = \\$4 WHERE \\(id = \\$5\\)$",
	).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.Update(13, "me@elpmaxe.com", true, []byte("secret")); err != nil {
		t.Errorf("unexpected Update error: %s\n", errors.Cause(err))
	}
}
