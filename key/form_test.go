package key

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql/driver"
	"encoding/pem"
	"regexp"
	"testing"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/user"

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

	namespaceStore.User = &user.User{ID: 10}

	tests := []struct {
		form        Form
		errs        []string
		shouldError bool
	}{
		{
			Form{Keys: keyStore},
			[]string{"name", "key"},
			true,
		},
		{
			Form{
				Keys:       keyStore,
				Name:       "private",
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
			Form{
				Resource: namespace.Resource{
					User:       &user.User{ID: 10},
					Namespaces: namespaceStore,
					Namespace:  "blackmesa",
				},
				Keys:       keyStore,
				Name:       "private",
				PrivateKey: string(spoofKey(t)),
			},
			[]string{},
			false,
		},
	}

	for i, test := range tests {
		uniqueQuery := "SELECT * FROM keys WHERE (name = $1 AND namespace_id IS NULL)"
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

		keyMock.ExpectQuery(
			regexp.QuoteMeta(uniqueQuery),
		).WithArgs(uniqueArgs...).WillReturnRows(sqlmock.NewRows(keyCols))

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				cause := errors.Cause(err)

				ferrs, ok := cause.(*webutil.Errors)

				if !ok {
					t.Errorf("test[%d] - expected error to be *webutil.Errors, it was '%v'\n", i, cause)
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
