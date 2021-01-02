package provider

import "github.com/andrewpillar/webutil"

// Form is the type that represents the input data for enabling/disabling a
// repository on a Git hosting provider.
type RepoForm struct {
	ProviderID int64  `schema:"provider_id"`
	RepoID     int64  `schema:"repo_id"`
	Name       string `schema:"name"`
	Href       string `schema:"href"`
}

var _ webutil.Form = (*RepoForm)(nil)

// Fields implements the form.Form interface. This will always return an empty
// map.
func (f RepoForm) Fields() map[string]string { return map[string]string{} }

// Validate implements the form.Form interface. This will always return nil as
// the Form cannot accept user inputs, it is something that is pre-populated.
func (f RepoForm) Validate() error { return nil }
