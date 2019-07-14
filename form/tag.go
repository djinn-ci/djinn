package form

type Tag struct {
	Tags tags `schema:"tags"`
}

func (f Tag) Fields() map[string]string {
	return make(map[string]string)
}

func (f Tag) Validate() error {
	return nil
}
