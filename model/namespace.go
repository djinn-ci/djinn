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

	"github.com/jmoiron/sqlx"
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
	*sqlx.DB

	User      *User
	Namespace *Namespace
}

func (n Namespace) AccessibleBy(u *User, a Action) bool {
	switch n.Visibility {
	case types.Public:
		return true
	case types.Internal:
		return u != nil && !u.IsZero()
	case types.Private:
		return u != nil && u.ID == n.UserID
	default:
		return false
	}
}

func (n *Namespace) BuildStore() BuildStore {
	return BuildStore{
		DB:        n.DB,
		Namespace: n,
	}
}

func (n *Namespace) KeyStore() KeyStore {
	return KeyStore{
		DB:        n.DB,
		Namespace: n,
	}
}

func (n *Namespace) ObjectStore() ObjectStore {
	return ObjectStore{
		DB:        n.DB,
		Namespace: n,
	}
}

func (n *Namespace) VariableStore() VariableStore {
	return VariableStore{
		DB:        n.DB,
		Namespace: n,
	}
}

func (n *Namespace) NamespaceStore() NamespaceStore {
	return NamespaceStore{
		DB:        n.DB,
		User:      n.User,
		Namespace: n,
	}
}

func (n *Namespace) CascadeVisibility() error {
	if n.ID != n.RootID.Int64 {
		return nil
	}

	q := query.Update(
		query.Table("namespaces"),
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

func (n *Namespace) Create() error {
	q := query.Insert(
		query.Table("namespaces"),
		query.Columns("user_id", "root_id", "parent_id", "name", "path", "description", "level", "visibility"),
		query.Values(n.UserID, n.RootID, n.ParentID, n.Name, n.Path, n.Description, n.Level, n.Visibility),
		query.Returning("id", "created_at", "updated_at"),
	)

	stmt, err := n.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt))
}

func (n *Namespace) Destroy() error {
	nn := make([]*Namespace, 0)

	q := query.Select(
		query.Columns("id"),
		query.Table("namespaces"),
		query.WhereEq("root_id", n.ID),
	)

	err := n.Select(&nn, q.Build(), q.Args()...)

	opts := []query.Option{
		query.Table("namespaces"),
	}

	if len(nn) > 0 {
		ids := make([]interface{}, len(nn))

		for i, n := range nn {
			ids[i] = n.ID
		}

		opts = append(opts, query.Or(query.WhereEq("id", n.ID), query.WhereIn("id", ids...)))
	} else {
		opts = append(opts, query.WhereEq("id", n.ID))
	}

	q = query.Delete(opts...)

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
		DB:   n.DB,
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
		DB: n.DB,
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

func (n *Namespace) Update() error {
	q := query.Update(
		query.Table("namespaces"),
		query.Set("root_id", n.RootID),
		query.Set("name", n.Name),
		query.Set("path", n.Path),
		query.Set("description", n.Description),
		query.Set("visibility", n.Visibility),
		query.SetRaw("updated_at", "NOW()"),
		query.WhereEq("id", n.ID),
		query.Returning("updated_at"),
	)

	stmt, err := n.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&n.UpdatedAt))
}

func (ns NamespaceStore) All(opts ...query.Option) ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	opts = append([]query.Option{query.Columns("*")}, opts...)
	opts = append(opts, ForParent(ns.Namespace), query.Table("namespaces"))

	q := query.Select(opts...)

	err := ns.Select(&nn, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, n := range nn {
		n.DB = ns.DB
	}

	return nn, errors.Err(err)
}

func (ns NamespaceStore) findBy(col string, val interface{}) (*Namespace, error) {
	n := &Namespace{
		Model: Model{
			DB: ns.DB,
		},
		User:   ns.User,
		Parent: ns.Namespace,
	}

	q := query.Select(
		query.Columns("*"),
		query.Table("namespaces"),
		query.WhereEq(col, val),
		ForUser(ns.User),
		ForParent(ns.Namespace),
	)

	err := ns.Get(n, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	return n, errors.Err(err)
}

func (ns NamespaceStore) Find(id int64) (*Namespace, error) {
	n, err := ns.findBy("id", id)

	return n, errors.Err(err)
}

func (ns NamespaceStore) FindByPath(path string) (*Namespace, error) {
	n, err := ns.findBy("path", path)

	return n, errors.Err(err)
}

func (ns NamespaceStore) FindOrCreate(path string) (*Namespace, error) {
	n, err := ns.FindByPath(path)

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

		n = ns.New()
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

		if err := n.Create(); err != nil {
			return n, errors.Err(err)
		}

		if parent.IsZero() {
			n.RootID = sql.NullInt64{
				Int64: n.ID,
				Valid: true,
			}

			if err := n.Update(); err != nil {
				return n, errors.Err(err)
			}
		}

		parent = n
	}

	n.User = ns.User

	return n, nil
}

func (ns NamespaceStore) Index(opts ...query.Option) ([]*Namespace, error) {
	nn, err := ns.All(opts...)

	if err := ns.LoadUsers(nn); err != nil {
		return nn, errors.Err(err)
	}

	if err := ns.LoadLastBuild(nn); err != nil {
		return nn, errors.Err(err)
	}

	return nn, errors.Err(err)
}

func (ns NamespaceStore) LoadLastBuild(nn []*Namespace) error {
	if len(nn) == 0 {
		return nil
	}

	ids := make([]interface{}, len(nn), len(nn))

	for i, n := range nn {
		ids[i] = n.ID
	}

	builds := BuildStore{
		DB: ns.DB,
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

func (ns NamespaceStore) LoadUsers(nn []*Namespace) error {
	if len(nn) == 0 {
		return nil
	}

	ids := make([]interface{}, len(nn), len(nn))

	for i, n := range nn {
		ids[i] = n.UserID
	}

	users := UserStore{
		DB: ns.DB,
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

func (ns NamespaceStore) New() *Namespace {
	n := &Namespace{
		Model: Model{
			DB: ns.DB,
		},
		User:   ns.User,
		Parent: ns.Namespace,
	}

	if ns.User != nil {
		n.UserID = ns.User.ID
	}

	if ns.Namespace != nil {
		n.ParentID = sql.NullInt64{
			Int64: ns.Namespace.ID,
			Valid: true,
		}
	}

	return n
}
