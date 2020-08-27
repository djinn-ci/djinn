package build

import (
	"database/sql/driver"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"

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

func triggerStore(t *testing.T) (*TriggerStore, sqlmock.Sqlmock, func() error) {
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
	tests := []struct {
		val         []byte
		expected    TriggerType
		shouldError bool
	}{
		{[]byte("manual"), Manual, false},
		{[]byte("push"), Push, false},
		{[]byte("pull"), Pull, false},
		{[]byte("foo"), TriggerType(0), true},
	}

	for i, test := range tests {
		var typ TriggerType

		if err := typ.Scan(test.val); err != nil {
			if test.shouldError {
				continue
			}
			t.Fatalf("test[%d] - %s\n", i, err)
		}

		if typ != test.expected {
			t.Errorf("test[%d] - expected type = '%s' actual type = '%s'\n", i, test.expected, typ)
		}
	}
}

func Test_TriggerData(t *testing.T) {
	tests := []struct {
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

	for i, test := range tests {
		var data triggerData

		if err := data.Scan(test.val); err != nil {
			if test.shouldError {
				continue
			}
			t.Fatal(err)
		}

		if !triggerDataEquals(data, test.expected) {
			t.Errorf("test[%d] - expected data '%s' actual data = '%s'\n", i, test.expected, data)
		}
	}
}

func Test_TriggerCommentTitleBody(t *testing.T) {
	tests := []struct {
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

	for i, test := range tests {
		if title := test.trig.CommentTitle(); title != test.expectedTitle {
			t.Errorf("test[%d] - expected title = '%s' actual title = '%s'\n", i, test.expectedTitle, title)
		}

		if body := test.trig.CommentBody(); body != test.expectedBody {
			t.Errorf("test[%d] - expected body = '%s' actual body = '%s'\n", i, test.expectedBody, body)
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
			[]database.Model{},
		},
		{
			"SELECT * FROM build_triggers WHERE (build_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(triggerCols),
			[]driver.Value{10},
			[]database.Model{&Build{ID: 10}},
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

func Test_TriggerStoreCreate(t *testing.T) {
	store, mock, close_ := triggerStore(t)
	defer close_()

	tr := &Trigger{
		BuildID: 1,
		Type:    Push,
		Comment: "some commit message",
		Data: map[string]string{
			"email":    "me@example.com",
			"username": "me",
		},
	}

	mock.ExpectQuery(
		"^INSERT INTO build_triggers \\((.+)\\) VALUES \\(\\$1, \\$2, \\$3, \\$4, \\$5\\) RETURNING id$",
	).WillReturnRows(mock.NewRows([]string{"id"}).AddRow(10))

	if err := store.Create(tr); err != nil {
		t.Errorf("unexpected Create error: %s\n", errors.Cause(err))
	}
}
