package namespace

import (
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type Namespace struct {
	ID          int64
	UserID      int64         `db:"user_id"`
	RootID      sql.NullInt64 `db:"root_id"`
	ParentID    sql.NullInt64 `db:"parent_id"`
	Name        string        `db:"name"`
	Path        string        `db:"path"`
	Description string        `db:"description"`
	Level       int64         `db:"level"`
	Visibility  Visibility    `db:"visibility"`
	CreatedAt   time.Time     `db:"created_at"`

	User          *user.User         `db:"-"`
	Parent        *Namespace         `db:"-"`
	Build         model.Model        `db:"-"`
	Collaborators map[int64]struct{} `db:"-"`
}

type Store struct {
	model.Store

	User      *user.User
	Namespace *Namespace
}

var (
	_ model.Model  = (*Namespace)(nil)
	_ model.Loader = (*Store)(nil)
	_ model.Binder = (*Store)(nil)

	table  = "namespaces"
	rename = regexp.MustCompile("^[a-zA-Z0-9]+$")

	MaxDepth int64 = 20
	ErrDepth       = errors.New("namespace cannot exceed depth of 20")
	ErrName        = errors.New("namespace name can only contain letters and numbers")
	ErrPermission  = errors.New("permission denied")
)

func NewStore(db *sqlx.DB, mm ...model.Model) Store {
	s := Store{
		Store: model.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func Model(nn []*Namespace) func(int) model.Model {
	return func(i int) model.Model {
		return nn[i]
	}
}

func SelectRootID(id int64) query.Query {
	return query.Select(
		query.Columns("root_id"),
		query.From(table),
		query.Where("id", "=", id),
	)
}

func SharedWith(m model.Model) query.Option {
	return func(q query.Query) query.Query {
		if m == nil || m.IsZero() {
			return q
		}
		_, id := m.Primary()
		return query.Options(
			query.Where("user_id", "=", id),
			query.OrWhereQuery("root_id", "IN",
				query.Select(
					query.Columns("namespace_id"),
					query.From(collaboratorTable),
					query.Where("user_id", "=", id),
				),
			),
		)(q)
	}
}

func (n Namespace) AccessibleBy(m model.Model) bool {
	_, id := m.Primary()

	switch n.Visibility {
	case Public:
		return true
	case Internal:
		return !m.IsZero()
	case Private:
		_, ok := n.Collaborators[id]

		if !ok {
			ok = n.UserID == id
		}
		return ok
	default:
		return false
	}
}

func (n Namespace) CanAdd(m model.Model) bool {
	if _, ok := m.(*user.User); !ok {
		return false
	}

	_, id := m.Primary()
	_, ok := n.Collaborators[id]

	if !ok {
		ok = n.UserID == id
	}
	return ok
}

// CascadeVisibility applies the visibility of the root namespace to all child
// namespaces. This will have no effect if called on a child namespace.
func (n *Namespace) CascadeVisibility(db *sqlx.DB) error {
	if n.ID != n.RootID.Int64 {
		return nil
	}

	q := query.Update(
		query.Table(table),
		query.Set("visibility", n.Visibility),
		query.Where("root_id", "=", n.RootID),
	)

	stmt, err := db.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(q.Args()...)
	return errors.Err(err)
}

func (n *Namespace) IsZero() bool {
	return n == nil || n.ID == 0 &&
		n.UserID == 0 &&
		!n.ParentID.Valid &&
		n.Name == "" &&
		n.Path == ""&&
		n.Description == "" &&
		n.Level == 0 &&
		n.Visibility == Visibility(0) &&
		n.CreatedAt == time.Time{}
}

func (n *Namespace) LoadCollaborators(cc []*Collaborator) {
	if n.Collaborators == nil {
		n.Collaborators = make(map[int64]struct{})
	}

	for _, c := range cc {
		if c.NamespaceID != n.ID {
			continue
		}
		n.Collaborators[c.NamespaceID] = struct{}{}
	}
}

func (n *Namespace) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			n.User = m.(*user.User)
		case *Namespace:
			n.Parent = m.(*Namespace)
		}
	}
}

func (n Namespace) Kind() string { return "namespace" }

func (n *Namespace) SetPrimary(id int64) {
	n.ID = id
}

func (n Namespace) Primary() (string, int64) {
	return "id", n.ID
}

func (n Namespace) Endpoint(uri ...string) string {
	if n.User == nil || n.User.IsZero() {
		return ""
	}

	username, _ := n.User.Values()["username"]
	endpoint := fmt.Sprintf("/n/%s/%s", username, n.Path)

	if len(uri) > 0 {
		return fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}
	return endpoint
}

func (n Namespace) Values() map[string]interface{} {
	return map[string]interface{}{
		"user_id":     n.UserID,
		"root_id":     n.RootID,
		"parent_id":   n.ParentID,
		"name":        n.Name,
		"path":        n.Path,
		"description": n.Description,
		"level":       n.Level,
		"visibility":  n.Visibility,
	}
}

func (s *Store) Bind(mm ...model.Model) {
	for _, m := range mm {
		switch m.(type) {
		case *user.User:
			s.User = m.(*user.User)
		case *Namespace:
			s.Namespace = m.(*Namespace)
		}
	}
}

func (s Store) Create(nn ...*Namespace) error {
	models := model.Slice(len(nn), Model(nn))
	return errors.Err(s.Store.Create(table, models...))
}

func (s Store) Update(nn ...*Namespace) error {
	models := model.Slice(len(nn), Model(nn))
	return errors.Err(s.Store.Update(table, models...))
}

func (s Store) Delete(nn ...*Namespace) error {
	models := model.Slice(len(nn), Model(nn))

	ids := model.MapKey("id", models)

	q := query.Delete(query.From(table), query.Where("root_id", "IN", ids...))

	stmt, err := s.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(q.Args()...)
	return errors.Err(err)
}

func (s Store) Paginate(page int64, opts ...query.Option) (model.Paginator, error) {
	paginator, err := s.Store.Paginate(table, page, opts...)
	return paginator, errors.Err(err)
}

func (s Store) New() *Namespace {
	n := &Namespace{
		User:   s.User,
		Parent: s.Namespace,
	}

	if s.User != nil {
		_, id := s.User.Primary()
		n.UserID = id
	}

	if s.Namespace != nil {
		_, id := s.Namespace.Primary()
		n.ParentID = sql.NullInt64{
			Int64: id,
			Valid: true,
		}
	}
	return n
}

func (s Store) All(opts ...query.Option) ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	opts = append([]query.Option{
		WhereCollaborator(s.User),
		model.Where(s.Namespace, "parent_id"),
	}, opts...)

	err := s.Store.All(&nn, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, n := range nn {
		n.User = s.User
	}
	return nn, errors.Err(err)
}

func (s Store) Index(vals url.Values, opts ...query.Option) ([]*Namespace, model.Paginator, error) {
	page, err := strconv.ParseInt(vals.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	opts = append([]query.Option{
		model.Search("path", vals.Get("search")),
	}, opts...)

	paginator, err := s.Paginate(page, opts...)

	if err != nil {
		return []*Namespace{}, paginator, errors.Err(err)
	}

	nn, err := s.All(append(
		opts,
		query.OrderAsc("path"),
		query.Limit(model.PageLimit),
		query.Offset(paginator.Offset),
	)...)
	return nn, paginator, errors.Err(err)
}

func (s Store) Get(opts ...query.Option) (*Namespace, error) {
	n := &Namespace{
		User:   s.User,
		Parent: s.Namespace,
	}

	opts = append([]query.Option{
		WhereCollaborator(s.User),
		model.Where(s.Namespace, "parent_id"),
	}, opts...)

	err := s.Store.Get(n, table, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return n, errors.Err(err)
}

// GetByPath returns the namespace by the given path. If it does not exist
// then it will be created, including all grandparents, and then returned.
// The namespace for a given user will be returned if the path contains
// an '@'.
func (s Store) GetByPath(path string) (*Namespace, error) {
	users := user.NewStore(s.DB)

	if strings.Contains(path, "@") {
		parts := strings.Split(path, "@")

		username := parts[1]
		path = parts[0]

		u, err := users.Get(query.Where("username", "=", username))

		if err != nil {
			return &Namespace{}, errors.Err(err)
		}

		if !u.IsZero() {
			s.User = u
		}
	}

	n, err := s.Get(query.Where("path", "=", path))

	if err != nil {
		return n, errors.Err(err)
	}

	if !n.IsZero() {
		cc, err := NewCollaboratorStore(s.DB, n).All()

		if err != nil {
			return n, errors.Err(err)
		}
		n.LoadCollaborators(cc)
		return n, nil
	}

	p := &Namespace{}

	parts := strings.Split(path, "/")

	for i, name := range parts {
		if p.Level + 1 > MaxDepth {
			break
		}

		if !rename.Match([]byte(name)) {
			if i == 0 && len(parts) == 1 {
				return n, ErrName
			}
			break
		}

		n = s.New()
		n.Name = name
		n.Path = name
		n.Level = p.Level + 1
		n.Visibility = Private

		if !p.IsZero() {
			n.RootID = p.RootID
			n.ParentID = sql.NullInt64{
				Int64: p.ID,
				Valid: true,
			}
			n.Path = strings.Join([]string{p.Path, n.Name}, "/")
		}

		if err := s.Create(n); err != nil {
			return n, errors.Err(err)
		}

		if p.IsZero() {
			n.RootID = sql.NullInt64{
				Int64: n.ID,
				Valid: true,
			}

			if err := s.Update(n); err != nil {
				return n, errors.Err(err)
			}
		}

		p = n
	}
	return n, nil
}

func (s Store) Load(key string, vals []interface{}, load model.LoaderFunc) error {
	nn, err := s.All(query.Where(key, "IN", vals...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range vals {
		for _, n := range nn {
			load(i, n)
		}
	}
	return nil
}
