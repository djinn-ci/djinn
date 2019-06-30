package model

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/andrewpillar/thrall/errors"

	"github.com/jmoiron/sqlx"
)

type Namespace struct {
	model

	UserID      int64         `db:"user_id"`
	RootID      sql.NullInt64 `db:"root_id"`
	ParentID    sql.NullInt64 `db:"parent_id"`
	Name        string        `db:"name"`
	Path        string        `db:"path"`
	Description string        `db:"description"`
	Level       int64         `db:"level"`
	Visibility  Visibility    `db:"visibility"`

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

func (n *Namespace) BuildStore() BuildStore {
	return BuildStore{
		DB:        n.DB,
		User:      n.User,
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

	q := Update(
		Table("namespaces"),
		Set("visibility", n.Visibility),
		WhereEq("root_id", n.RootID),
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
	q := Insert(
		Table("namespaces"),
		Columns("user_id", "root_id", "parent_id", "name", "path", "description", "level", "visibility"),
		Values(n.UserID, n.RootID, n.ParentID, n.Name, n.Path, n.Description, n.Level, n.Visibility),
		Returning("id", "created_at", "updated_at"),
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
	nn, err := n.NamespaceStore().All(Columns("id"))

	if err != nil {
		return errors.Err(err)
	}

	opts := []Option{
		Table("namespaces"),
		WhereEq("id", n.ID),
	}

	if len(nn) > 0 {
		ids := make([]int64, len(nn))

		for i, n := range nn {
			ids[i] = n.ID
		}

		opts = append(opts, WhereIn("id", ids))
	}

	q := Delete(opts...)

	stmt, err := n.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(q.Args()...)

	return errors.Err(err)
}

func (n *Namespace) IsZero() bool {
	return n.ID == 0 &&
           n.UserID == 0 &&
           !n.ParentID.Valid &&
           n.Name == "" &&
           n.Path == "" &&
           n.Description == "" &&
           n.Level == 0 &&
           n.Visibility == Visibility(0) &&
           n.CreatedAt == nil &&
           n.UpdatedAt == nil
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

	endpoint := fmt.Sprintf("/u/%s/%s", n.User.Username, n.Path)

	if len(uri) > 0 {
		endpoint = fmt.Sprintf("%s/%s", endpoint, strings.Join(uri, "/"))
	}

	return endpoint
}

func (n *Namespace) Update() error {
	q := Update(
		Table("namespaces"),
		Set("root_id", n.RootID),
		Set("name", n.Name),
		Set("path", n.Path),
		Set("description", n.Description),
		Set("visibility", n.Visibility),
		SetRaw("updated_at", "NOW()"),
		Returning("updated_at"),
	)

	stmt, err := n.Prepare(q.Build())

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(q.Args()...)

	return errors.Err(row.Scan(&n.UpdatedAt))
}

func (ns NamespaceStore) All(opts ...Option) ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	opts = append([]Option{Columns("*")}, opts...)

	q := Select(append(opts, ForParent(ns.Namespace), Table("namespaces"))...)

	err := ns.Select(&nn, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, n := range nn {
		n.DB = ns.DB
	}

	return nn, errors.Err(err)
}

func (ns NamespaceStore) Find(id int64) (*Namespace, error) {
	n := &Namespace{
		model: model{
			DB: ns.DB,
		},
		User:   ns.User,
		Parent: ns.Namespace,
	}

	q := Select(
		Columns("*"),
		Table("namespaces"),
		WhereEq("id", id),
		ForUser(ns.User),
		ForParent(ns.Namespace),
	)

	err := ns.Get(n, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil

		n.CreatedAt = nil
		n.UpdatedAt = nil
	}

	return n, errors.Err(err)
}

func (ns NamespaceStore) FindByPath(path string) (*Namespace, error) {
	n := &Namespace{
		model: model{
			DB: ns.DB,
		},
		User:   ns.User,
		Parent: ns.Namespace,
	}

	q := Select(
		Columns("*"),
		Table("namespaces"),
		WhereEq("path", path),
		ForUser(ns.User),
		ForParent(ns.Namespace),
	)

	err := ns.Get(n, q.Build(), q.Args()...)

	if err == sql.ErrNoRows {
		err = nil

		n.CreatedAt = nil
		n.UpdatedAt = nil
	}

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
		if parent.Level + 1 > 20 {
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

func (ns NamespaceStore) Index(opts ...Option) ([]*Namespace, error) {
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

	bb, err := builds.All(WhereIn("namespace_id", ids...))

	if err != nil {
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

	uu, err := users.All(WhereIn("id", ids...))

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
		model: model{
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
