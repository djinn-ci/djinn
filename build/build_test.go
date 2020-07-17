package build

import (
	"database/sql/driver"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/runner"
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
			"SELECT * FROM builds",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM builds WHERE (id IN (SELECT build_id FROM build_tags WHERE (name LIKE $1)))",
			[]query.Option{WhereSearch("borealis")},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"%borealis%"},
			[]database.Model{},
		},
		{
			"SELECT * FROM builds WHERE (status IN ($1, $2))",
			[]query.Option{WhereStatus("passed")},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"passed", "passed_with_failures"},
			[]database.Model{},
		},
		{
			"SELECT * FROM builds WHERE (status IN ($1))",
			[]query.Option{WhereStatus("queued")},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"queued"},
			[]database.Model{},
		},
		{
			"SELECT * FROM builds WHERE (id IN (SELECT build_id FROM build_tags WHERE (name = $1)))",
			[]query.Option{WhereTag("borealis")},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"borealis"},
			[]database.Model{},
		},
		{
			"SELECT * FROM builds WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{1},
			[]database.Model{&namespace.Namespace{ID: 1}},
		},
		{
			"SELECT * FROM builds WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{1, 1, 1},
			[]database.Model{&user.User{ID: 1}},
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

func Test_StoreIndex(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM builds WHERE (status IN ($1))",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"running"},
			[]database.Model{},
		},
		{
			"SELECT * FROM builds WHERE (id IN (SELECT build_id FROM build_tags WHERE (name LIKE $1))",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"%ravenholm%"},
			[]database.Model{},
		},
		{
			"SELECT * FROM builds WHERE (id IN (SELECT build_id FROM build_tags WHERE (name = $1))",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{"ravenholm"},
			[]database.Model{},
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

func Test_StoreGet(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []testQuery{
		{
			"SELECT * FROM builds",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{},
			[]database.Model{},
		},
		{
			"SELECT * FROM builds WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{1, 1, 1},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT * FROM builds WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows(buildCols),
			[]driver.Value{1},
			[]database.Model{&namespace.Namespace{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WithArgs(test.args...).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Get(test.opts...); err != nil {
			t.Errorf("tests[%d] - %s\n", i, errors.Cause(err))
		}

		store.User = nil
		store.Namespace = nil
	}
}

func Test_StoreCreate(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	tests := []struct {
		models   []database.Model
		manifest config.Manifest
		trigger  *Trigger
		tags     []string
		queries  []string
		rows     []*sqlmock.Rows
	}{
		{
			[]database.Model{},
			config.Manifest{Driver: map[string]string{"type": "qemu", "image": "centos/7"}},
			&Trigger{
				Type: Manual,
				Data: map[string]string{
					"email":    "me@example.com",
					"username": "me",
				},
			},
			[]string{"centos/7"},
			[]string{
				"^INSERT INTO builds (.+) VALUES (.+)$",
				"^INSERT INTO build_triggers (.+) VALUES (.+)$",
				"^INSERT INTO build_tags (.+) VALUES (.+)$",
			},
			[]*sqlmock.Rows{
				sqlmock.NewRows([]string{"id"}).AddRow(10),
				sqlmock.NewRows([]string{"id"}).AddRow(10),
				sqlmock.NewRows([]string{"id"}).AddRow(10),
			},
		},
		{
			[]database.Model{
				&user.User{ID: 27},
			},
			config.Manifest{
				Namespace: "example",
				Driver:    map[string]string{"type": "qemu", "image": "centos/7"},
			},
			&Trigger{
				Type: Manual,
				Data: map[string]string{
					"email":    "me@example.com",
					"username": "me",
				},
			},
			[]string{"centos/7"},
			[]string{
				"^SELECT \\* FROM namespaces WHERE \\(user_id = \\$1 OR root_id IN \\(SELECT namespace_id FROM namespace_collaborators WHERE \\(user_id = \\$2\\)\\)\\) AND \\(path = \\$3\\)$",
				"^SELECT \\* FROM namespace_collaborators WHERE \\(namespace_id = \\$1\\)$",
				"^INSERT INTO builds (.+) VALUES (.+)$",
				"^INSERT INTO build_triggers (.+) VALUES (.+)$",
				"^INSERT INTO build_tags (.+) VALUES (.+)$",
			},
			[]*sqlmock.Rows{
				sqlmock.NewRows([]string{"id", "user_id"}).AddRow(13, 27),
				sqlmock.NewRows([]string{"id"}).AddRow(13),
				sqlmock.NewRows([]string{"id"}).AddRow(10),
				sqlmock.NewRows([]string{"id"}).AddRow(10),
				sqlmock.NewRows([]string{"id"}).AddRow(10),
			},
		},
	}

	for i, test := range tests {
		store.Bind(test.models...)

		for j, q := range test.queries {
			mock.ExpectQuery(q).WillReturnRows(test.rows[j])
		}

		if _, err := store.Create(test.manifest, test.trigger, test.tags...); err != nil {
			t.Errorf("tests[%d] - unexpected Create error: %s\n", i, errors.Cause(err))
		}
	}
}

func Test_StoreStarted(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectExec("^UPDATE builds SET status = \\$1, started_at = \\$2 WHERE (id = \\$3)$")

	store.Started(1)
}

func Test_StoreFinished(t *testing.T) {
	store, mock, close_ := store(t)
	defer close_()

	mock.ExpectExec("^UPDATE builds SET status = \\$1, output = \\$2, finished_at = \\$3 WHERE (id = \\$4)$")

	store.Finished(1, "done", runner.Passed)
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
			[]database.Model{},
		},
		{
			"SELECT COUNT(*) FROM builds WHERE (namespace_id IN (SELECT id FROM namespaces WHERE (root_id IN (SELECT namespace_id FROM namespace_collaborators WHERE (user_id = $1) UNION SELECT id FROM namespaces WHERE (user_id = $2)))) OR user_id = $3)",
			[]query.Option{},
			sqlmock.NewRows([]string{"*"}).AddRow(1),
			[]driver.Value{},
			[]database.Model{&user.User{ID: 1}},
		},
		{
			"SELECT COUNT(*) FROM builds WHERE (namespace_id = $1)",
			[]query.Option{},
			sqlmock.NewRows([]string{"*"}).AddRow(1),
			[]driver.Value{},
			[]database.Model{&namespace.Namespace{ID: 1}},
		},
	}

	for i, test := range tests {
		mock.ExpectQuery(regexp.QuoteMeta(test.query)).WillReturnRows(test.rows)

		store.Bind(test.models...)

		if _, err := store.Paginate(1); err != nil {
			t.Errorf("tests[%d] - %s\n", i, errors.Cause(err))
		}

		store.User = nil
		store.Namespace = nil
	}
}
