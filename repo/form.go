package repo

import "github.com/andrewpillar/thrall/form"

type Form struct {
	RepoID   int64  `schema:"repo_id"`
	Name     string `schema:"name"`
	Provider string `schema:"provider"`
}

var _ form.Form = (*Form)(nil)

func (f Form) Fields() map[string]string {
	return map[string]string{}
}

func (f Form) Validate() error {
	return nil
}
