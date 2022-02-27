package http

import "github.com/andrewpillar/webutil"

type RepoForm struct {
	ProviderID int64 `schema:"provider_id"`
	RepoID     int64 `schema:"repo_id"`
	Name       string
	Href       string
}

var _ webutil.Form = (*RepoForm)(nil)

func (RepoForm) Fields() map[string]string { return nil }
