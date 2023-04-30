package namespace

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/queue"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil/v2"
)

type Namespace struct {
	loaded    []string
	collabtab map[int64]struct{}

	ID          int64
	UserID      int64
	RootID      database.Null[int64]
	ParentID    database.Null[int64]
	Name        string
	Path        string
	Description string
	Level       int64
	Visibility  Visibility
	CreatedAt   time.Time

	User   *auth.User
	Parent *Namespace
	Build  database.Model
}

func CanAccess(db *database.Pool, u *auth.User) webutil.ValidatorFunc {
	return func(ctx context.Context, val any) error {
		p := val.(Path)

		if p.Valid {
			_, n, err := p.Resolve(ctx, db, u)

			if err != nil {
				if _, ok := err.(*PathError); !ok {
					return errors.Err(err)
				}
				return err
			}

			if err := n.IsCollaborator(ctx, db, u); err != nil {
				return err
			}
		}
		return nil
	}
}

func ResourceUnique[M database.Model](s *database.Store[M], u *auth.User, field string, p Path) webutil.ValidatorFunc {
	return func(ctx context.Context, val any) error {
		opts := []query.Option{
			query.Where("user_id", "=", query.Arg(u.ID)),
			query.Where(field, "=", query.Arg(val)),
		}

		if p.Valid {
			_, n, err := p.Resolve(ctx, s.Pool, u)

			if err != nil {
				if _, ok := err.(*PathError); !ok {
					return errors.Err(err)
				}
				return err
			}

			if err := n.IsCollaborator(ctx, s.Pool, u); err != nil {
				return err
			}
			opts[0] = query.Where("namespace_id", "=", query.Arg(n.ID))
		}

		_, ok, err := s.SelectOne(ctx, []string{field}, opts...)

		if err != nil {
			return errors.Err(err)
		}
		if ok {
			return webutil.ErrFieldExists
		}
		return nil
	}
}

func LoadResourceRelations[M database.Model](ctx context.Context, db *database.Pool, mm ...M) error {
	if len(mm) == 0 {
		return nil
	}

	ids := database.Map[M, any](mm, func(m M) any {
		param := m.Params()["namespace_id"]
		return param.Value
	})

	nn, err := NewStore(db).All(ctx, query.Where("id", "IN", query.List(ids...)))

	if err != nil {
		return errors.Err(err)
	}

	loaded := make([]database.Model, 0, len(nn))

	for _, n := range nn {
		loaded = append(loaded, n)

		for _, m := range mm {
			m.Bind(n)
		}
	}

	if err := user.Loader(db).Load(ctx, "user_id", "id", loaded...); err != nil {
		return errors.Err(err)
	}

	rels := []database.Relation{
		{From: "user_id", To: "id", Loader: user.Loader(db)},
		{From: "author_id", To: "id", Loader: user.Loader(db)},
	}

	if err := database.LoadRelations[M](ctx, mm, rels...); err != nil {
		return errors.Err(err)
	}
	return nil
}

var _ database.Model = (*Namespace)(nil)

func (n *Namespace) Primary() (string, any) { return "id", n.ID }

func (n *Namespace) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":          &n.ID,
		"user_id":     &n.UserID,
		"root_id":     &n.RootID,
		"parent_id":   &n.ParentID,
		"name":        &n.Name,
		"path":        &n.Path,
		"description": &n.Description,
		"level":       &n.Level,
		"visibility":  &n.Visibility,
		"created_at":  &n.CreatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}

	n.loaded = r.Columns
	return nil
}

func (n *Namespace) Params() database.Params {
	params := database.Params{
		"id":          database.ImmutableParam(n.ID),
		"user_id":     database.CreateOnlyParam(n.UserID),
		"root_id":     database.CreateUpdateParam(n.RootID),
		"parent_id":   database.CreateUpdateParam(n.ParentID),
		"name":        database.CreateOnlyParam(n.Name),
		"path":        database.CreateOnlyParam(n.Path),
		"description": database.CreateUpdateParam(n.Description),
		"level":       database.CreateOnlyParam(n.Level),
		"visibility":  database.CreateUpdateParam(n.Visibility),
		"created_at":  database.CreateOnlyParam(n.CreatedAt),
	}

	if len(n.loaded) > 0 {
		params.Only(n.loaded...)
	}
	return params
}

func (n *Namespace) MarshalJSON() ([]byte, error) {
	if n == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"id":                n.ID,
		"user_id":           n.UserID,
		"root_id":           n.RootID,
		"parent_id":         n.ParentID,
		"name":              n.Name,
		"path":              n.Path,
		"description":       n.Description,
		"visibility":        n.Visibility,
		"created_at":        n.CreatedAt,
		"url":               env.DJINN_API_SERVER + n.Endpoint(),
		"builds_url":        env.DJINN_API_SERVER + n.Endpoint("builds"),
		"namespaces_url":    env.DJINN_API_SERVER + n.Endpoint("namespaces"),
		"images_url":        env.DJINN_API_SERVER + n.Endpoint("images"),
		"objects_url":       env.DJINN_API_SERVER + n.Endpoint("objects"),
		"variables_url":     env.DJINN_API_SERVER + n.Endpoint("variables"),
		"keys_url":          env.DJINN_API_SERVER + n.Endpoint("keys"),
		"invites_url":       env.DJINN_API_SERVER + n.Endpoint("invites"),
		"collaborators_url": env.DJINN_API_SERVER + n.Endpoint("collaborators"),
		"webhooks_url":      env.DJINN_API_SERVER + n.Endpoint("webhooks"),
		"user":              n.User,
		"build":             n.Build,
		"parent":            n.Parent,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (n *Namespace) Bind(m database.Model) {
	switch v := m.(type) {
	case *auth.User:
		if n.UserID == v.ID {
			n.User = v
		}
	case *Namespace:
		if n.ID == v.ParentID.Elem {
			n.Parent = v
		}
	}
}

func (n *Namespace) Endpoint(elems ...string) string {
	if n.User == nil {
		return ""
	}

	if len(elems) > 0 {
		return "/n/" + n.User.Username + "/" + n.Path + "/-/" + strings.Join(elems, "/")
	}
	return "/n/" + n.User.Username + "/" + n.Path
}

func (n *Namespace) loadCollaborators(ctx context.Context, pool *database.Pool) error {
	if n.collabtab != nil {
		return nil
	}

	cc, err := NewCollaboratorStore(pool).Select(
		ctx,
		[]string{"user_id"},
		query.Where("namespace_id", "=", query.Arg(n.RootID)),
	)

	if err != nil {
		return errors.Err(err)
	}

	n.collabtab = make(map[int64]struct{})

	for _, c := range cc {
		n.collabtab[c.UserID] = struct{}{}
	}
	return nil
}

// IsCollaborator checks to see if the user is a collaborator in the current
// namespace. This returns ErrPermission if the user is not a collaborator
// or the namespace owner.
func (n *Namespace) IsCollaborator(ctx context.Context, pool *database.Pool, u *auth.User) error {
	if n.UserID == u.ID {
		return nil
	}

	if err := n.loadCollaborators(ctx, pool); err != nil {
		return errors.Err(err)
	}

	if _, ok := n.collabtab[u.ID]; !ok {
		return database.ErrPermission
	}
	return nil
}

// HasAccess checks to see if a user can access the current namespace. This
// will return database.ErrPermission if the user cannot access the namespace. A
// namespace can be accessed if,
//
// - It is public
// - It is internal, and the user is logged in
// - It is private, and the user is a collaborator in that namespace
func (n *Namespace) HasAccess(ctx context.Context, pool *database.Pool, u *auth.User) error {
	switch n.Visibility {
	case Public:
		return nil
	case Internal:
		if u.ID > 0 {
			return nil
		}
		return database.ErrPermission
	case Private:
		if err := n.loadCollaborators(ctx, pool); err != nil {
			return errors.Err(err)
		}

		if _, ok := n.collabtab[u.ID]; !ok {
			if n.UserID != u.ID {
				return database.ErrPermission
			}
		}
		return nil
	default:
		return database.ErrPermission
	}
}

type Event struct {
	dis event.Dispatcher

	Namespace *Namespace
	Action    string
}

var _ queue.Job = (*Event)(nil)

func InitEvent(dis event.Dispatcher) queue.InitFunc {
	return func(j queue.Job) {
		if ev, ok := j.(*Event); ok {
			ev.dis = dis
		}
	}
}

func (e *Event) Name() string { return "event:" + event.Namespaces.String() }

func (e *Event) Perform() error {
	if e.dis == nil {
		return event.ErrNilDispatcher
	}

	namespaceId := database.Null[int64]{
		Elem:  e.Namespace.ID,
		Valid: true,
	}

	ev := event.New(namespaceId, event.Namespaces, map[string]any{
		"namespace": e.Namespace,
		"action":    e.Action,
	})

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

type Store struct {
	*database.Store[*Namespace]
}

func Select(expr query.Expr, opts ...query.Option) query.Query {
	return query.Select(expr, append([]query.Option{query.From(table)}, opts...)...)
}

const table = "namespaces"

func Loader(pool *database.Pool) database.Loader {
	return database.ModelLoader(pool, table, func() database.Model {
		return &Namespace{}
	})
}

func NewStore(pool *database.Pool) *database.Store[*Namespace] {
	return database.NewStore[*Namespace](pool, table, func() *Namespace {
		return &Namespace{}
	})
}

const MaxDepth int64 = 20

var (
	ErrOwner      = errors.New("namespace: invalid owner")
	ErrDepth      = errors.New("namespace: cannot exceed depth of 20")
	ErrName       = errors.New("namespace: name can only contain letters and numbers")
	ErrDeleteSelf = errors.New("namespace: cannot delete self from namespace")
)

// SelectRootId returns query.Query that will return the root_id of the
// Namespace by the given id.
func SelectRootID(id int64) query.Query {
	return query.Select(
		query.Columns("root_id"),
		query.From(table),
		query.Where("id", "=", query.Arg(id)),
	)
}

func SharedWith(userId int64) query.Option {
	return func(q query.Query) query.Query {
		return query.Options(
			query.Where("user_id", "=", query.Arg(userId)),
			query.OrWhere("root_id", "IN",
				query.Select(
					query.Columns("namespace_id"),
					query.From(collaboratorTable),
					query.Where("user_id", "=", query.Arg(userId)),
				),
			),
		)(q)
	}
}

type Params struct {
	User        *auth.User
	Parent      string
	Name        string
	Description string
	Visibility  Visibility
}

func (s Store) Create(ctx context.Context, p *Params) (*Namespace, error) {
	parent, ok, err := s.Get(
		ctx,
		query.Where("user_id", "=", query.Arg(p.User.ID)),
		query.Where("path", "=", query.Arg(p.Parent)),
	)

	if err != nil {
		return nil, errors.Err(err)
	}

	if !ok && p.Parent != "" {
		return nil, database.ErrNoRows
	}

	var (
		rootId   database.Null[int64]
		parentId database.Null[int64]
		level    int64
	)

	path := p.Name

	if ok {
		rootId = parent.RootID

		parentId.Elem = parent.ID
		parentId.Valid = true

		level = parent.Level + 1

		p.Visibility = parent.Visibility
		path = strings.Join([]string{parent.Path, p.Name}, "/")
	}

	if level >= MaxDepth {
		return nil, ErrDepth
	}

	n := Namespace{
		loaded:      []string{"*"},
		UserID:      p.User.ID,
		RootID:      rootId,
		ParentID:    parentId,
		Name:        p.Name,
		Path:        path,
		Description: p.Description,
		Level:       level,
		Visibility:  p.Visibility,
		CreatedAt:   time.Now(),
		User:        p.User,
	}

	if err := s.Store.Create(ctx, &n); err != nil {
		return nil, errors.Err(err)
	}

	if !n.RootID.Valid {
		n.loaded[0] = "root_id"

		n.RootID.Elem = n.ID
		n.RootID.Valid = true

		if err := s.Store.Update(ctx, &n); err != nil {
			return nil, errors.Err(err)
		}
	}
	return &n, nil
}

func (s Store) Update(ctx context.Context, n *Namespace) error {
	parent, ok, err := s.SelectOne(
		ctx,
		[]string{"visibility"},
		query.Where("id", "=", query.Select(
			query.Columns("parent_id"),
			query.From(table),
			query.Where("id", "=", query.Arg(n.ID)),
		)),
	)

	if err != nil {
		return errors.Err(err)
	}

	if ok {
		n.Visibility = parent.Visibility
	} else {
		// Ensure we only update the visibility column.
		loaded := n.loaded
		n.loaded = []string{"visibility"}

		if err := s.Store.UpdateMany(ctx, n, query.Where("root_id", "=", query.Arg(n.ID))); err != nil {
			return errors.Err(err)
		}
		n.loaded = loaded
	}

	if err := s.Store.Update(ctx, n); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) Delete(ctx context.Context, n *Namespace) error {
	q := query.Delete(
		table,
		query.Where("id", "=", query.Arg(n.ID)),
		query.OrWhere("root_id", "=", query.Arg(n.ID)),
		query.OrWhere("parent_id", "=", query.Arg(n.ID)),
	)

	if _, err := s.Exec(ctx, q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) Index(ctx context.Context, vals url.Values, opts ...query.Option) (*database.Paginator[*Namespace], error) {
	page, _ := strconv.Atoi(vals.Get("page"))

	opts = append([]query.Option{
		database.Search("path", vals.Get("search")),
	}, opts...)

	p, err := s.Paginate(ctx, page, database.PageLimit, opts...)

	if err != nil {
		return nil, errors.Err(err)
	}

	if err := p.Load(ctx, s.Store, append(opts, query.OrderAsc("path"))...); err != nil {
		return nil, errors.Err(err)
	}
	return p, nil
}
