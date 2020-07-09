package repo

import "github.com/andrewpillar/thrall/form"

// Form is the type that represents the input data for enabling/disabling a
// repository on a Git hosting provider.
type Form struct {
	RepoID   int64  `schema:"repo_id"`
	Name     string `schema:"name"`
	Provider string `schema:"provider"`
}

var _ form.Form = (*Form)(nil)

// Fields implements the form.Form interface. This will always return an empty
// map.
func (f Form) Fields() map[string]string { return map[string]string{} }

// Validate implements the form.Form interface. This will always return nil as
// the Form cannot accept user inputs, it is something that is pre-populated.
func (f Form) Validate() error { return nil }
