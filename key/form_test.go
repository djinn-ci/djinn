package key

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql/driver"
	"encoding/pem"
	"regexp"
	"testing"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/user"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/jmoiron/sqlx"
)

func namespaceStore(t *testing.T) (namespace.Store, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return namespace.NewStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func spoofKey(t *testing.T) []byte {
	key, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		t.Fatal(err)
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

func spoofInvalidKey() []byte {
	key := make([]byte, 1024)
	rand.Reader.Read(key)
	return key
}

func Test_FormValidate(t *testing.T) {
	keyStore, keyMock, keyClose := store(t)
	defer keyClose()

	namespaceStore, namespaceMock, namespaceClose := namespaceStore(t)
	defer namespaceClose()

	tests := []struct{
		form        Form
		errs        []string
		shouldError bool
	}{
		{
			Form{
				Resource: namespace.Resource{
					User:       &user.User{ID: 10},
					Namespaces: namespaceStore,
					Namespace:  "blackmesa",
				},
				Keys:       keyStore,
				Name:      "private",
				PrivateKey: string(spoofKey(t)),
			},
			[]string{},
			false,
		},
		{
			Form{
				Keys:       keyStore,
				Name:      "private",
				PrivateKey: string(spoofKey(t)),
			},
			[]string{},
			false,
		},
		{
			Form{
				Keys:       keyStore,
				PrivateKey: string(spoofKey(t)),
			},
			[]string{"name"},
			true,
		},
		{
			Form{
				Name:       "private",
				Keys:       keyStore,
				PrivateKey: string(spoofInvalidKey()),
			},
			[]string{"key"},
			true,
		},
		{
			Form{Keys: keyStore},
			[]string{"name", "key"},
			true,
		},
	}

	for _, test := range tests {
		uniqueQuery := "SELECT * FROM keys WHERE (name = $1)"
		uniqueArgs := []driver.Value{test.form.Name}

		if test.form.Namespace != "" {
			var (
				collabId    int64 = 13
				namespaceId int64 = 1
				userId      int64 = 10
			)

			uniqueQuery = "SELECT * FROM keys WHERE (namespace_id = $1 AND name = $2)"
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

		keyMock.ExpectQuery(
			regexp.QuoteMeta(uniqueQuery),
		).WithArgs(uniqueArgs...).WillReturnRows(sqlmock.NewRows(keyCols))

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				cause := errors.Cause(err)

				ferrs, ok := cause.(form.Errors)

				if !ok {
					t.Fatalf("expected error to be form.Errors, is was '%s'\n", cause)
				}

				for _, err := range test.errs {
					if _, ok := ferrs[err]; !ok {
						t.Errorf("expected '%s' to be in form.Errors\n", err)
					}
				}
				continue
			}
			t.Fatal(errors.Cause(err))
		}
	}
}
