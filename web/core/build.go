package core

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/config"
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

	Namespaces model.NamespaceStore
	Builds     model.BuildStore
	Client     *redis.Client
	Drivers    map[string]struct{}
	Queue      *machinery.Server
}

func (h Build) Build(r *http.Request) *model.Build {
	val := r.Context().Value("build")

	b, _ := val.(*model.Build)

	return b
}

func (h Build) Index(builds model.BuildStore, r *http.Request, opts ...query.Option) ([]*model.Build, error) {
	q := r.URL.Query()

	tag := q.Get("tag")
	search := q.Get("search")
	status := q.Get("status")

	index := []query.Option{
		model.BuildTag(tag),
		model.BuildSearch(search),
		model.BuildStatus(status),
		query.OrderDesc("created_at"),
	}

	bb, err := builds.All(append(index, opts...)...)

	if err != nil {
		return bb, errors.Err(err)
	}

	if err := builds.LoadNamespaces(bb); err != nil {
		return bb, errors.Err(err)
	}

	if err := builds.LoadTags(bb); err != nil {
		return bb, errors.Err(err)
	}

	if err := builds.LoadUsers(bb); err != nil {
		return bb, errors.Err(err)
	}

	nn := make([]*model.Namespace, 0, len(bb))

	for _, b := range bb {
		if b.Namespace != nil {
			nn = append(nn, b.Namespace)
		}
	}

	err = h.Namespaces.LoadUsers(nn)

	return bb, errors.Err(err)
}

func (h Build) Kill(r *http.Request) error {
	b := h.Build(r)

	if b.Status != runner.Running {
		return ErrBuildNotRunning
	}

	_, err := h.Client.Publish("kill", b.Secret.String).Result()

	return errors.Err(err)
}

func (h Build) Show(r *http.Request) (*model.Build, error) {
	b := h.Build(r)

	if err := b.LoadNamespace(); err != nil {
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

	if err := h.ValidateForm(f, w, r); err != nil {
		return &model.Build{}, ErrValidationFailed
	}

	manifest, _ := config.DecodeManifest(strings.NewReader(f.Manifest))

	if _, ok := h.Drivers[manifest.Driver["type"]]; !ok {
		return &model.Build{}, ErrUnsupportedDriver
	}

	secret, err := crypto.HashNow()

	if err != nil {
		return &model.Build{}, errors.Err(err)
	}

	builds := u.BuildStore()

	b := builds.New()
	b.User = u
	b.Manifest = f.Manifest
	b.Secret = sql.NullString{
		String: secret,
		Valid:  true,
	}

	if f.Namespace != "" {
		n, err := u.NamespaceStore().FindOrCreate(f.Namespace)

		if err != nil {
			return b, errors.Err(err)
		}

		b.Namespace = n
		b.NamespaceID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}
	}

	if err := builds.Create(b); err != nil {
		return b, errors.Err(err)
	}

	triggers := b.TriggerStore()

	t := triggers.New()
	t.Type = types.Manual
	t.Comment = f.Comment
	t.Data.User = u.Username
	t.Data.Email = u.Email

	if err := triggers.Create(t); err != nil {
		return b, errors.Err(err)
	}

	if err := b.Submit(h.Queue); err != nil {
		return b, errors.Err(err)
	}

	tags := b.TagStore()
	tt := make([]*model.Tag, 0, len(f.Tags))

	for _, name := range f.Tags {
		t := tags.New()
		t.UserID = u.ID
		t.Name = name

		tt = append(tt, t)
	}

	err = tags.Create(tt...)

	return b, errors.Err(err)
}
