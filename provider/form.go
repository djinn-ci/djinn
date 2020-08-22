package provider

import "github.com/andrewpillar/thrall/form"

// Form is the type that represents the input data for enabling/disabling a
// repository on a Git hosting provider.
type RepoForm struct {
	RepoID   int64  `schema:"repo_id"`
	Name     string `schema:"name"`
	Provider string `schema:"provider"`
}

var _ form.Form = (*RepoForm)(nil)

// Fields implements the form.Form interface. This will always return an empty
// map.
func (f RepoForm) Fields() map[string]string { return map[string]string{} }

// Validate implements the form.Form interface. This will always return nil as
// the Form cannot accept user inputs, it is something that is pre-populated.
func (f RepoForm) Validate() error { return nil }
