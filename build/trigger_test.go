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

var triggerCols = []string{
	"build_id",
	"type",
	"comment",
	"data",
}

func triggerStore(t *testing.T) (TriggerStore, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return NewTriggerStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func triggerDataEquals(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func Test_TriggerType(t *testing.T) {
	tests := []struct{
		val         []byte
		expected    triggerType
		shouldError bool
	}{
		{[]byte("manual"), Manual, false},
		{[]byte("push"), Push, false},
		{[]byte("pull"), Pull, false},
		{[]byte("foo"), triggerType(0), true},
	}

	for _, test := range tests {
		var typ triggerType

		if err := typ.Scan(test.val); err != nil {
			if test.shouldError {
				continue
			}
			t.Fatal(err)
		}

		if typ != test.expected {
			t.Errorf("mismatch triggerType\n\texpected = '%s'\n\t  actual = '%s'\n", test.expected, typ)
		}
	}
}

func Test_TriggerData(t *testing.T) {
	tests := []struct{
		val         []byte
		expected    triggerData
		shouldError bool
	}{
		{
			[]byte(`{"email":"email@example.com","comment":"some commit message"}`),
			triggerData(map[string]string{
				"email": "email@example.com", "comment": "some commit message",
			}),
			false,
		},
		{
			[]byte(`{"email:"email@example.com"}`),
			nil,
			true,
		},
	}

	for _, test := range tests {
		var data triggerData

		if err := data.Scan(test.val); err != nil {
			if test.shouldError {
				continue
			}
			t.Fatal(err)
		}

		if !triggerDataEquals(data, test.expected) {
			t.Errorf("mismatch triggerData\n\texpected = '%s'\n\t  actual = '%s'\n", test.expected, data)
		}
	}
}

func Test_TriggerCommentTitleBody(t *testing.T) {
	tests := []struct{
		trig          Trigger
		expectedTitle string
		expectedBody  string
	}{
		{
			Trigger{Comment: "Title                                                                        "},
			"Title",
			"",
		},
		{
			Trigger{Comment: `Another comment title
The body`},
			"Another comment title...",
			"...The body",
		},
		{
			Trigger{Comment: `A super long title that should be longer than 72 characters, this should wrap round to the comment body too.`},
			"A super long title that should be longer than 72 characters, this should...",
			"...wrap round to the comment body too.",
		},
		{
			Trigger{Comment: "short comment"},
			"short comment",
			"",
		},
	}

	for _, test := range tests {
		if title := test.trig.CommentTitle(); title != test.expectedTitle {
			t.Errorf("mismatch trigger title\n\texpected = '%s'\n\t  actual = '%s'\n", test.expectedTitle, title)
		}

		if body := test.trig.CommentBody(); body != test.expectedBody {
			t.Errorf("mismatch trigger body\n\texpected = '%s'\n\t  actual = '%s'\n", test.expectedBody, body)
		}
	}
}

func Test_TriggerStoreAll(t *testing.T) {
	store, mock, close_ := triggerStore(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM build_triggers",
			[]query.Option{},
			sqlmock.NewRows(triggerCols),
			[]driver.Value{},
			[]model.Model{},
		},
		{
			"SELECT * FROM build_triggers WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(triggerCols),
			[]driver.Value{10},
			[]model.Model{&Build{ID: 10}},
		},
	}

	for _, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.All(test.opts...); err != nil {
			t.Fatal(errors.Cause(err))
		}

		store.Build = nil
	}
}

func Test_TriggerStoreCreate(t *testing.T) {
	store, mock, close_ := triggerStore(t)
	defer close_()

	tr := &Trigger{}

	id := int64(10)
	expected := fmt.Sprintf(insertFmt, triggerTable)

	rows := mock.NewRows([]string{"id"}).AddRow(id)

	mock.ExpectPrepare(expected).ExpectQuery().WillReturnRows(rows)

	if err := store.Create(tr); err != nil {
		t.Fatal(errors.Cause(err))
	}

	if tr.ID != id {
		t.Fatalf("trigger id mismatch\n\texpected = '%d'\n\tactual   = '%d'\n", id, tr.ID)
	}
}