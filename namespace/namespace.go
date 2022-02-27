package namespace

import (
	"database/sql"
	"net/url"
	"strconv"
	"strings"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/queue"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type Namespace struct {
	ID          int64
	UserID      int64
	RootID      sql.NullInt64
	ParentID    sql.NullInt64
	Name        string
	Path        string
	Description string
	Level       int64
	Visibility  Visibility
	CreatedAt   time.Time

	User   *user.User
	Parent *Namespace
	Build  database.Model

	collabtab map[int64]struct{}
}

var _ database.Model = (*Namespace)(nil)

func Relations(db database.Pool) []database.RelationFunc {
	return []database.RelationFunc{
		database.Relation("user_id", "id", user.Store{Pool: db}),
	}
}

func ResourceRelations(db database.Pool) []database.RelationFunc {
	users := user.Store{Pool: db}

	return []database.RelationFunc{
		database.Relation("user_id", "id", users),
		database.Relation("author_id", "id", users),
	}
}

// Load the namespace and owner to the given models if they belong to a
// namespace.
func Load(db database.Pool, mm ...database.Model) error {
	vals := database.Values("namespace_id", mm)

	namespaces := Store{Pool: db}

	nn, err := namespaces.All(query.Where("id", "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	mm2 := make([]database.Model, 0, len(nn))

	for _, n := range nn {
		mm2 = append(mm2, n)
	}

	rel := database.Relation("user_id", "id", user.Store{Pool: db})

	if err := database.LoadRelations(mm2, rel); err != nil {
		return errors.Err(err)
	}

	database.Bind("namespace_id", "id", mm2, mm)
	return nil
}

func LoadRelations(db database.Pool, nn ...*Namespace) error {
	mm := make([]database.Model, 0, len(nn))

	for _, n := range nn {
		mm = append(mm, n)
	}

	if err := database.LoadRelations(mm, Relations(db)...); err != nil {
		return errors.Err(err)
	}
	return nil
}

// loadCollaborators loads the collaborators for the root of the namespace.
func (n *Namespace) loadCollaborators(db database.Pool) error {
	if n.collabtab != nil {
		return nil
	}

	q := query.Select(
		query.Columns("user_id"),
		query.From(collaboratorTable),
		query.Where("namespace_id", "=", query.Arg(n.RootID)),
	)

	rows, err := db.Query(q.Build(), q.Args()...)

	if err != nil {
		return errors.Err(err)
	}

	n.collabtab = make(map[int64]struct{})

	var id int64

	for rows.Next() {
		if err := rows.Scan(&id); err != nil {
			return errors.Err(err)
		}
		n.collabtab[id] = struct{}{}
	}
	return nil
}

// IsCollaborator checks to see if the user is a collaborator in the current
// namespace. This returns ErrPermission if the user is not a collaborator
// or the namespace owner.
func (n *Namespace) IsCollaborator(db database.Pool, userId int64) error {
	if err := n.loadCollaborators(db); err != nil {
		return errors.Err(err)
	}

	if _, ok := n.collabtab[userId]; !ok {
		if n.UserID != userId {
			return ErrPermission
		}
	}
	return nil
}

// HasAccess checks to see if a user can access the current namespace. This
// will return ErrPermission if the user cannot access the namespace. A
// namespace can be accessed if,
//
// - It is public
// - It is internal, and the user is logged in
// - It is private, and the user is a collaborator in that namespace
func (n *Namespace) HasAccess(db database.Pool, userId int64) error {
	switch n.Visibility {
	case Public:
		return nil
	case Internal:
		if userId > 0 {
			return nil
		}
		return ErrPermission
	case Private:
		if err := n.loadCollaborators(db); err != nil {
			return errors.Err(err)
		}

		if _, ok := n.collabtab[userId]; !ok {
			if n.UserID != userId {
				return ErrPermission
			}
		}
		return nil
	default:
		return ErrPermission
	}
}

func (n *Namespace) JSON(addr string) map[string]interface{} {
	if n == nil {
		return nil
	}

	json := map[string]interface{}{
		"id":                n.ID,
		"user_id":           n.UserID,
		"root_id":           n.RootID.Int64,
		"parent_id":         nil,
		"name":              n.Name,
		"path":              n.Path,
		"description":       n.Description,
		"visibility":        n.Visibility.String(),
		"created_at":        n.CreatedAt.Format(time.RFC3339),
		"url":               addr + n.Endpoint(),
		"builds_url":        addr + n.Endpoint("builds"),
		"namespaces_url":    addr + n.Endpoint("namespaces"),
		"images_url":        addr + n.Endpoint("images"),
		"objects_url":       addr + n.Endpoint("objects"),
		"variables_url":     addr + n.Endpoint("variables"),
		"keys_url":          addr + n.Endpoint("keys"),
		"invites_url":       addr + n.Endpoint("invites"),
		"collaborators_url": addr + n.Endpoint("collaborators"),
		"webhooks_url":      addr + n.Endpoint("webhooks"),
	}

	if n.User != nil {
		json["user"] = n.User.JSON(addr)
	}

	if n.Build != nil {
		if v := n.Build.JSON(addr); v != nil {
			json["build"] = v
		}
	}

	if n.ParentID.Valid {
		json["parent_id"] = n.ParentID.Int64

		if n.Parent != nil {
			json["parent"] = n.Parent.JSON(addr)
		}
	}
	return json
}

func (n *Namespace) Dest() []interface{} {
	return []interface{}{
		&n.ID,
		&n.UserID,
		&n.RootID,
		&n.ParentID,
		&n.Name,
		&n.Path,
		&n.Description,
		&n.Level,
		&n.Visibility,
		&n.CreatedAt,
	}
}

func (n *Namespace) Bind(m database.Model) {
	switch v := m.(type) {
	case *user.User:
		if n.UserID == v.ID {
			n.User = v
		}
	case *Namespace:
		if n.ID == v.ParentID.Int64 {
			n.Parent = v
		}
	}
}

func (n *Namespace) Endpoint(uri ...string) string {
	if n.User == nil {
		return ""
	}

	if len(uri) > 0 {
		return "/n/" + n.User.Username + "/" + n.Path + "/-/" + strings.Join(uri, "/")
	}
	return "/n/" + n.User.Username + "/" + n.Path
}

func (n *Namespace) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":          n.ID,
		"user_id":     n.UserID,
		"root_id":     n.RootID,
		"parent_id":   n.ParentID,
		"name":        n.Name,
		"path":        n.Path,
		"description": n.Description,
		"level":       n.Level,
		"visibility":  n.Visibility,
		"created_at":  n.CreatedAt,
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

	namespaceId := sql.NullInt64{
		Int64: e.Namespace.ID,
		Valid: true,
	}

	ev := event.New(namespaceId, event.Namespaces, map[string]interface{}{
		"namespace": e.Namespace.JSON(env.DJINN_API_SERVER),
		"action":    e.Action,
	})

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

type Store struct {
	database.Pool
}

var (
	_ database.Loader = (*Store)(nil)

	table = "namespaces"

	MaxDepth int64 = 20

	ErrOwner      = errors.New("invalid namespace owner")
	ErrDepth      = errors.New("namespace cannot exceed depth of 20")
	ErrName       = errors.New("namespace name can only contain letters and numbers")
	ErrPermission = errors.New("namespace permissions invalid")
	ErrDeleteSelf = errors.New("cannot delete self from namespace")
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
	UserID      int64
	Parent      string
	Name        string
	Description string
	Visibility  Visibility
}

func (s Store) Create(p Params) (*Namespace, error) {
	parent, ok, err := s.Get(
		query.Where("user_id", "=", query.Arg(p.UserID)),
		query.Where("path", "=", query.Arg(p.Parent)),
	)

	if err != nil {
		return nil, errors.Err(err)
	}

	if !ok && p.Parent != "" {
		return nil, database.ErrNotFound
	}

	var (
		rootId   sql.NullInt64
		parentId sql.NullInt64
		level    int64
	)

	path := p.Name

	if ok {
		rootId = parent.RootID
		parentId = sql.NullInt64{
			Int64: parent.ID,
			Valid: true,
		}

		level = parent.Level + 1
		path = strings.Join([]string{parent.Path, p.Name}, "/")

		p.Visibility = parent.Visibility
	}

	if level >= MaxDepth {
		return nil, ErrDepth
	}

	now := time.Now()

	q := query.Insert(
		table,
		query.Columns("user_id", "root_id", "parent_id", "name", "path", "description", "level", "visibility", "created_at"),
		query.Values(p.UserID, rootId, parentId, p.Name, path, p.Description, level, p.Visibility, now),
		query.Returning("id"),
	)

	var id int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}

	if !ok {
		rootId = sql.NullInt64{
			Int64: id,
			Valid: true,
		}

		q = query.Update(
			table,
			query.Set("root_id", query.Arg(rootId)),
			query.Where("id", "=", query.Arg(id)),
		)

		if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
			return nil, errors.Err(err)
		}
	}

	return &Namespace{
		ID:          id,
		UserID:      p.UserID,
		RootID:      rootId,
		ParentID:    parentId,
		Name:        p.Name,
		Path:        path,
		Description: p.Description,
		Level:       level,
		Visibility:  p.Visibility,
		CreatedAt:   now,
		Parent:      parent,
	}, nil
}

func (s Store) Update(id int64, p Params) error {
	parent, ok, err := s.Get(
		query.Where("id", "=", query.Select(
			query.Columns("parent_id"),
			query.From(table),
			query.Where("id", "=", query.Arg(id)),
		)),
	)

	if err != nil {
		return errors.Err(err)
	}

	if ok {
		p.Visibility = parent.Visibility
	}

	if !ok {
		q := query.Update(
			table,
			query.Set("visibility", query.Arg(p.Visibility)),
			query.Where("root_id", "=", query.Arg(id)),
		)

		if _, err = s.Exec(q.Build(), q.Args()...); err != nil {
			return errors.Err(err)
		}
	}

	q := query.Update(
		table,
		query.Set("description", query.Arg(p.Description)),
		query.Set("visibility", query.Arg(p.Visibility)),
		query.Where("id", "=", query.Arg(id)),
	)

	if _, err = s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) Delete(id int64) error {
	q := query.Delete(
		table,
		query.Where("id", "=", query.Arg(id)),
		query.OrWhere("root_id", "=", query.Arg(id)),
		query.OrWhere("parent_id", "=", query.Arg(id)),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (s Store) Get(opts ...query.Option) (*Namespace, bool, error) {
	var n Namespace

	ok, err := s.Pool.Get(table, &n, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &n, ok, nil
}

func (s Store) All(opts ...query.Option) ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	new := func() database.Model {
		n := &Namespace{}
		nn = append(nn, n)
		return n
	}

	if err := s.Pool.All(table, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return nn, nil
}

func (s Store) Paginate(page, limit int64, opts ...query.Option) (database.Paginator, error) {
	paginator, err := s.Pool.Paginate(table, page, limit, opts...)

	if err != nil {
		return paginator, errors.Err(err)
	}
	return paginator, nil
}

func (s Store) Index(vals url.Values, opts ...query.Option) ([]*Namespace, database.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		database.Search("path", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, database.PageLimit, opts...)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}

	nn, err := s.All(append(
		opts,
		query.OrderAsc("path"),
		query.Limit(paginator.Limit),
		query.Offset(paginator.Offset),
	)...)

	if err != nil {
		return nil, paginator, errors.Err(err)
	}
	return nn, paginator, nil
}

func (s Store) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	nn, err := s.All(query.Where(pk, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	loaded := make([]database.Model, 0, len(nn))

	for _, n := range nn {
		loaded = append(loaded, n)
	}

	database.Bind(fk, pk, loaded, mm)
	return nil
}
