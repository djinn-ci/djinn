package form

import (
	"regexp"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
)

var (
	namespacePattern = "^[-a-zA-Z0-9]+$"
	namespaceRegex   = regexp.MustCompile(namespacePattern)

	ErrNamespaceNameRequired = errors.New("Name can't be blank")
	ErrNamespaceInvalid      = errors.New("Name can only contain numbers, letters, and dashes")
	ErrNamespaceExists       = errors.New("Namespace already exists")
)

type CreateNamespace struct {
	UserID      int64
	Parent      string `schema:"parent"`
	Name        string `schema:"name"`
	Description string `schema:"description"`
	Visibility  string `schema:"visibility"`
}

func (f CreateNamespace) Get(key string) string {
	if key == "name" {
		return f.Name
	}

	if key == "description" {
		return f.Description
	}

	return ""
}

func (f CreateNamespace) Validate() error {
	errs := NewErrors()

	if f.Name == "" {
		errs.Put("name", ErrNamespaceNameRequired)
	}

	if !namespaceRegex.Match([]byte(f.Name)) {
		errs.Put("name", ErrNamespaceInvalid)
	}

	count, err := model.Count(model.NamespacesTable, map[string]interface{}{"user_id": f.UserID, "name": f.Name})

	if err != nil {
		log.Error.Println(errors.Err(err))

		errs.Put("namespace", errors.Cause(err))
	} else if count > 0 {
		errs.Put("name", ErrNamespaceExists)
	}

	return errs.Final()
}
