package image

import (
	"crypto/rand"
	"database/sql/driver"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/user"

	"github.com/andrewpillar/webutil"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

func namespaceStore(t *testing.T) (*namespace.Store, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return namespace.NewStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func spoofFile(t *testing.T, useMagic bool) (*http.Request, http.ResponseWriter) {
	pr, pw := io.Pipe()

	mw := multipart.NewWriter(pw)

	go func() {
		defer mw.Close()

		w, err := mw.CreateFormFile("file", "combine.qcow2")

		if err != nil {
			t.Fatal(err)
		}

		if useMagic {
			w.Write(qcow)
		}

		buf := make([]byte, 1024)
		rand.Reader.Read(buf)
		w.Write(buf)
	}()

	r := httptest.NewRequest("POST", "/", pr)
	r.Header.Add("Content-Type", mw.FormDataContentType())

	return r, httptest.NewRecorder()
}

func Test_FormValidate(t *testing.T) {
	imageStore, imageMock, imageClose := store(t)
	defer imageClose()

	namespaceStore, namespaceMock, namespaceClose := namespaceStore(t)
	defer namespaceClose()

	namespaceStore.User = &user.User{ID: 10}

	tests := []struct {
		form        Form
		magic       bool
		errs        []string
		shouldError bool
	}{
		{
			Form{Images: imageStore},
			false,
			[]string{"name", "file"},
			true,
		},
		{
			Form{
				Resource: namespace.Resource{
					Author:     &user.User{ID: 10},
					Namespaces: namespaceStore,
					Namespace:  "city17",
				},
				Images: imageStore,
				Name:   "combine",
			},
			true,
			[]string{},
			false,
		},
		{
			Form{
				Resource: namespace.Resource{
					Author:     &user.User{ID: 10},
					Namespaces: namespaceStore,
					Namespace:  "city17",
				},
				Images: imageStore,
			},
			true,
			[]string{"name"},
			true,
		},
		{
			Form{
				Resource: namespace.Resource{
					Author:     &user.User{ID: 10},
					Namespaces: namespaceStore,
					Namespace:  "city17",
				},
				Images: imageStore,
				Name:   "combine",
			},
			false,
			[]string{"file"},
			true,
		},
		{
			Form{
				Resource: namespace.Resource{
					Author:     &user.User{ID: 10},
					Namespaces: namespaceStore,
					Namespace:  "city17",
				},
				Images: imageStore,
			},
			false,
			[]string{"name", "file"},
			true,
		},
	}

	for i, test := range tests {
		uniqueQuery := "SELECT \\* FROM images WHERE \\(name = \\$1 AND namespace_id IS NULL\\)"
		uniqueArgs := []driver.Value{test.form.Name}

		if test.form.Namespace != "" {
			var (
				collabId    int64 = 13
				namespaceId int64 = 1
				userId      int64 = 10
			)

			uniqueQuery = "SELECT \\* FROM images WHERE \\((.+)\\)"
			uniqueArgs = []driver.Value{userId, userId, userId, namespaceId, test.form.Name}

			namespaceMock.ExpectQuery(
				regexp.QuoteMeta("SELECT * FROM namespaces WHERE (user_id = $1 AND path = $2)"),
			).WithArgs(userId, test.form.Namespace).WillReturnRows(
				sqlmock.NewRows([]string{"id", "root_id"}).AddRow(namespaceId, namespaceId),
			)

			namespaceMock.ExpectQuery(
				regexp.QuoteMeta("SELECT * FROM namespace_collaborators WHERE (namespace_id = $1)"),
			).WithArgs(namespaceId).WillReturnRows(
				sqlmock.NewRows([]string{"id", "user_id", "namespace_id"}).AddRow(collabId, userId, namespaceId),
			)
		}

		imageMock.ExpectQuery(uniqueQuery).WithArgs(uniqueArgs...).WillReturnRows(sqlmock.NewRows(imageCols))

		r, _ := spoofFile(t, test.magic)

		test.form.File = webutil.NewFile("file", 0, r)

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				cause := errors.Cause(err)

				ferrs, ok := cause.(*webutil.Errors)

				if !ok {
					t.Fatalf("test[%d] - %v\n", i, cause)
				}

				for _, err := range test.errs {
					if _, ok := (*ferrs)[err]; !ok {
						t.Errorf("test[%d] - expected '%s' to be in *webutil.Errors\n", i, err)
					}
				}
				continue
			}
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}
	}
}
