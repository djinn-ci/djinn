package form

import (
	"regexp"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/model"
)

var (
	namespacePattern = "^[a-zA-Z0-9]+$"
	namespaceRegex   = regexp.MustCompile(namespacePattern)

	ErrNamespaceNameRequired = errors.New("Name can't be blank")
	ErrNamespaceNameLen      = errors.New("Name must be between 3 and 32 characters")
	ErrNamespaceInvalid      = errors.New("Name can only contain numbers and letters")
	ErrNamespaceExists       = errors.New("Namespace already exists")
	ErrNamespaceTooDeep      = errors.New("Namespaces can only be nested to 20 levels")
)

type Namespace struct {
	UserID      int64
	Parent      string           `schema:"parent"`
	Name        string           `schema:"name"`
	Description string           `schema:"description"`
	Visibility  model.Visibility `schema:"visibility"`
	Namespace   *model.Namespace
}

func (f Namespace) Get(key string) string {
	if key == "name" {
		return f.Name
	}

	if key == "description" {
		return f.Description
	}

	return ""
}

func (f Namespace) Validate() error {
	errs := NewErrors()

	if f.Name == "" {
		errs.Put("name", ErrNamespaceNameRequired)
	}

	if len(f.Name) < 3 || len(f.Name) > 64 {
		errs.Put("name", ErrNamespaceNameLen)
	}

	if !namespaceRegex.Match([]byte(f.Name)) {
		errs.Put("name", ErrNamespaceInvalid)
	}

	shouldCount := true

	if f.Namespace != nil && !f.Namespace.IsZero() {
		parts := strings.Split(f.Namespace.FullName, "/")
		parts[len(parts) - 1] = f.Name

		if f.Namespace.Name == f.Name {
			shouldCount = false
		}

		f.Name = strings.Join(parts, "/")
	} else if f.Parent != "" {
		f.Name = strings.Join([]string{f.Parent, f.Name}, "/")
	}

	if shouldCount {
		cols := map[string]interface{}{
			"user_id":   f.UserID,
			"full_name": f.Name,
		}

		count, err := model.Count(model.NamespacesTable, cols)

		if err != nil {
			log.Error.Println(errors.Err(err))

			errs.Put("namespace", errors.Cause(err))
		} else if count > 0 {
			errs.Put("name", ErrNamespaceExists)
		}
	}

	return errs.Final()
}
