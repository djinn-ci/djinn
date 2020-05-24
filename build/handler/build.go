package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/build"
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/filestore"
	"github.com/andrewpillar/thrall/image"
	"github.com/andrewpillar/thrall/key"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/object"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/variable"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/go-redis/redis"

	"github.com/RichardKnop/machinery/v1"
)

type Build struct {
	web.Handler

	Loaders    model.Loaders
	Builds     *build.Store
	Tags       *build.TagStore
	Triggers   *build.TriggerStore
	Stages     *build.StageStore
	Jobs       *build.JobStore
	Artifacts  *build.ArtifactStore
	Keys       *key.Store
	Namespaces *namespace.Store
	Objects    *object.Store
	Providers  *provider.Store
	Images     *image.Store
	Variables  *variable.Store
	FileStore  filestore.FileStore

	Client          *redis.Client
	Queues          map[string]*machinery.Server
	Oauth2Providers map[string]oauth2.Provider
}

func copyKeys(s *build.KeyStore, kk []*key.Key) error {
	bkk := make([]*build.Key, 0, len(kk))

	for _, k := range kk {
		bk := s.New()
		bk.KeyID = sql.NullInt64{
			Int64: k.ID,
			Valid: true,
		}
		bk.Name = k.Name
		bk.Key = k.Key
		bk.Config = k.Config
		bk.Location = "/root/.ssh/"+bk.Name
		bkk = append(bkk, bk)
	}
	return errors.Err(s.Create(bkk...))
}

func copyVariables(s *build.VariableStore, vv []*variable.Variable) error {
	bvv := make([]*build.Variable, 0, len(vv))

	for _, v := range vv {
		bv := s.New()
		bv.VariableID = sql.NullInt64{
			Int64: v.ID,
			Valid: true,
		}
		bv.Key = v.Key
		bv.Value = v.Value
		bvv = append(bvv, bv)
	}
	return errors.Err(s.Create(bvv...))
}

// Model returns the *build.Build from the current request context.
func Model(r *http.Request) *build.Build {
	val := r.Context().Value("build")
	b, _ := val.(*build.Build)
	return b
}

func (h Build) objectsWithRelations(b *build.Build) ([]*build.Object, error) {
	oo, err := build.NewObjectStore(h.DB, b).All()

	if err != nil {
		return oo, errors.Err(err)
	}

	mm := model.Slice(len(oo), build.ObjectModel(oo))

	err = h.Objects.Load("id", model.MapKey("object_id", mm), model.Bind("object_id", "id", mm...))
	return oo, errors.Err(err)
}

func (h Build) variablesWithRelations(b *build.Build) ([]*build.Variable, error) {
	vv, err := build.NewVariableStore(h.DB, b).All()

	if err != nil {
		return vv, errors.Err(err)
	}

	mm := model.Slice(len(vv), build.VariableModel(vv))

	err = h.Variables.Load("id", model.MapKey("variable_id", mm), model.Bind("variable_id", "id", mm...))
	return vv, errors.Err(err)
}

// IndexWithRelations returns a slice of paginated Build models with the
// relationships loaded.
func (h Build) IndexWithRelations(s *build.Store, vals url.Values) ([]*build.Build, model.Paginator, error) {
	bb, paginator, err := s.Index(vals)

	if err != nil {
		return bb, paginator, errors.Err(err)
	}

	if err := build.LoadRelations(h.Loaders, bb...); err != nil {
		return bb, paginator, errors.Err(err)
	}

	nn := make([]model.Model, 0, len(bb))

	for _, b := range bb {
		if b.Namespace != nil {
			nn = append(nn, b.Namespace)
		}
	}

	err = h.Users.Load("id", model.MapKey("user_id", nn), model.Bind("user_id", "id", nn...))
	return bb, paginator, errors.Err(err)
}

// ShowWithRelations returns a single Build model with the relationships
// loaded.
func (h Build) ShowWithRelations(r *http.Request) (*build.Build, error) {
	b := Model(r)

	if err := build.LoadRelations(h.Loaders, b); err != nil {
		return b, errors.Err(err)
	}

	ss := make([]model.Model, 0, len(b.Stages))

	for _, s := range b.Stages {
		ss = append(ss, s)
	}

	err := h.Jobs.Load("stage_id", model.MapKey("id", ss), model.Bind("id", "stage_id", ss...))

	if err != nil {
		return b, errors.Err(err)
	}

	if b.Namespace != nil {
		err = h.Users.Load(
			"id",
			[]interface{}{b.Namespace.Values()["user_id"]},
			model.Bind("user_id", "id", b.Namespace),
		)
		return b, errors.Err(err)
	}
	return b, nil
}

func (h Build) realStore(m config.Manifest, u *user.User, t *build.Trigger, tags ...string) (*build.Build, error) {
	if _, ok := h.Queues[m.Driver["type"]]; !ok {
		return &build.Build{}, build.ErrDriver
	}

	secret, _ := crypto.HashNow()

	builds := build.NewStore(h.DB, u)

	b := builds.New()
	b.Manifest = m
	b.Secret = sql.NullString{
		String: secret,
		Valid:  true,
	}

	if m.Namespace != "" {
		n, err := namespace.NewStore(h.DB, u).GetByPath(m.Namespace)

		if err != nil {
			if err == namespace.ErrName {
				return b, err
			}
			return b, errors.Err(err)
		}

		if !n.CanAdd(u) {
			return b, namespace.ErrPermission
		}

		b.Namespace = n
		b.NamespaceID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}
	}

	for _, name := range tags {
		t := &build.Tag{
			UserID: u.ID,
			Name:   name,
		}
		b.Tags = append(b.Tags, t)
	}

	if err := builds.Create(b); err != nil {
		return b, errors.Err(err)
	}

	t.BuildID = b.ID

	if err := build.NewTriggerStore(h.DB, b).Create(t); err != nil {
		return b, errors.Err(err)
	}

	for _, t := range b.Tags {
		t.BuildID = b.ID
	}

	err := build.NewTagStore(h.DB, b).Create(b.Tags...)
	return b, errors.Err(err)
}

// StoreModel stores a new Build model in the database. It takes the current
// request session to flash any data to if the given session is not nil.
func (h Build) StoreModel(r *http.Request, sess *sessions.Session) (*build.Build, error) {
	defer r.Body.Close()

	u := h.User(r)
	f := &build.Form{}

	if err := h.ValidateForm(f, r, sess); err != nil {
		return &build.Build{}, errors.Err(err)
	}

	t := &build.Trigger{
		Type:    build.Manual,
		Comment: f.Comment,
		Data:    map[string]string{
			"email":    u.Email,
			"username": u.Username,
		},
	}

	b, err := h.realStore(f.Manifest, u, t, []string(f.Tags)...)

	if err != nil && sess != nil {
		sess.AddFlash(f.Fields(), "form_fields")

		if errors.Cause(err) == namespace.ErrName {
			errs := form.NewErrors()
			errs.Put("manifest", errors.New("Namespace name can only contain letters and numbers"))
			sess.AddFlash(errs, "form_errors")
		}
	}
	return b, errors.Err(err)
}

// Submit puts the given Build model onto the given queue server for running.
func (h Build) Submit(b *build.Build, srv *machinery.Server) error {
	buf := &bytes.Buffer{}

	enc := json.NewEncoder(buf)
	enc.Encode(b.Manifest.Driver)

	drivers := build.NewDriverStore(h.DB, b)

	d := drivers.New()
	d.Config = buf.String()
	d.Type.UnmarshalText([]byte(b.Manifest.Driver["type"]))

	if err := drivers.Create(d); err != nil {
		return errors.Err(err)
	}

	vv, err := h.Variables.All(
		query.Where("user_id", "=", b.UserID),
		query.WhereRaw("namespace_id", "IS", "NULL"),
		model.OrWhere(b.Namespace, "namespace_id"),
	)

	if err != nil {
		return errors.Err(err)
	}

	buildVars := build.NewVariableStore(h.DB, b)

	if err := copyVariables(buildVars, vv); err != nil {
		return errors.Err(err)
	}

	for _, env := range b.Manifest.Env {
		parts := strings.SplitN(env, "=", 2)

		bv := buildVars.New()
		bv.Key = parts[0]
		bv.Value = parts[1]

		if err := buildVars.Create(bv); err != nil {
			return errors.Err(err)
		}
	}

	kk, err := h.Keys.All(
		query.Where("user_id", "=", b.UserID),
		query.WhereRaw("namespace_id", "IS", "NULL"),
		model.OrWhere(b.Namespace, "namespace_id"),
	)

	if err != nil {
		return errors.Err(err)
	}

	if err := copyKeys(build.NewKeyStore(h.DB, b), kk); err != nil {
		return errors.Err(err)
	}

	names := make([]interface{}, 0, len(b.Manifest.Objects.Values))

	for src := range b.Manifest.Objects.Values {
		names = append(names, src)
	}

	if len(names) > 0 {
		oo, err := h.Objects.All(
			query.Where("name", "IN", names...),
			query.Where("user_id", "=", b.UserID),
			query.WhereRaw("namespace_id", "IS", "NULL"),
			model.OrWhere(b.Namespace, "namespace_id"),
		)

		if err != nil {
			return errors.Err(err)
		}

		buildObjs := build.NewObjectStore(h.DB, b)

		for _, o := range oo {
			bo := buildObjs.New()
			bo.ObjectID = sql.NullInt64{
				Int64: o.ID,
				Valid: true,
			}
			bo.Source = o.Name
			bo.Name = b.Manifest.Objects.Values[o.Name]

			if err := buildObjs.Create(bo); err != nil {
				return errors.Err(err)
			}
		}
	}

	stages := build.NewStageStore(h.DB, b)

	setup := stages.New()
	setup.Name = fmt.Sprintf("setup - #%v", b.ID)

	if err := stages.Create(setup); err != nil {
		return errors.Err(err)
	}

	canFail := make(map[string]struct{})

	for _, stage := range b.Manifest.AllowFailures {
		canFail[stage] = struct{}{}
	}

	stageModels := make(map[string]*build.Stage)

	for _, name := range b.Manifest.Stages {
		_, ok := canFail[name]

		s := stages.New()
		s.Name = name
		s.CanFail = ok

		if err := stages.Create(s); err != nil {
			return errors.Err(err)
		}
		stageModels[s.Name] = s
	}

	jobs := build.NewJobStore(h.DB, b, setup)

	create := jobs.New()
	create.Name = "create driver"

	if err := jobs.Create(create); err != nil {
		return errors.Err(err)
	}

	for i, src := range b.Manifest.Sources {
		commands := []string{
			"git clone "+src.URL+" "+src.Dir,
			"cd "+src.Dir,
			"git checkout -q "+src.Ref,
		}

		if src.Dir != "" {
			commands = append([]string{"mkdir -p "+src.Dir}, commands...)
		}

		j := jobs.New()
		j.Name = fmt.Sprintf("clone.%d", i+1)
		j.Commands = strings.Join(commands, "\n")

		if err := jobs.Create(j); err != nil {
			return errors.Err(err)
		}
	}

	stage := ""
	jobId := 1

	for _, job := range b.Manifest.Jobs {
		s, ok := stageModels[job.Stage]

		if !ok {
			continue
		}

		if s.Name != stage {
			stage = s.Name
			jobId = 1
		} else {
			jobId++
		}

		jobs := build.NewJobStore(h.DB, b, s)

		j := jobs.New()
		j.Name = job.Name
		j.Commands = strings.Join(job.Commands, "\n")

		if j.Name == "" {
			j.Name = fmt.Sprintf("%s.%v", s.Name, jobId)
		}

		if err := jobs.Create(j); err != nil {
			return errors.Err(err)
		}

		for src, dst := range job.Artifacts.Values {
			hash, _ := crypto.HashNow()

			artifacts := build.NewArtifactStore(h.DB, j, b)

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

func (h Build) JobGet(r *http.Request) (*build.Job, error) {
	b := Model(r)

	if err := build.LoadRelations(h.Loaders, b); err != nil {
		return &build.Job{}, errors.Err(err)
	}

	vars := mux.Vars(r)

	id, _ := strconv.ParseInt(vars["job"], 10, 64)

	j, err := build.NewJobStore(h.DB, b).Get(query.Where("id", "=", id))

	if err != nil {
		return j, errors.Err(err)
	}

	if err := h.Stages.Load("id", []interface{}{j.StageID}, model.Bind("stage_id", "id", j)); err != nil {
		return j, errors.Err(err)
	}

	err = h.Artifacts.Load("job_id", []interface{}{j.ID}, model.Bind("id", "job_id", j))
	return j, errors.Err(err)
}
