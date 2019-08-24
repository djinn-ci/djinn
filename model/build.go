package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"database/sql"
	"strings"

	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
	"github.com/andrewpillar/thrall/runner"

	"github.com/lib/pq"

	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/tasks"
)

type Build struct {
	Model

	UserID      int64          `db:"user_id"`
	NamespaceID sql.NullInt64  `db:"namespace_id"`
	Manifest    string         `db:"manifest"`
	Status      runner.Status  `db:"status"`
	Output      sql.NullString `db:"output"`
	StartedAt   pq.NullTime    `db:"started_at"`
	FinishedAt  pq.NullTime    `db:"finished_at"`

	User      *User
	Namespace *Namespace
	Driver    *Driver
	Trigger   *Trigger
	Tags      []*Tag
	Stages    []*Stage
	Objects   []*BuildObject
	Artifacts []*Artifact
	Variables []*BuildVariable
}

type BuildStore struct {
	Store

	User      *User
	Namespace *Namespace
}

func buildToInterface(bb ...*Build) func (i int) Interface {
	return func(i int) Interface {
		return bb[i]
	}
}

func BuildSearch(search string) query.Option {
	return func(q query.Query) query.Query {
		if search == "" {
			return q
		}

		return query.WhereInQuery("id",
			query.Select(
				query.Columns("build_id"),
				query.Table("tags"),
				query.WhereLike("name", "%" + search + "%"),
			),
		)(q)
	}
}

func BuildStatus(status string) query.Option {
	return func(q query.Query) query.Query {
		if status == "" {
			return q
		}

		return query.WhereEq("status", status)(q)
	}
}

func BuildTag(tag string) query.Option {
	return func(q query.Query) query.Query {
		if tag == "" {
			return q
		}

		return query.WhereInQuery("id",
			query.Select(
				query.Columns("build_id"),
				query.Table("tags"),
				query.WhereEq("name", tag),
			),
		)(q)
	}
}

func (b *Build) ArtifactStore() ArtifactStore {
	return ArtifactStore{
		Store: Store{
			DB: b.DB,
		},
		Build: b,
	}
}

func (b *Build) BuildObjectStore() BuildObjectStore {
	return BuildObjectStore{
		Store: Store{
			DB: b.DB,
		},
		Build: b,
	}
}

func (b *Build) BuildVariableStore() BuildVariableStore {
	return BuildVariableStore{
		Store: Store{
			DB: b.DB,
		},
		Build: b,
	}
}

func (b *Build) DriverStore() DriverStore {
	return DriverStore{
		Store: Store{
			DB: b.DB,
		},
		Build: b,
	}
}

func (b *Build) IsZero() bool {
	return b.Model.IsZero() &&
		b.UserID == 0 &&
		!b.NamespaceID.Valid &&
		b.Manifest == "" &&
		b.Status == runner.Status(0) &&
		!b.Output.Valid &&
		!b.StartedAt.Valid &&
		!b.FinishedAt.Valid
}

func (b *Build) JobStore() JobStore {
	return JobStore{
		Store: Store{
			DB: b.DB,
		},
		Build: b,
	}
}

func (b *Build) LoadArtifacts() error {
	var err error

	b.Artifacts, err = b.ArtifactStore().All()

	return errors.Err(err)
}

func (b *Build) LoadDriver() error {
	var err error

	b.Driver, err = b.DriverStore().First()

	return errors.Err(err)
}

func (b *Build) LoadNamespace() error {
	var err error

	namespaces := NamespaceStore{
		Store: Store{
			DB: b.DB,
		},
	}

	b.Namespace, err = namespaces.Find(b.NamespaceID.Int64)

	return errors.Err(err)
}

func (b *Build) LoadObjects() error {
	var err error

	b.Objects, err = b.BuildObjectStore().All()

	return errors.Err(err)
}

func (b *Build) LoadStages() error {
	var err error

	b.Stages, err = b.StageStore().All()

	return errors.Err(err)
}

func (b *Build) LoadTags() error {
	var err error

	b.Tags, err = b.TagStore().All()

	return errors.Err(err)
}

func (b *Build) LoadTrigger() error {
	var err error

	b.Trigger, err = b.TriggerStore().First()

	return errors.Err(err)
}

func (b *Build) LoadUser() error {
	var err error

	users := UserStore{
		Store: Store{
			DB: b.DB,
		},
	}

	b.User, err = users.Find(b.UserID)

	return errors.Err(err)
}

func (b *Build) LoadVariables() error {
	var err error

	b.Variables, err = b.BuildVariableStore().All()

	return errors.Err(err)
}

func (b *Build) StageStore() StageStore {
	return StageStore{
		Store: Store{
			DB: b.DB,
		},
		Build: b,
	}
}

func (b *Build) TagStore() TagStore {
	return TagStore{
		Store: Store{
			DB: b.DB,
		},
		User:  b.User,
		Build: b,
	}
}

func (b *Build) TriggerStore() TriggerStore {
	return TriggerStore{
		Store: Store{
			DB: b.DB,
		},
		Build: b,
	}
}

func (b Build) UIEndpoint(uri ...string) string {
	if b.User == nil || b.User.IsZero() {
		return ""
	}

	endpoint := fmt.Sprintf("/b/%s/%v", b.User.Username, b.ID)

	if len(uri) > 0 {
		endpoint = fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}

	return endpoint
}

func (b Build) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      b.UserID,
		"namespace_id": b.NamespaceID,
		"manifest":     b.Manifest,
		"status":       b.Status,
		"output":       b.Output,
		"started_at":   b.StartedAt,
		"finished_at":  b.FinishedAt,
	}
}

func (b Build) Signature() *tasks.Signature {
	return &tasks.Signature{
		Name:       "run_build",
		RetryCount: 3,
		Args:       []tasks.Arg{
			tasks.Arg{
				Type: "int64",
				Value: b.ID,
			},
		},
	}
}

func (b Build) Submit(srv *machinery.Server) error {
	m, _ := config.DecodeManifest(strings.NewReader(b.Manifest))

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)

	if err := enc.Encode(m.Driver); err != nil {
		return errors.Err(err)
	}

	drivers := b.DriverStore()

	d := drivers.New()
	d.Config = buf.String()
	d.Type.UnmarshalText([]byte(m.Driver.Type))

	if err := drivers.Create(d); err != nil {
		return errors.Err(err)
	}

	stages := b.StageStore()

	setupStage := stages.New()
	setupStage.Name = fmt.Sprintf("setup - #%v", b.ID)

	if err := stages.Create(setupStage); err != nil {
		return errors.Err(err)
	}

	jobs := setupStage.JobStore()

	createJob := jobs.New()
	createJob.Name = "create driver"

	if err := jobs.Create(createJob); err != nil {
		return errors.Err(err)
	}

	vv, err := b.User.VariableStore().All()

	if err != nil {
		return errors.Err(err)
	}

	buildVars := b.BuildVariableStore()

	if err := buildVars.Copy(vv); err != nil {
		return errors.Err(err)
	}

	for _, env := range m.Env {
		parts := strings.SplitN(env, "=", 2)

		bv := buildVars.New()
		bv.Key = parts[0]
		bv.Value = parts[1]

		if err := buildVars.Create(bv); err != nil {
			return errors.Err(err)
		}
	}

	buildObjs := b.BuildObjectStore()

	for src, dst := range m.Objects {
		o, err := b.User.ObjectStore().FindByName(src)

		if err != nil {
			return errors.Err(err)
		}

		bo := buildObjs.New()
		bo.ObjectID = sql.NullInt64{
			Int64: o.ID,
			Valid: o.ID > 0,
		}
		bo.Source = src
		bo.Name = dst

		if err := buildObjs.Create(bo); err != nil {
			return errors.Err(err)
		}
	}

	jdd := make([]*JobDependency, 0)

	setupJobs := setupStage.JobStore()

	prev := createJob

	for i, src := range m.Sources {
		commands := []string{
			"git clone " + src.URL + " " + src.Dir,
			"cd " + src.Dir,
			"git checkout -q " + src.Ref,
		}

		if src.Dir != "" {
			commands = append([]string{"mkdir -p " + src.Dir}, commands...)
		}

		j := setupJobs.New()
		j.Name = fmt.Sprintf("clone.%d", i + 1)
		j.Commands = strings.Join(commands, "\n")

		if err := setupJobs.Create(j); err != nil {
			return errors.Err(err)
		}

		jd := j.JobDependencyStore().New()
		jd.DependencyID = prev.ID

		jdd = append(jdd, jd)

		prev = j
	}

	stageModels := make(map[string]*Stage)

	for _, name := range m.Stages {
		canFail := false

		for _, allowed := range m.AllowFailures {
			if name == allowed {
				canFail = true
				break
			}
		}

		s := stages.New()
		s.Name = name
		s.CanFail = canFail

		if err := stages.Create(s); err != nil {
			return errors.Err(err)
		}

		stageModels[s.Name] = s
	}

	stage := ""
	jobId := 0

	stageJobs := make(map[string]*Job)

	for _, job := range m.Jobs {
		s, ok := stageModels[job.Stage]

		if !ok {
			continue
		}

		if s.Name != stage {
			stage = s.Name
			jobId = 0
		}

		jobId++

		if job.Name == "" {
			job.Name = fmt.Sprintf("%s.%d", job.Stage, jobId)
		}

		jobs := s.JobStore()

		j := jobs.New()
		j.Name = job.Name
		j.Commands = strings.Join(job.Commands, "\n")

		if err := jobs.Create(j); err != nil {
			return errors.Err(err)
		}

		stageJobs[s.Name + j.Name] = j

		for _, d := range job.Depends {
			dep, ok := stageJobs[s.Name + d]

			if !ok {
				continue
			}

			jd := j.JobDependencyStore().New()
			jd.DependencyID = dep.ID

			jdd = append(jdd, jd)
		}

		for src, dst := range job.Artifacts {
			hash, err := crypto.HashNow()

			if err != nil {
				return errors.Err(err)
			}

			artifacts := j.ArtifactStore()

			a := artifacts.New()
			a.Hash = hash
			a.Source = src
			a.Name = dst

			if err := artifacts.Create(a); err != nil {
				return errors.Err(err)
			}
		}
	}

	dependencies := JobDependencyStore{
		Store: Store{
			DB: b.DB,
		},
	}

	if err := dependencies.Create(jdd...); err != nil {
		return errors.Err(err)
	}

	_, err = srv.SendTask(b.Signature())

	return errors.Err(err)
}

func (s BuildStore) All(opts ...query.Option) ([]*Build, error) {
	bb := make([]*Build, 0)

	opts = append(opts, ForUser(s.User), ForNamespace(s.Namespace))

	err := s.Store.All(&bb, BuildTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, b := range bb {
		b.DB = s.DB
		b.User = s.User
		b.Namespace = s.Namespace
	}

	return bb, nil
}

func (s BuildStore) Create(bb ...*Build) error {
	models := interfaceSlice(len(bb), buildToInterface(bb...))

	return errors.Err(s.Store.Create(BuildTable, models...))
}

func (s BuildStore) Find(id int64) (*Build, error) {
	b := &Build{
		Model: Model{
			DB: s.DB,
		},
		User:      s.User,
		Namespace: s.Namespace,
	}

	err := s.FindBy(b, BuildTable, "id", id)

	if err == sql.ErrNoRows {
		err = nil
	}

	return b, errors.Err(err)
}

func (s BuildStore) Index(opts ...query.Option) ([]*Build, error) {
	bb, err := s.All(opts...)

	if err != nil {
		return bb, errors.Err(err)
	}

	if err := s.LoadNamespaces(bb); err != nil {
		return bb, errors.Err(err)
	}

	if err := s.LoadTags(bb); err != nil {
		return bb, errors.Err(err)
	}

	if err := s.LoadUsers(bb); err != nil {
		return bb, errors.Err(err)
	}

	nn := make([]*Namespace, 0, len(bb))

	for _, b := range bb {
		if b.Namespace != nil {
			nn = append(nn, b.Namespace)
		}
	}

	namespaces := NamespaceStore{
		Store: s.Store,
	}

	err = namespaces.LoadUsers(nn)

	return bb, errors.Err(err)
}

func (s BuildStore) Show(id int64) (*Build, error) {
	b, err := s.Find(id)

	if err != nil {
		return b, errors.Err(err)
	}

	if err := b.LoadNamespace(); err != nil {
		return b, errors.Err(err)
	}

	if err := b.Namespace.LoadUser(); err != nil {
		return b, errors.Err(err)
	}

	if err := b.LoadTrigger(); err != nil {
		return b, errors.Err(err)
	}

	if err := b.LoadTags(); err != nil {
		return b, errors.Err(err)
	}

	if err := b.LoadStages(); err != nil {
		return b, errors.Err(err)
	}

	err = b.StageStore().LoadJobs(b.Stages)

	return b, errors.Err(err)
}

func (s *BuildStore) LoadNamespaces(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	ids := make([]interface{}, len(bb), len(bb))

	for i, b := range bb {
		if b.NamespaceID.Valid {
			ids[i] = b.NamespaceID.Int64
		}
	}

	namespaces := NamespaceStore{
		Store: s.Store,
	}

	nn, err := namespaces.All(query.WhereIn("id", ids...))

	if err != nil {
		return errors.Err(err)
	}

	for _, b := range bb {
		for _, n := range nn {
			if b.NamespaceID.Int64 == n.ID {
				b.Namespace = n
			}
		}
	}

	return nil
}

func (s *BuildStore) LoadTags(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	ids := make([]interface{}, len(bb), len(bb))

	for i, b := range bb {
		ids[i] = b.ID
	}

	tags := TagStore{
		Store: s.Store,
	}

	tt, err := tags.All(query.WhereIn("build_id", ids...))

	if err != nil {
		return errors.Err(err)
	}

	for _, b := range bb {
		for _, t := range tt {
			if b.ID == t.BuildID {
				b.Tags = append(b.Tags, t)
			}
		}
	}

	return nil
}

func (s *BuildStore) LoadUsers(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	ids := make([]interface{}, len(bb), len(bb))

	for i, b := range bb {
		ids[i] = b.UserID
	}

	users := UserStore{
		Store: s.Store,
	}

	uu, err := users.All(query.WhereIn("id", ids...))

	if err != nil {
		return errors.Err(err)
	}

	for _, b := range bb {
		for _, u := range uu {
			if b.UserID == u.ID {
				b.User = u
			}
		}
	}

	return nil
}

func (s BuildStore) New() *Build {
	b := &Build{
		Model: Model{
			DB: s.DB,
		},
		User:      s.User,
		Namespace: s.Namespace,
	}

	if s.User != nil {
		b.UserID = s.User.ID
	}

	if s.Namespace != nil {
		b.NamespaceID = sql.NullInt64{
			Int64: s.Namespace.ID,
			Valid: true,
		}
	}

	return b
}

func (s BuildStore) Update(bb ...*Build) error {
	models := interfaceSlice(len(bb), buildToInterface(bb...))

	return errors.Err(s.Store.Update(BuildTable, models...))
}
