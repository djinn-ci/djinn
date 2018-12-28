package runner

type Stage struct {
	Name    string
	CanFail bool
	Jobs    JobStore
}

func NewStage(name string, canFail bool) *Stage {
	return &Stage{
		Name:    name,
		CanFail: canFail,
		Jobs:    NewJobStore(),
	}
}

func (s *Stage) Add(jobs ...*Job) {
	for _, j := range jobs {
		j.Stage = s.Name
		j.CanFail = s.CanFail

		s.Jobs.Put(j)
	}
}
