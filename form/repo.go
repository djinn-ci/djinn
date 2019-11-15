package form

type Repo struct {
	RepoID   int64  `schema:"repo_id"`
	Name     string `schema:"name"`
	Provider string `schema:"provider"`
}

func (f Repo) Fields() map[string]string {
	return map[string]string{}
}

func (r Repo) Validate() error {
	return nil
}
