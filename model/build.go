package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"database/sql"
	"strings"

	"github.com/andrewpillar/thrall/config"
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
	StartedAt   *pq.NullTime   `db:"started_at"`
	FinishedAt  *pq.NullTime   `db:"finished_at"`

	User      *User
	Namespace *Namespace
	Driver    *Driver
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

func (b *Build) Create() error {
	stmt, err := b.Prepare(`
		INSERT INTO builds (user_id, namespace_id, manifest)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(b.UserID, b.NamespaceID, b.Manifest)

	return errors.Err(row.Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt))
}

func (b *Build) IsZero() bool {
	return b.model.IsZero() &&
           b.UserID == 0 &&
           !b.NamespaceID.Valid &&
           b.Manifest == "" &&
           b.Status == runner.Status(0) &&
           !b.Output.Valid &&
           b.StartedAt == nil &&
           b.FinishedAt == nil
}

func (b *Build) Update() error {
	stmt, err := b.Prepare(`
		UPDATE builds
		SET status = $1, output = $2, started_at = $3, finished_at = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(b.Status, b.Output, b.StartedAt, b.FinishedAt, b.ID)

	return errors.Err(row.Scan(&b.UpdatedAt))
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

	setupJobs := setupStage.JobStore()
	parent := int64(0)

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

		if parent > 0 {
			j.ParentID = sql.NullInt64{
				Int64: parent,
				Valid: true,
			}
		}

		if err := j.Create(); err != nil {
			return errors.Err(err)
		}

		parent = j.ID
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
	}

	_, err = srv.SendTask(b.Signature())

	return errors.Err(err)
}

func (b Build) UIEndpoint() string {
	return fmt.Sprintf("/builds/%v", b.ID)
}

func (bs BuildStore) All() ([]*Build, error) {
	bb := make([]*Build, 0)

	query := "SELECT * FROM builds"
	args := []interface{}{}

	if bs.User != nil {
		query += " WHERE user_id = $1"
		args = append(args, bs.User.ID)
	}

	if bs.Namespace != nil {
		if bs.User != nil {
			query += " AND namespace_id = $2"
		} else {
			query += " WHERE namespace_id = $1"
		}

		args = append(args, bs.Namespace.ID)
	}

	query += " ORDER BY created_at DESC"

	err := bs.Select(&bb, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, b := range bb {
		b.DB = bs.DB
		b.User = bs.User
		b.Namespace = bs.Namespace
	}

	return bb, errors.Err(err)
}

func (bs BuildStore) ByStatus(status string) ([]*Build, error) {
	bb := make([]*Build, 0)

	query := "SELECT * FROM builds WHERE status = $1"
	args := []interface{}{status}

	if bs.User != nil {
		query += " AND user_id = $2"
		args = append(args, bs.User.ID)
	}

	if bs.Namespace != nil {
		if bs.User != nil {
			query += " AND namespace_id = $3"
		} else {
			query += " AND namespace_id = $2"
		}

		args = append(args, bs.Namespace.ID)
	}

	query += " ORDER BY created_at DESC"

	err := bs.Select(&bb, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, b := range bb {
		b.DB = bs.DB
		b.User = bs.User
		b.Namespace = bs.Namespace
	}

	return bb, errors.Err(err)
}

func (bs BuildStore) Find(id int64) (*Build, error) {
	b := &Build{
		model: model{
			DB: bs.DB,
		},
		User:      bs.User,
		Namespace: bs.Namespace,
	}

	query := "SELECT * FROM builds WHERE id = $1"
	args := []interface{}{id}

	if bs.User != nil {
		query += " AND user_id = $2"
		args = append(args, bs.User.ID)
	}

	if bs.Namespace != nil {
		if bs.User != nil {
			query += " AND namespace_id = $3"
		} else {
			query += " AND namespace_id = $2"
		}

		args = append(args, bs.Namespace.ID)
	}

	err := bs.Get(b, query, args...)

	if err == sql.ErrNoRows {
		err = nil

		b.CreatedAt = nil
		b.UpdatedAt = nil
	}

	return b, errors.Err(err)
}

func (bs *BuildStore) In(ids ...int64) ([]*Build, error) {
	bb := make([]*Build, 0)

	if len(ids) == 0 {
		return bb, nil
	}

	query, args, err := sqlx.In("SELECT * FROM builds WHERE id IN (?)", ids)

	if err != nil {
		return bb, errors.Err(err)
	}

	err = bs.Select(&bb, bs.Rebind(query), args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, b := range bb {
		b.DB = bs.DB
	}

	return bb, errors.Err(err)
}

func (bs *BuildStore) LoadNamespaces(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	ids := make([]int64, len(bb), len(bb))

	for i, b := range bb {
		if b.NamespaceID.Valid {
			ids[i] = b.NamespaceID.Int64
		}
	}

	namespaces := NamespaceStore{
		DB: bs.DB,
	}

	nn, err := namespaces.In(ids...)

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

	ids := make([]int64, len(bb), len(bb))

	for i, b := range bb {
		ids[i] = b.ID
	}

	tags := TagStore{
		DB: bs.DB,
	}

	tt, err := tags.InBuildID(ids...)

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

	ids := make([]int64, len(bb), len(bb))

	for i, b := range bb {
		ids[i] = b.UserID
	}

	users := UserStore{
		DB: bs.DB,
	}

	uu, err := users.In(ids...)

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
	stmt, err := bv.Prepare(`
		INSERT INTO build_variables (build_id, variable_id, key, value)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(bv.BuildID, bv.VariableID, bv.Key, bv.Value)

	return errors.Err(row.Scan(&bv.ID, &bv.CreatedAt, &bv.UpdatedAt))
}
