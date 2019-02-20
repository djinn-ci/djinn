package form

import (
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
)

var (
	ErrNamespaceNameRequired = errors.New("Name can't be blank")
	ErrNamespaceExists       = errors.New("Namespace already exists")
)

type CreateNamespace struct {
	UserID      int64
	Name        string `schema:"name"`
	Description string `schema:"description"`
	Private     bool   `schema:"private"`
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

	count, err := model.Count(model.NamespacesTable, map[string]interface{}{"user_id": f.UserID, "name": f.Name})

	if err != nil {
		log.Error.Println(errors.Err(err))

		errs.Put("namespace", errors.Cause(err))
	} else if count > 0 {
		errs.Put("name", ErrNamespaceExists)
	}

	return errs.Final()
}
