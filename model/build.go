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

	"github.com/andrewpillar/query"

	"github.com/lib/pq"

	"github.com/RichardKnop/machinery/v1/tasks"
)

type Build struct {
	Model

	UserID      int64           `db:"user_id"`
	NamespaceID sql.NullInt64   `db:"namespace_id"`
	Manifest    config.Manifest `db:"manifest"`
	Status      runner.Status   `db:"status"`
	Output      sql.NullString  `db:"output"`
	Secret      sql.NullString  `db:"secret"`
	StartedAt   pq.NullTime     `db:"started_at"`
	FinishedAt  pq.NullTime     `db:"finished_at"`

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
		b.Manifest.String() == "" &&
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

	b.Driver, err = b.DriverStore().Get()

	return errors.Err(err)
}

func (b *Build) LoadNamespace() error {
	var err error

	namespaces := NamespaceStore{
		Store: Store{
			DB: b.DB,
		},
	}

	b.Namespace, err = namespaces.Get(query.Where("id", "=", b.NamespaceID.Int64))

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

	b.Trigger, err = b.TriggerStore().Get()

	return errors.Err(err)
}

func (b *Build) LoadUser() error {
	var err error

	users := UserStore{
		Store: Store{
			DB: b.DB,
		},
	}

	b.User, err = users.Get(query.Where("id", "=", b.UserID))

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

func (s BuildStore) Get(opts ...query.Option) (*Build, error) {
	b := &Build{
		Model: Model{
			DB: s.DB,
		},
		User:      s.User,
		Namespace: s.Namespace,
	}

	baseOpts := []query.Option{
		query.Columns("*"),
		query.From(BuildTable),
		ForUser(s.User),
		ForNamespace(s.Namespace),
	}


	q := query.Select(append(baseOpts, opts...)...)

	err := s.Store.Get(b, q.Build(), q.Args()...)

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

func (s *BuildStore) LoadTriggers(bb []*Build) error {
	if len(bb) == 0 {
		return nil
	}

	models := interfaceSlice(len(bb), buildToInterface(bb))

	triggers := TriggerStore{
		Store: s.Store,
	}

	tt, err := triggers.All(query.Where("build_id", "IN", mapKey("id", models)...))

	if err != nil {
		return errors.Err(err)
	}

	for _, b := range bb {
		for _, t := range tt {
			if b.ID == t.BuildID {
				b.Trigger = t
			}
		}
	}

	return nil
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

func (s BuildStore) Paginate(page int64, opts ...query.Option) (Paginator, error) {
	paginator, err := s.Store.Paginate(BuildTable, page, opts...)

	return paginator, errors.Err(err)
}

func (s BuildStore) Update(bb ...*Build) error {
	models := interfaceSlice(len(bb), buildToInterface(bb))

	return errors.Err(s.Store.Update(BuildTable, models...))
}
