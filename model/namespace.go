package model

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model/query"
	"github.com/andrewpillar/thrall/model/types"
)

var NamespaceMaxDepth int64 = 20

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

	User      *User
	Root      *Namespace
	Parent    *Namespace
	Children  []*Namespace
	Builds    []*Build
	LastBuild *Build
}

type NamespaceStore struct {
	Store

	User      *User
	Namespace *Namespace
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
		query.WhereEq("root_id", n.RootID),
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

	opts = append(opts, ForParent(s.Namespace), ForUser(s.User), query.Table(NamespaceTable))

	err := s.Store.All(&nn, NamespaceTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, n := range nn {
		n.DB = s.DB
	}

	return nn, errors.Err(err)
}

func (s NamespaceStore) Create(nn ...*Namespace) error {
	return errors.Err(s.Store.Create(NamespaceTable, s.interfaceSlice(nn...)...))
}

func (s NamespaceStore) Delete(nn ...*Namespace) error {
	rootIds := make([]interface{}, len(nn), len(nn))

	for i, n := range nn {
		rootIds[i] = n.ID
	}

	q := query.Select(
		query.Columns("id"),
		query.Table(NamespaceTable),
		query.WhereIn("root_id", rootIds...),
	)

	children := make([]*Namespace, 0)

	err := s.Select(&children, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	if err != nil {
		return errors.Err(err)
	}

	opts := []query.Option{
		query.Table(NamespaceTable),
	}

	if len(children) > 0 {
		ids := make([]interface{}, len(children), len(children))

		for i, n := range children {
			ids[i] = n.ID
		}

		opts = append(opts, query.Or(query.WhereIn("id", rootIds...), query.WhereIn("id", ids...)))
	} else {
		opts = append(opts, query.WhereIn("id", rootIds...))
	}

	q = query.Delete(opts...)

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
		query.Table(NamespaceTable),
		query.WhereEq(col, val),
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

func (s NamespaceStore) FindOrCreate(path string) (*Namespace, error) {
	n, err := s.FindByPath(path)

	if err != nil {
		return n, errors.Err(err)
	}

	if !n.IsZero() {
		return n, nil
	}

	parent := &Namespace{}
	parts := strings.Split(path, "/")

	for _, name := range parts {
		if parent.Level + 1 > NamespaceMaxDepth {
			break
		}

		if matched, err := regexp.Match("^[a-zA-Z0-9]+$", []byte(name)); !matched || err != nil {
			break
		}

		n = s.New()
		n.Name = name
		n.Path = name
		n.Level = parent.Level + 1

		if !parent.IsZero() {
			n.RootID = parent.RootID
			n.ParentID = sql.NullInt64{
				Int64: parent.ID,
				Valid: true,
			}

			n.Path = strings.Join([]string{parent.Path, n.Name}, "/")
		}

		if err := s.Create(n); err != nil {
			return n, errors.Err(err)
		}

		if parent.IsZero() {
			n.RootID = sql.NullInt64{
				Int64: n.ID,
				Valid: true,
			}

			if err := s.Update(n); err != nil {
				return n, errors.Err(err)
			}
		}

		parent = n
	}

	n.User = s.User

	return n, nil
}

func (s NamespaceStore) Index(opts ...query.Option) ([]*Namespace, error) {
	nn, err := s.All(opts...)

	if err := s.LoadUsers(nn); err != nil {
		return nn, errors.Err(err)
	}

	if err := s.LoadLastBuild(nn); err != nil {
		return nn, errors.Err(err)
	}

	return nn, errors.Err(err)
}

func (s NamespaceStore) interfaceSlice(nn ...*Namespace) []Interface {
	ii := make([]Interface, len(nn), len(nn))

	for i, n := range nn {
		ii[i] = n
	}

	return ii
}

func (s NamespaceStore) Load(ids []interface{}, load func(i int, n *Namespace)) error {
	if len(ids) == 0 {
		return nil
	}

	nn, err := s.All(query.WhereIn("id", ids...))

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

	ids := make([]interface{}, len(nn), len(nn))

	for i, n := range nn {
		ids[i] = n.ID
	}

	builds := BuildStore{
		Store: s.Store,
	}

	bb, err := builds.All(query.WhereIn("namespace_id", ids...))

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

func (s NamespaceStore) LoadUsers(nn []*Namespace) error {
	if len(nn) == 0 {
		return nil
	}

	ids := make([]interface{}, len(nn), len(nn))

	for i, n := range nn {
		ids[i] = n.UserID
	}

	users := UserStore{
		Store: s.Store,
	}

	uu, err := users.All(query.WhereIn("id", ids...))

	if err != nil {
		return errors.Err(err)
	}

	for _, n := range nn {
		for _, u := range uu {
			if n.UserID == u.ID {
				n.User = u
			}
		}
	}

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
	return errors.Err(s.Store.Update(NamespaceTable, s.interfaceSlice(nn...)...))
}
