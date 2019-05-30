package model

import (
	"database/sql"
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

	User     *User
	Root     *Namespace
	Parent   *Namespace
	Children []*Namespace
}

type NamespaceStore struct {
	*sqlx.DB

	User      *User
	Namespace *Namespace
}

func NewNamespaceStore(db *sqlx.DB) *NamespaceStore {
	return &NamespaceStore{
		DB: db,
	}
}

func (n *Namespace) BuildStore() BuildStore {
	return BuildStore{
		DB:        n.DB,
		Namespace: n,
	}
}

func (n *Namespace) NamespaceStore() NamespaceStore {
	return NamespaceStore{
		DB:        n.DB,
		Namespace: n,
	}
}

func (n *Namespace) CascadeVisibility() error {
	if n.ID != n.RootID.Int64 {
		return nil
	}

	stmt, err := n.Prepare("UPDATE namespaces SET visibility = $1 WHERE root_id = $2")

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(n.Visibility, n.RootID)

	return errors.Err(err)
}

func (n *Namespace) Create() error {
	stmt, err := n.Prepare(`
		INSERT INTO namespaces (user_id, root_id, parent_id, name, path, description, level, visibility)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(
		n.UserID,
		n.RootID,
		n.ParentID,
		n.Name,
		n.Path,
		n.Description,
		n.Level,
		n.Visibility,
	)

	return errors.Err(row.Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt))
}

func (n *Namespace) Destroy() error {
	nn, err := n.NamespaceStore().All()

	if err != nil {
		return errors.Err(err)
	}

	for _, n := range nn {
		if err := n.Destroy(); err != nil {
			return errors.Err(err)
		}
	}

	stmt, err := n.Prepare("DELETE FROM namespaces WHERE id = $1")

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	_, err = stmt.Exec(n.ID)

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

func (n *Namespace) LoadChildren() error {
	var err error

	namespaces := NamespaceStore{
		DB:   n.DB,
		User: n.User,
	}

	n.Children, err = namespaces.ByRootID(n.ID)

	return errors.Err(err)
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

func (n Namespace) UIEndpoint() string {
	if n.User == nil {
		return ""
	}

	return "/u/" + n.User.Username + "/" + n.Path
}

func (n *Namespace) Update() error {
	stmt, err := n.Prepare(`
		UPDATE namespaces
		SET root_id = $1, name = $2, path = $3, description = $4, visibility = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(n.RootID, n.Name, n.Path, n.Description, n.Visibility, n.ID)

	return errors.Err(row.Scan(&n.UpdatedAt))
}

func (ns NamespaceStore) All() ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	query := "SELECT * FROM namespaces"
	args := []interface{}{}

	if ns.User != nil {
		query += " WHERE user_id = $1"
		args = append(args, ns.User.ID)
	}

	if ns.Namespace != nil {
		if ns.User != nil {
			query += " AND WHERE parent_id = $2"
		} else {
			query += " WHERE parent_id = $1"
		}

		args = append(args, ns.Namespace.ID)
	}

	query += " ORDER BY path ASC"

	err := ns.Select(&nn, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, n := range nn {
		n.DB = ns.DB
		n.User = ns.User
		n.Parent = ns.Namespace
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

	query := "SELECT * FROM namespaces WHERE id = $1"
	args := []interface{}{id}

	if ns.User != nil {
		query += " AND user_id = $2"
		args = append(args, ns.User.ID)
	}

	if ns.Namespace != nil {
		if ns.User != nil {
			query += " AND parent_id = $3"
		} else {
			query += " AND parent_id = $2"
		}

		args = append(args, ns.Namespace.ID)
	}

	err := ns.Get(n, query, args...)

	if err == sql.ErrNoRows {
		err = nil

		n.CreatedAt = nil
		n.UpdatedAt = nil
	}

	return n, errors.Err(err)
}

func (ns NamespaceStore) ByRootID(id int64) ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	query := "SELECT * FROM namespaces WHERE root_id = $1"
	args := []interface{}{id}

	if ns.User != nil {
		query += " AND user_id = $2"
		args = append(args, ns.User.ID)
	}

	err := ns.Select(&nn, query, args...)

	if err == sql.ErrNoRows {
		err = nil

		for _, n := range nn {
			n.CreatedAt = nil
			n.UpdatedAt = nil
		}
	}

	for _, n := range nn {
		n.DB = ns.DB
	}

	return nn, errors.Err(err)
}

func (ns NamespaceStore) In(ids ...int64) ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	if len(ids) == 0 {
		return nn, nil
	}

	query, args, err := sqlx.In("SELECT * FROM namespaces WHERE id IN (?)", ids)

	if err != nil {
		return nn, errors.Err(err)
	}

	err = ns.Select(&nn, ns.Rebind(query), args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, n := range nn {
		n.DB = ns.DB
	}

	return nn, errors.Err(err)
}

func (ns NamespaceStore) FindByPath(path string) (*Namespace, error) {
	n := &Namespace{
		model: model{
			DB: ns.DB,
		},
		User:   ns.User,
		Parent: ns.Namespace,
	}

	query := "SELECT * FROM namespaces WHERE path = $1"
	args := []interface{}{path}

	if ns.User != nil {
		query += " AND user_id = $2"
		args = append(args, ns.User.ID)
	}

	if ns.Namespace != nil {
		if ns.User != nil {
			query += " AND parent_id = $3"
		} else {
			query += " AND parent_id = $2"
		}

		args = append(args, ns.Namespace.ID)
	}

	err := ns.Get(n, query, args...)

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

func (ns NamespaceStore) Like(like string) ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	query := "SELECT * FROM namespaces WHERE path LIKE $1"
	args := []interface{}{"%" + like + "%"}

	if ns.User != nil {
		query += " AND user_id = $2"
		args = append(args, ns.User.ID)
	}

	if ns.Namespace != nil {
		if ns.User != nil {
			query += " AND parent_id = $3"
		} else {
			query += " AND parent_id = $2"
		}

		args = append(args, ns.Namespace.ID)
	}

	err := ns.Select(&nn, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, n := range nn {
		n.DB = ns.DB
		n.User = ns.User
		n.Parent = ns.Namespace
	}

	return nn, errors.Err(err)
}

func (ns NamespaceStore) LoadUsers(nn []*Namespace) error {
	if len(nn) == 0 {
		return nil
	}

	ids := make([]int64, len(nn), len(nn))

	for i, n := range nn {
		ids[i] = n.UserID
	}

	users := UserStore{
		DB: ns.DB,
	}

	uu, err := users.In(ids...)

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
