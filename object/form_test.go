package object

import (
	"database/sql/driver"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

var (
	width  = 200
	height = 100
)

func spoofFile(t *testing.T) (*http.Request, http.ResponseWriter) {
	img := image.NewRGBA(image.Rectangle{
		image.Point{0, 0},
		image.Point{width, height},
	})

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < width; i++ {
		for j := 0; j < height; j++ {
			r := uint8(rand.Intn(255))
			g := uint8(rand.Intn(255))
			b := uint8(rand.Intn(255))

			img.Set(i, j, color.RGBA{r, g, b, 0xFF})
		}
	}

	pr, pw := io.Pipe()

	mw := multipart.NewWriter(pw)

	go func() {
		defer mw.Close()

		w, err := mw.CreateFormFile("file", "rand.png")

		if err != nil {
			t.Fatal(err)
		}

		if err := png.Encode(w, img); err != nil {
			t.Fatal(err)
		}
	}()

	r := httptest.NewRequest("POST", "/", pr)
	r.Header.Add("Content-Type", mw.FormDataContentType())

	return r, httptest.NewRecorder()
}

func namespaceStore(t *testing.T) (*namespace.Store, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return namespace.NewStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_FormValidate(t *testing.T) {
	objectStore, objectMock, objectClose := store(t)
	defer objectClose()

	namespaceStore, namespaceMock, namespaceClose := namespaceStore(t)
	defer namespaceClose()

	tests := []struct {
		form        Form
		errs        []string
		shouldError bool
	}{
		{
			Form{
				Objects: objectStore,
				Name:    "image.png",
			},
			[]string{},
			false,
		},
		{
			Form{Objects: objectStore},
			[]string{},
			true,
		},
		{
			Form{
				Resource: namespace.Resource{
					User:       &user.User{ID: 10},
					Namespaces: namespaceStore,
					Namespace:  "aperture",
				},
				Objects: objectStore,
				Name:    "image.png",
			},
			[]string{},
			false,
		},
		{
			Form{
				Resource: namespace.Resource{
					User:       &user.User{ID: 10},
					Namespaces: namespaceStore,
					Namespace:  "aperture",
				},
				Objects: objectStore,
			},
			[]string{"name"},
			true,
		},
	}

	for i, test := range tests {
		uniqueQuery := "SELECT * FROM objects WHERE (name = $1)"
		uniqueArgs := []driver.Value{test.form.Name}

		if test.form.Namespace != "" {
			var (
				collabId    int64 = 13
				namespaceId int64 = 1
				userId      int64 = 10
			)

			uniqueQuery = "SELECT * FROM objects WHERE (namespace_id = $1 AND name = $2)"
			uniqueArgs = []driver.Value{namespaceId, test.form.Name}

			namespaceMock.ExpectQuery(
				regexp.QuoteMeta("SELECT * FROM namespaces WHERE (path = $1)"),
			).WithArgs(test.form.Namespace).WillReturnRows(
				sqlmock.NewRows([]string{"id", "root_id"}).AddRow(namespaceId, namespaceId),
			)

			namespaceMock.ExpectQuery(
				regexp.QuoteMeta("SELECT * FROM namespace_collaborators WHERE (namespace_id = $1)"),
			).WithArgs(namespaceId).WillReturnRows(
				sqlmock.NewRows([]string{"id", "user_id", "namespace_id"}).AddRow(collabId, userId, namespaceId),
			)
		}

		objectMock.ExpectQuery(
			regexp.QuoteMeta(uniqueQuery),
		).WithArgs(uniqueArgs...).WillReturnRows(sqlmock.NewRows(objectCols))

		r, w := spoofFile(t)

		test.form.File = form.File{
			Writer:  w,
			Request: r,
		}

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				cause := errors.Cause(err)

				ferrs, ok := cause.(form.Errors)

				if !ok {
					t.Fatalf("test[%d] - expected error to be form.Errors, is was '%s'\n", i, cause)
				}

				for _, err := range test.errs {
					if _, ok := ferrs[err]; !ok {
						t.Errorf("test[%d] - expected field '%s' to be in form.Errors\n", i, err)
					}
				}
				continue
			}
			t.Fatalf("test[%d] - %s\n", i, errors.Cause(err))
		}
	}
}
