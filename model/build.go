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

	"github.com/andrewpillar/query"

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
	Secret      sql.NullString `db:"secret"`
	StartedAt   pq.NullTime    `db:"started_at"`
	FinishedAt  pq.NullTime    `db:"finished_at"`

	User      *User            `json:"-"`
	Namespace *Namespace       `json:"-"`
	Driver    *Driver          `json:"-"`
	Trigger   *Trigger         `json:"-"`
	Tags      []*Tag           `json:"-"`
	Stages    []*Stage         `json:"-"`
	Keys      []*BuildKey      `json:"-"`
	Objects   []*BuildObject   `json:"-"`
	Artifacts []*Artifact      `json:"-"`
	Variables []*BuildVariable `json:"-"`
}

type BuildStore struct {
	Store

	User      *User
	Namespace *Namespace
}

func buildToInterface(bb []*Build) func (i int) Interface {
	return func(i int) Interface {
		return bb[i]
	}
}

func BuildSearch(search string) query.Option {
	return func(q query.Query) query.Query {
		if search == "" {
			return q
		}

		return query.WhereQuery("id", "IN",
			query.Select(
				query.Columns("build_id"),
				query.From("tags"),
				query.Where("name", "LIKE", "%" + search + "%"),
			),
		)(q)
	}
}

func BuildStatus(status string) query.Option {
	return func(q query.Query) query.Query {
		if status == "" {
			return q
		}

		vals := []interface{}{
			status,
		}

		if status == "passed" {
			vals = append(vals, "passed_with_failures")
		}

		return query.Where("status", "IN", vals...)(q)
	}
}

func BuildTag(tag string) query.Option {
	return func(q query.Query) query.Query {
		if tag == "" {
			return q
		}

		return query.WhereQuery("id", "IN",
			query.Select(
				query.Columns("build_id"),
				query.From("tags"),
				query.Where("name", "=", tag),
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

func (b *Build) BuildKeyStore() BuildKeyStore {
	return BuildKeyStore{
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

func (b *Build) LoadKeys() error {
	var err error

	b.Keys, err = b.BuildKeyStore().All()

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
		"secret":       b.Secret,
		"started_at":   b.StartedAt,
		"finished_at":  b.FinishedAt,
	}
}

func (b Build) Signature() *tasks.Signature {
	buf := &bytes.Buffer{}

	enc := json.NewEncoder(buf)
	enc.Encode(b)

	return &tasks.Signature{
		Name:       "run_build",
		RetryCount: 3,
		Args:       []tasks.Arg{
			tasks.Arg{
				Type: "string",
				Value: buf.String(),
			},
		},
	}
}

func (b Build) Submit(srv *machinery.Server) error {
	m, _ := config.DecodeManifest(strings.NewReader(b.Manifest))

	images := ImageStore{
		Store: Store{
			DB: b.DB,
		},
		User: b.User,
	}

	image := m.Driver["image"]

	i, err := images.FindByName(image)

	if err != nil {
		return errors.Err(err)
	}

	if !i.IsZero() {
		m.Driver["image"] = i.Name + "::" + i.Hash
	}

	buf := &bytes.Buffer{}

	enc := json.NewEncoder(buf)
	enc.Encode(m.Driver)

	drivers := b.DriverStore()
	d := drivers.New()
	d.Config = buf.String()
	d.Type.UnmarshalText([]byte(m.Driver["type"]))

	if err := drivers.Create(d); err != nil {
		return errors.Err(err)
	}

	variables := VariableStore{
		Store: Store{
			DB: b.DB,
		},
		User: b.User,
	}

	vv, err := variables.All(
		query.WhereRaw("namespace_id", "IS", "NULL"),
		OrForNamespace(b.Namespace),
	)

	if err != nil {
		return errors.Err(err)
	}

	buildVars := b.BuildVariableStore()

	if err := buildVars.Copy(vv); err != nil {
		return errors.Err(err)
	}

	keys := KeyStore{
		Store: Store{
			DB: b.DB,
		},
		User: b.User,
	}

	kk, err := keys.All(
		query.WhereRaw("namespace_id", "IS", "NULL"),
		OrForNamespace(b.Namespace),
	)

	if err != nil {
		return errors.Err(err)
	}

	if err := b.BuildKeyStore().Copy(kk); err != nil {
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

	objects := ObjectStore{
		Store: Store{
			DB: b.DB,
		},
	}

	names := make([]interface{}, 0, len(m.Objects))

	for src := range m.Objects {
		names = append(names, src)
	}

	if len(names) > 0 {
		oo, err := objects.All(
			ForUser(b.User),
			query.WhereRaw("namespace_id", "IS", "NULL"),
			OrForNamespace(b.Namespace),
			query.Where("name", "IN", names...),
		)

		if err != nil {
			return errors.Err(err)
		}

		buildObjs := b.BuildObjectStore()

		for _, o := range oo {
			bo := buildObjs.New()
			bo.ObjectID = sql.NullInt64{
				Int64: o.ID,
				Valid: true,
			}
			bo.Source = o.Name
			bo.Name = m.Objects[o.Name]

			if err := buildObjs.Create(bo); err != nil {
				return errors.Err(err)
			}
		}
	}

	stages := b.StageStore()

	setup := stages.New()
	setup.Name = fmt.Sprintf("setup - #%v", b.ID)

	if err := stages.Create(setup); err != nil {
		return errors.Err(err)
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

	jobs := setup.JobStore()

	create := jobs.New()
	create.Name = "create driver"

	if err := jobs.Create(create); err != nil {
		return errors.Err(err)
	}

	for i, src := range m.Sources {
		commands := []string{
			"git clone " + src.URL + " " + src.Dir,
			"cd " + src.Dir,
			"git checkout -q " + src.Ref,
		}

		if src.Dir != "" {
			commands = append([]string{"mkdir -p " + src.Dir}, commands...)
		}

		j := jobs.New()
		j.Name = fmt.Sprintf("clone.%d", i + 1)
		j.Commands = strings.Join(commands, "\n")

		if err := jobs.Create(j); err != nil {
			return errors.Err(err)
		}
	}

	stage := ""
	jobId := 0

	jobModels := make(map[string]*Job)

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

		jobModels[s.Name + j.Name] = j

		for src, dst := range job.Artifacts {
			hash, _ := crypto.HashNow()

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

	_, err = srv.SendTask(b.Signature())

	return errors.Err(err)
}

func (s BuildStore) All(opts ...query.Option) ([]*Build, error) {
	bb := make([]*Build, 0)

	opts = append([]query.Option{ForCollaborator(s.User), ForNamespace(s.Namespace)}, opts...)

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
	models := interfaceSlice(len(bb), buildToInterface(bb))

	return errors.Err(s.Store.Create(BuildTable, models...))
}

func (s BuildStore) Paginate(page int64, opts ...query.Option) (Paginator, error) {
	paginator, err := s.Store.Paginate(BuildTable, page, opts...)

	return paginator, errors.Err(err)
}

func (s BuildStore) Find(id int64) (*Build, error) {
	b, err := s.findBy("id", id)

	return b, errors.Err(err)
}

func (s BuildStore) findBy(col string, val interface{}) (*Build, error) {
	b := &Build{
		Model: Model{
			DB: s.DB,
		},
		User:      s.User,
		Namespace: s.Namespace,
	}

	q := query.Select(
		query.Columns("*"),
		query.From(BuildTable),
		query.Where(col, "=", val),
		ForUser(s.User),
		ForNamespace(s.Namespace),
	)

	err := s.Get(b, q.Build(), q.Args()...)

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

func (s BuildStore) loadNamespace(bb []*Build) func(i int, n *Namespace) {
	return func(i int, n *Namespace) {
		b := bb[i]

		if b.NamespaceID.Int64 == n.ID {
			b.Namespace = n
		}
	}
}

func (s *BuildStore) LoadNamespaces(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	models := interfaceSlice(len(bb), buildToInterface(bb))

	namespaces := NamespaceStore{
		Store: s.Store,
	}

	err := namespaces.Load(mapKey("namespace_id", models), s.loadNamespace(bb))

	return errors.Err(err)
}

func (s *BuildStore) LoadTags(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	models := interfaceSlice(len(bb), buildToInterface(bb))

	tags := TagStore{
		Store: s.Store,
	}

	tt, err := tags.All(query.Where("build_id", "IN", mapKey("id", models)...))

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

func (s BuildStore) loadUser(bb []*Build) func(i int, u *User) {
	return func(i int, u *User) {
		b := bb[i]

		if b.UserID == u.ID {
			b.User = u
		}
	}
}

func (s *BuildStore) LoadUsers(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	models := interfaceSlice(len(bb), buildToInterface(bb))

	users := UserStore{
		Store: s.Store,
	}

	err := users.Load(mapKey("user_id", models), s.loadUser(bb))

	return errors.Err(err)
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
	models := interfaceSlice(len(bb), buildToInterface(bb))

	return errors.Err(s.Store.Update(BuildTable, models...))
}
