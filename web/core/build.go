package core

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/crypto"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/model/types"
	"github.com/andrewpillar/thrall/runner"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/go-redis/redis"

	"github.com/RichardKnop/machinery/v1"
)

type Build struct {
	web.Handler

	Namespace  Namespace
	Namespaces model.NamespaceStore
	Builds     model.BuildStore
	Images     model.ImageStore
	Variables  model.VariableStore
	Keys       model.KeyStore
	Objects    model.ObjectStore
	Client     *redis.Client
	Queues     map[string]*machinery.Server
}

func (h Build) Build(r *http.Request) *model.Build {
	val := r.Context().Value("build")

	b, _ := val.(*model.Build)

	return b
}

func (h Build) Index(builds model.BuildStore, r *http.Request, opts ...query.Option) ([]*model.Build, model.Paginator, error) {
	q := r.URL.Query()

	tag := q.Get("tag")
	search := q.Get("search")
	status := q.Get("status")
	page, err := strconv.ParseInt(q.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	index := []query.Option{
		model.BuildTag(tag),
		model.BuildSearch(search),
		model.BuildStatus(status),
	}

	paginator, err := builds.Paginate(page, append(index, opts...)...)

	if err != nil {
		return []*model.Build{}, paginator, errors.Err(err)
	}

	index = append(index, query.OrderDesc("created_at"))

	bb, err := builds.All(append(index, opts...)...)

	if err != nil {
		return bb, paginator, errors.Err(err)
	}

	if err := builds.LoadNamespaces(bb); err != nil {
		return bb, paginator, errors.Err(err)
	}

	if err := builds.LoadTags(bb); err != nil {
		return bb, paginator, errors.Err(err)
	}

	if err := builds.LoadUsers(bb); err != nil {
		return bb, paginator, errors.Err(err)
	}

	nn := make([]*model.Namespace, 0, len(bb))

	for _, b := range bb {
		if b.Namespace != nil {
			nn = append(nn, b.Namespace)
		}
	}

	err = h.Namespaces.LoadUsers(nn)

	return bb, paginator, errors.Err(err)
}

func (h Build) Kill(r *http.Request) error {
	b := h.Build(r)

	if b.Status != runner.Running {
		return ErrBuildNotRunning
	}

	_, err := h.Client.Publish(fmt.Sprintf("kill-%v", b.ID), b.Secret.String).Result()

	return errors.Err(err)
}

func (h Build) Show(r *http.Request) (*model.Build, error) {
	b := h.Build(r)

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

	err := b.StageStore().LoadJobs(b.Stages)

	return b, errors.Err(err)
}

func (h Build) Store(w http.ResponseWriter, r *http.Request) (*model.Build, error) {
	u := h.User(r)

	f := &form.Build{}

	if err := form.Unmarshal(f, r); err != nil {
		cause := errors.Cause(err)

		if strings.Contains(cause.Error(), "cannot unmarshal") {
			errs := form.NewErrors()
			errs.Put("manifest", errors.New("Build manifest is invalid, check the YAML is valid"))

			h.FlashErrors(w, r, errs)

			return &model.Build{}, ErrValidationFailed
		}

		return &model.Build{}, errors.Err(err)
	}

	if err := f.Validate(); err != nil {
		h.FlashErrors(w, r, err.(form.Errors))
		h.FlashForm(w, r, f)

		return &model.Build{}, ErrValidationFailed
	}

	if _, ok := h.Queues[f.Manifest.Driver["type"]]; !ok {
		return &model.Build{}, ErrUnsupportedDriver
	}

	secret, _ := crypto.HashNow()

	b := u.BuildStore().New()
	b.User = u
	b.Manifest = f.Manifest
	b.Secret = sql.NullString{
		String: secret,
		Valid:  true,
	}

	if f.Manifest.Namespace != "" {
		n, err := h.Namespace.Get(f.Manifest.Namespace, u)

		if err != nil {
			return b, errors.Err(err)
		}

		if !n.CanAdd(u) {
			return b, ErrAccessDenied
		}

		b.Namespace = n
		b.NamespaceID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}
	}

	b.Trigger = b.TriggerStore().New()
	b.Trigger.Type = types.Manual
	b.Trigger.Comment = f.Comment
	b.Trigger.Data.Set("email", u.Email)
	b.Trigger.Data.Set("username", u.Username)

	for _, name := range f.Tags {
		t := &model.Tag{
			UserID: u.ID,
			Name:   name,
		}

		b.Tags = append(b.Tags, t)
	}

	if err := h.Builds.Create(b); err != nil {
		return b, errors.Err(err)
	}

	b.Trigger.BuildID = b.ID

	if err := b.TriggerStore().Create(b.Trigger); err != nil {
		return b, errors.Err(err)
	}

	for _, t := range b.Tags {
		t.BuildID = b.ID
	}

	err := b.TagStore().Create(b.Tags...)

	return b, errors.Err(err)
}

func (h Build) Submit(b *model.Build, srv *machinery.Server) error {
	i, err := h.Images.FindByName(b.Manifest.Driver["image"])

	if err != nil {
		return errors.Err(err)
	}

	if !i.IsZero() {
		b.Manifest.Driver["image"] = i.Name + "::" + i.Hash
	}

	buf := &bytes.Buffer{}

	enc := json.NewEncoder(buf)
	enc.Encode(b.Manifest.Driver)

	drivers := b.DriverStore()

	d := drivers.New()
	d.Config = buf.String()
	d.Type.UnmarshalText([]byte(b.Manifest.Driver["type"]))

	if err := drivers.Create(d); err != nil {
		return errors.Err(err)
	}

	vv, err := h.Variables.All(
		query.WhereRaw("namespace_id", "IS", "NULL"),
		model.OrForNamespace(b.Namespace),
	)

	if err != nil {
		return errors.Err(err)
	}

	buildVars := b.BuildVariableStore()

	if err := buildVars.Copy(vv); err != nil {
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
		query.WhereRaw("namespace_id", "IS", "NULL"),
		model.OrForNamespace(b.Namespace),
	)

	if err != nil {
		return errors.Err(err)
	}

	if err := b.BuildKeyStore().Copy(kk); err != nil {
		return errors.Err(err)
	}

	names := make([]interface{}, 0, b.Manifest.Objects.Len())

	for src := range b.Manifest.Objects.Values() {
		names = append(names, src)
	}

	if len(names) > 0 {
		oo, err := h.Objects.All(
			model.ForUser(b.User),
			query.WhereRaw("namespace_id", "IS", "NULL"),
			model.OrForNamespace(b.Namespace),
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
			bo.Name = b.Manifest.Objects.Values()[o.Name]

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

	stageModels := make(map[string]*model.Stage)

	for _, name := range b.Manifest.Stages {
		canFail := false

		for _, allowed := range b.Manifest.AllowFailures {
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

	for i, src := range b.Manifest.Sources {
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

	jobModels := make(map[string]*model.Job)

	for _, job := range b.Manifest.Jobs {
		s, ok := stageModels[job.Stage]

		if !ok {
			continue
		}

		if s.Name != stage {
			stage = s.Name
			jobId = 0
		}

		jobId++

		jobs := s.JobStore()

		j := jobs.New()
		j.Name = job.Name
		j.Commands = strings.Join(job.Commands, "\n")

		if err := jobs.Create(j); err != nil {
			return errors.Err(err)
		}

		jobModels[s.Name + j.Name] = j

		for src, dst := range job.Artifacts.Values() {
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
