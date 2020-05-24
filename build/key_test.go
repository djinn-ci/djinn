package build

import (
	"database/sql/driver"
	"fmt"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var keyCols = []string{
	"build_id",
	"key_id",
	"name",
	"config",
	"location",
}

func keyStore(t *testing.T) (*KeyStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewKeyStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_KeyStoreAll(t *testing.T) {
	store, mock, close_ := keyStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM build_keys",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM build_keys WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(keyCols),
			[]driver.Value{10},
			[]model.Model{&Build{ID: 10}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}

		store.Build = nil
	}
}

func Test_KeyStoreCreate(t *testing.T) {
	store, mock, close_ := keyStore(t)
	defer close_()

	k := &Key{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, keyTable)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(k); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if k.ID != id {
		t.Fatalf("key id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, k.ID)
	}
}
