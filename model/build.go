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
	"github.com/andrewpillar/thrall/runner"

	"github.com/jmoiron/sqlx"

	"github.com/lib/pq"

	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/tasks"
)

type Build struct {
	model

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
	*sqlx.DB

	User      *User
	Namespace *Namespace
}

func BuildSearch(search string) Option {
	return func(q Query) Query {
		if search == "" {
			return q
		}

		return WhereInQuery("id",
			Select(
				Columns("build_id"),
				Table("tags"),
				WhereLike("name", "%" + search + "%"),
			),
		)(q)
	}
}

func BuildStatus(status string) Option {
	return func(q Query) Query {
		if status == "" {
			return q
		}

		return WhereEq("status", status)(q)
	}
}

func BuildTag(tag string) Option {
	return func(q Query) Query {
		if tag == "" {
			return q
		}

		return WhereInQuery("id",
			Select(
				Columns("build_id"),
				Table("tags"),
				WhereEq("name", tag),
			),
		)(q)
	}
}

func (b *Build) ArtifactStore() ArtifactStore {
	return ArtifactStore{
		DB:    b.DB,
		Build: b,
	}
}

func (b *Build) BuildObjectStore() BuildObjectStore {
	return BuildObjectStore{
		DB:    b.DB,
		Build: b,
	}
}

func (b *Build) BuildVariableStore() BuildVariableStore {
	return BuildVariableStore{
		DB:    b.DB,
		Build: b,
	}
}

func (b *Build) DriverStore() DriverStore {
	return DriverStore{
		DB:    b.DB,
		Build: b,
	}
}

func (b *Build) JobStore() JobStore {
	return JobStore{
		DB:    b.DB,
		Build: b,
	}
}

func (b *Build) StageStore() StageStore {
	return StageStore{
		DB:    b.DB,
		Build: b,
	}
}

func (b *Build) TagStore() TagStore {
	return TagStore{
		DB:    b.DB,
		User:  b.User,
		Build: b,
	}
}

func (b *Build) TriggerStore() TriggerStore {
	return TriggerStore{
		DB:    b.DB,
		Build: b,
	}
}

func (b *Build) Create() error {
	q := Insert(
		Columns("user_id", "namespace_id", "manifest"),
		Table("builds"),
		Values(b.UserID, b.NamespaceID, b.Manifest),
		Returning("id", "created_at", "updated_at"),
	)

	stmt, err := b.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt))
}

func (b *Build) IsZero() bool {
	return b.model.IsZero() &&
		b.UserID == 0 &&
		!b.NamespaceID.Valid &&
		b.Manifest == "" &&
		b.Status == runner.Status(0) &&
		!b.Output.Valid &&
		!b.StartedAt.Valid &&
		!b.FinishedAt.Valid
}

func (b *Build) Update() error {
	q := Update(
		Table("builds"),
		Set("status", b.Status),
		Set("output", b.Output),
		Set("started_at", b.StartedAt),
		Set("finished_at", b.FinishedAt),
		SetRaw("updated_at", "NOW()"),
		WhereEq("id", b.ID),
		Returning("updated_at"),
	)

	stmt, err := b.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&b.UpdatedAt))
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

	d := b.DriverStore().New()
	d.Config = buf.String()
	d.Type.UnmarshalText([]byte(m.Driver.Type))

	if err := d.Create(); err != nil {
		return errors.Err(err)
	}

	setupStage := b.StageStore().New()
	setupStage.Name = fmt.Sprintf("setup - #%v", b.ID)

	if err := setupStage.Create(); err != nil {
		return errors.Err(err)
	}

	createJob := setupStage.JobStore().New()
	createJob.Name = "create driver"

	if err := createJob.Create(); err != nil {
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

		if err := bv.Create(); err != nil {
			return errors.Err(err)
		}
	}

	for src, dst := range m.Objects {
		o, err := b.User.ObjectStore().FindByName(src)

		if err != nil {
			return errors.Err(err)
		}

		bo := b.BuildObjectStore().New()
		bo.ObjectID = sql.NullInt64{
			Int64: o.ID,
			Valid: o.ID > 0,
		}
		bo.Source = src
		bo.Name = dst

		if err := bo.Create(); err != nil {
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

		if err := j.Create(); err != nil {
			return errors.Err(err)
		}

		jd := j.JobDependencyStore().New()
		jd.DependencyID = prev.ID

		jdd = append(jdd, jd)

		prev = j
	}

	stages := b.StageStore()
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

		if err := s.Create(); err != nil {
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

		j := s.JobStore().New()
		j.Name = job.Name
		j.Commands = strings.Join(job.Commands, "\n")

		if err := j.Create(); err != nil {
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

			a := j.ArtifactStore().New()
			a.Hash = hash
			a.Source = src
			a.Name = dst

			if err := a.Create(); err != nil {
				return errors.Err(err)
			}
		}
	}

	for _, jd := range jdd {
		if err := jd.Create(); err != nil {
			return errors.Err(err)
		}
	}

	_, err = srv.SendTask(b.Signature())

	return errors.Err(err)
}

func (bs BuildStore) All(opts ...Option) ([]*Build, error) {
	bb := make([]*Build, 0)

	opts = append([]Option{Columns("*")}, opts...)

	q := Select(append(opts, ForUser(bs.User), ForNamespace(bs.Namespace), Table("builds"))...)

	if err := bs.Select(&bb, q.Build(), q.Args()...); err != nil {
		if err == sql.ErrNoRows {
			return bb, nil
		}

		return bb, errors.Err(err)
	}

	for _, b := range bb {
		b.DB = bs.DB
		b.User = bs.User
		b.Namespace = bs.Namespace
	}

	return bb, nil
}

func (bs BuildStore) Find(id int64) (*Build, error) {
	b := &Build{
		model: model{
			DB: bs.DB,
		},
		User:      bs.User,
		Namespace: bs.Namespace,
	}

	q := Select(
		Columns("*"),
		Table("builds"),
		WhereEq("id", id),
		ForUser(bs.User),
		ForNamespace(bs.Namespace),
	)

	err := bs.Get(b, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return b, errors.Err(err)
}

func (bs BuildStore) Index(opts ...Option) ([]*Build, error) {
	bb, err := bs.All(opts...)

	if err != nil {
		return bb, errors.Err(err)
	}

	if err != nil {
		return bb, errors.Err(err)
	}

	if err := bs.LoadNamespaces(bb); err != nil {
		return bb, errors.Err(err)
	}

	if err := bs.LoadTags(bb); err != nil {
		return bb, errors.Err(err)
	}

	if err := bs.LoadUsers(bb); err != nil {
		return bb, errors.Err(err)
	}

	nn := make([]*Namespace, 0, len(bb))

	for _, b := range bb {
		if b.Namespace != nil {
			nn = append(nn, b.Namespace)
		}
	}

	ns := NamespaceStore{
		DB: bs.DB,
	}

	err = ns.LoadUsers(nn)

	return bb, errors.Err(err)
}

func (bs BuildStore) Show(id int64) (*Build, error) {
	b, err := bs.Find(id)

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
		DB: b.DB,
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
		DB: b.DB,
	}

	b.User, err = users.Find(b.UserID)

	return errors.Err(err)
}

func (b *Build) LoadVariables() error {
	var err error

	b.Variables, err = b.BuildVariableStore().All()

	return errors.Err(err)
}


func (b Build) UIEndpoint(uri ...string) string {
	endpoint := fmt.Sprintf("/builds/%v", b.ID)

	if len(uri) > 0 {
		endpoint = fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}

	return endpoint
}

func (bs *BuildStore) LoadNamespaces(bb []*Build) error {
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
		DB: bs.DB,
	}

	nn, err := namespaces.All(WhereIn("id", ids...))

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

func (bs *BuildStore) LoadTags(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	ids := make([]interface{}, len(bb), len(bb))

	for i, b := range bb {
		ids[i] = b.ID
	}

	tags := TagStore{
		DB: bs.DB,
	}

	tt, err := tags.All(WhereIn("build_id", ids...))

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

func (bs *BuildStore) LoadUsers(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	ids := make([]interface{}, len(bb), len(bb))

	for i, b := range bb {
		ids[i] = b.UserID
	}

	users := UserStore{
		DB: bs.DB,
	}

	uu, err := users.All(WhereIn("id", ids...))

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

func (bs BuildStore) New() *Build {
	b := &Build{
		model: model{
			DB: bs.DB,
		},
		User:      bs.User,
		Namespace: bs.Namespace,
	}

	if bs.User != nil {
		b.UserID = bs.User.ID
	}

	if bs.Namespace != nil {
		b.NamespaceID = sql.NullInt64{
			Int64: bs.Namespace.ID,
			Valid: true,
		}
	}

	return b
}

func (bv *BuildVariable) Create() error {
	q := Insert(
		Columns("build_id", "variable_id", "key", "value"),
		Table("build_variables"),
		Values(bv.BuildID, bv.VariableID, bv.Key, bv.Value),
		Returning("id", "created_at", "updated_at"),
	)

	stmt, err := bv.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&bv.ID, &bv.CreatedAt, &bv.UpdatedAt))
}
