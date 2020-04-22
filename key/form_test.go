package key

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
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
	namespaceCols = []string{
		"id",
		"user_id",
		"root_id",
		"parent_id",
		"name",
		"path",
		"description",
		"level",
		"visibility",
		"created_at",
	}

	collabCols = []string{
		"namespace_id",
		"user_id",
	}
)

func genKey(t *testing.T) []byte {
	key, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		t.Fatal(err)
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

func namespaceStore(t *testing.T) (namespace.Store, sqlmock.Sqlmock, func() error) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatal(err)
	}
	return namespace.NewStore(sqlx.NewDb(db, "sqlmock")), mock, db.Close
}

func Test_FormValidate(t *testing.T) {
	keyStore, keyMock, keyClose := store(t)
	defer keyClose()

	namespaceStore, namespaceMock, namespaceClose := namespaceStore(t)
	defer namespaceClose()

	tests := []struct{
		form        Form
		key         *Key
		namespace   *namespace.Namespace
		errs        []string
		shouldError bool
	}{
		{
			Form{
				ResourceForm: namespace.ResourceForm{
					User: &user.User{ID: 10},
				},
				Keys:       keyStore,
				Name:       "private",
				PrivateKey: string(genKey(t)),
			},
			&Key{},
			&namespace.Namespace{},
			[]string{},
			false,
		},
		{
			Form{
				ResourceForm: namespace.ResourceForm{
					Namespaces: namespaceStore,
					Namespace:  "blackmesa",
					User:       &user.User{ID: 10},
				},
				Keys:       keyStore,
				Name:       "private",
				PrivateKey: string(genKey(t)),
			},
			&Key{},
			&namespace.Namespace{},
			[]string{},
			false,
		},
		{
			Form{
				ResourceForm: namespace.ResourceForm{
					Namespaces: namespaceStore,
					Namespace:  "blackmesa",
					User:       &user.User{ID: 10},
				},
				Keys:       keyStore,
			},
			&Key{},
			&namespace.Namespace{},
			[]string{"name"},
			true,
		},
		{
			Form{
				ResourceForm: namespace.ResourceForm{
					Namespaces: namespaceStore,
					Namespace:  "blackmesa",
					User:       &user.User{ID: 10},
				},
				Keys:       keyStore,
			},
			&Key{},
			&namespace.Namespace{},
			[]string{"name", "key"},
			true,
		},
		{
			Form{
				ResourceForm: namespace.ResourceForm{
					Namespaces: namespaceStore,
					Namespace:  "blackmesa",
					User:       &user.User{ID: 10},
				},
				Keys:       keyStore,
				Name:       "private",
				PrivateKey: string(genKey(t)),
			},
			&Key{},
			&namespace.Namespace{},
			[]string{"name", "key"},
			true,
		},
	}

	for _, test := range tests {
		if test.form.Namespace != "" {
			namespaceMock.ExpectQuery(
				regexp.QuoteMeta("SELECT * FROM namespaces WHERE (path = $1)"),
			).WithArgs(test.form.Namespace).WillReturnRows(
				sqlmock.NewRows(namespaceCols).AddRow(1, 10, 1, 0, "blackmesa", "blackmesa", "", 1, namespace.Internal, time.Now()),
			)

			namespaceMock.ExpectQuery(
				regexp.QuoteMeta("SELECT * FROM namespace_collaborators WHERE (namespace_id = $1)"),
			).WithArgs(1).WillReturnRows(
				sqlmock.NewRows(collabCols).AddRow(1, test.form.User.ID),
			)
		}

		keyMock.ExpectQuery(
			regexp.QuoteMeta("SELECT * FROM keys WHERE (name = $1)"),
		).WithArgs(test.form.Name).WillReturnRows(sqlmock.NewRows(keyCols))

		if err := test.form.Validate(); err != nil {
			if test.shouldError {
				if len(test.errs) == 0 {
					continue
				}

				ferrs, ok := err.(form.Errors)

				if !ok {
					t.Fatalf("expected error to be form.Errors, it was not\n%s\n", errors.Cause(err))
				}

				for _, err := range test.errs {
					if _, ok := ferrs[err]; !ok {
						t.Fatalf("expected field '%s' to be in form.Errors, it was not\n", err)
					}
				}
				continue
			}
			t.Fatal(errors.Cause(err))
		}
	}
}
