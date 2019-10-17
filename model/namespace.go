package model

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/types"

	"github.com/andrewpillar/query"
)

type Namespace struct {
	Model

	UserID      int64            `db:"user_id"`
	RootID      sql.NullInt64    `db:"root_id"`
	ParentID    sql.NullInt64    `db:"parent_id"`
	Name        string           `db:"name"`
	Path        string           `db:"path"`
	Description string           `db:"description"`
	Level       int64            `db:"level"`
	Visibility  types.Visibility `db:"visibility"`

	User          *User
	Root          *Namespace
	Parent        *Namespace
	Children      []*Namespace
	Builds        []*Build
	LastBuild     *Build
	Collaborators map[int64]struct{}
}

type NamespaceStore struct {
	Store

	User      *User
	Namespace *Namespace
}

func namespaceToInterface(nn []*Namespace) func(i int) Interface {
	return func(i int) Interface {
		return nn[i]
	}
}

func NamespaceSharedWith(u *User) query.Option {
	return func(q query.Query) query.Query {
		if u == nil || u.IsZero() {
			return q
		}

		return query.Options(
			query.Where("user_id", "=", u.ID),
			query.OrWhereQuery("root_id", "IN",
				query.Select(
					query.Columns("namespace_id"),
					query.From(CollaboratorTable),
					query.Where("user_id", "=", u.ID),
				),
			),
		)(q)
	}
}

// Determine if the given user can add to a namespace.
func (n Namespace) CanAdd(u *User) bool {
	_, ok := n.Collaborators[u.ID]

	if !ok {
		ok = n.UserID == u.ID
	}

	return ok
}

func (n Namespace) AccessibleBy(u *User) bool {
	switch n.Visibility {
	case types.Public:
		return true
	case types.Internal:
		return !u.IsZero()
	case types.Private:
		_, ok := n.Collaborators[u.ID]

		if !ok {
			ok = n.UserID == u.ID
		}

		return ok
	default:
		return false
	}
}

func (n *Namespace) BuildStore() BuildStore {
	return BuildStore{
		Store: Store{
			DB: n.DB,
		},
		Namespace: n,
	}
}

func (n *Namespace) CollaboratorStore() CollaboratorStore {
	return CollaboratorStore{
		Store: Store{
			DB: n.DB,
		},
		Namespace: n,
	}
}

func (n *Namespace) ImageStore() ImageStore {
	return ImageStore{
		Store: Store{
			DB: n.DB,
		},
		Namespace: n,
	}
}

func (n *Namespace) InviteStore() InviteStore {
	return InviteStore{
		Store: Store{
			DB: n.DB,
		},
		Namespace: n,
	}
}

func (n *Namespace) KeyStore() KeyStore {
	return KeyStore{
		Store: Store{
			DB: n.DB,
		},
		Namespace: n,
	}
}

func (n *Namespace) LoadCollaborators() error {
	cc, err := n.CollaboratorStore().All()

	if err != nil {
		return errors.Err(err)
	}

	if n.Collaborators == nil {
		n.Collaborators = make(map[int64]struct{})
	}

	for _, c := range cc {
		n.Collaborators[c.UserID] = struct{}{}
	}

	return nil
}

func (n *Namespace) ObjectStore() ObjectStore {
	return ObjectStore{
		Store: Store{
			DB: n.DB,
		},
		Namespace: n,
	}
}

func (n *Namespace) VariableStore() VariableStore {
	return VariableStore{
		Store: Store{
			DB: n.DB,
		},
		Namespace: n,
	}
}

func (n *Namespace) NamespaceStore() NamespaceStore {
	return NamespaceStore{
		Store: Store{
			DB: n.DB,
		},
		User:      n.User,
		Namespace: n,
	}
}

func (n *Namespace) CascadeVisibility() error {
	if n.ID != n.RootID.Int64 {
		return nil
	}

	q := query.Update(
		query.Table(NamespaceTable),
		query.Set("visibility", n.Visibility),
		query.Where("root_id", "=", n.RootID),
	)

	stmt, err := n.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(q.Args()...)

	return errors.Err(err)
}

func (n *Namespace) IsZero() bool {
	return n.Model.IsZero() &&
		n.UserID == 0 &&
		!n.ParentID.Valid &&
		n.Name == "" &&
		n.Path == ""&&
		n.Description == "" &&
		n.Level == 0 &&
		n.Visibility == types.Visibility(0) &&
		n.CreatedAt == time.Time{} &&
		n.UpdatedAt == time.Time{}
}

func (n *Namespace) LoadParent() error {
	if !n.ParentID.Valid {
		n.Parent = &Namespace{}

		return nil
	}

	var err error

	namespaces := NamespaceStore{
		Store: Store{
			DB: n.DB,
		},
		User: n.User,
	}

	n.Parent, err = namespaces.Find(n.ParentID.Int64)

	return errors.Err(err)
}

func (n *Namespace) LoadParents() error {
	if !n.ParentID.Valid {
		return nil
	}

	if err := n.LoadParent(); err != nil {
		return errors.Err(err)
	}

	if n.Parent.IsZero() {
		return nil
	}

	return errors.Err(n.Parent.LoadParents())
}

func (n *Namespace) LoadUser() error {
	var err error

	users := UserStore{
		Store: Store{
			DB: n.DB,
		},
	}

	n.User, err = users.Find(n.UserID)

	return errors.Err(err)
}

func (n Namespace) UIEndpoint(uri ...string) string {
	if n.User == nil {
		return ""
	}

	endpoint := fmt.Sprintf("/n/%s/%s", n.User.Username, n.Path)

	if len(uri) > 0 {
		endpoint = fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
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

func (s NamespaceStore) All(opts ...query.Option) ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	opts = append([]query.Option{ForParent(s.Namespace)}, opts...)

	err := s.Store.All(&nn, NamespaceTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, n := range nn {
		n.DB = s.DB
	}

	return nn, errors.Err(err)
}

func (s NamespaceStore) Paginate(page int64, opts ...query.Option) (Paginator, error) {
	paginator, err := s.Store.Paginate(NamespaceTable, page, opts...)

	return paginator, errors.Err(err)
}

func (s NamespaceStore) Create(nn ...*Namespace) error {
	models := interfaceSlice(len(nn), namespaceToInterface(nn))

	return errors.Err(s.Store.Create(NamespaceTable, models...))
}

func (s NamespaceStore) Delete(nn ...*Namespace) error {
	ids := make([]interface{}, len(nn), len(nn))

	for i, n := range nn {
		ids[i] = n.ID
	}

	q := query.Delete(
		query.From(NamespaceTable),
		query.Where("root_id", "IN", ids...),
	)

	stmt, err := s.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(q.Args()...)

	return errors.Err(err)
}

func (s NamespaceStore) findBy(col string, val interface{}) (*Namespace, error) {
	n := &Namespace{
		Model: Model{
			DB: s.DB,
		},
		User:   s.User,
		Parent: s.Namespace,
	}

	q := query.Select(
		query.Columns("*"),
		query.From(NamespaceTable),
		query.Where(col, "=", val),
		ForUser(s.User),
		ForParent(s.Namespace),
	)

	err := s.Get(n, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return n, errors.Err(err)
}

func (s NamespaceStore) Find(id int64) (*Namespace, error) {
	n, err := s.findBy("id", id)

	return n, errors.Err(err)
}

func (s NamespaceStore) FindByPath(path string) (*Namespace, error) {
	n, err := s.findBy("path", path)

	return n, errors.Err(err)
}

func (s NamespaceStore) FindRoot(id int64) (*Namespace, error) {
	n := &Namespace{
		Model: Model{
			DB: s.DB,
		},
	}

	rootq := query.Select(
		query.Columns("root_id"),
		query.From(NamespaceTable),
		query.Where("id", "=", id),
	)

	q := query.Select(
		query.Columns("*"),
		query.From(NamespaceTable),
		query.WhereQuery("root_id", "=", rootq),
		query.WhereQuery("id", "=", rootq),
	)

	stmt, err := s.Preparex(q.Build())

	if err != nil {
		return n, errors.Err(err)
	}

	defer stmt.Close()

	err = stmt.Get(n, q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return n, errors.Err(err)
}

func (s NamespaceStore) Load(ids []interface{}, load func(i int, n *Namespace)) error {
	if len(ids) == 0 {
		return nil
	}

	nn, err := s.All(query.Where("id", "IN", ids...))

	if err != nil {
		return errors.Err(err)
	}

	for i := range ids {
		for _, n := range nn {
			load(i, n)
		}
	}

	return nil
}

func (s NamespaceStore) LoadLastBuild(nn []*Namespace) error {
	if len(nn) == 0 {
		return nil
	}

	models := interfaceSlice(len(nn), namespaceToInterface(nn))

	builds := BuildStore{
		Store: s.Store,
	}

	bb, err := builds.All(query.Where("namespace_id", "IN", mapKey("id", models)...))

	if err != nil {
		return errors.Err(err)
	}

	if err := builds.LoadUsers(bb); err != nil {
		return errors.Err(err)
	}

	for _, n := range nn {
		for _, b := range bb {
			if n.ID == b.NamespaceID.Int64 {
				n.LastBuild = b
			}
		}
	}

	return errors.Err(err)
}

func (s NamespaceStore) loadUser(nn []*Namespace) func(i int, u *User) {
	return func(i int, u *User) {
		n := nn[i]

		if n.UserID == u.ID {
			n.User = u
		}
	}
}

func (s NamespaceStore) LoadUsers(nn []*Namespace) error {
	if len(nn) == 0 {
		return nil
	}

	models := interfaceSlice(len(nn), namespaceToInterface(nn))

	users := UserStore{
		Store: s.Store,
	}

	err := users.Load(mapKey("user_id", models), s.loadUser(nn))

	return errors.Err(err)
}

func (s NamespaceStore) New() *Namespace {
	n := &Namespace{
		Model: Model{
			DB: s.DB,
		},
		User:   s.User,
		Parent: s.Namespace,
	}

	if s.User != nil {
		n.UserID = s.User.ID
	}

	if s.Namespace != nil {
		n.ParentID = sql.NullInt64{
			Int64: s.Namespace.ID,
			Valid: true,
		}
	}

	return n
}

func (s NamespaceStore) Update(nn ...*Namespace) error {
	models := interfaceSlice(len(nn), namespaceToInterface(nn))

	return errors.Err(s.Store.Update(NamespaceTable, models...))
}
