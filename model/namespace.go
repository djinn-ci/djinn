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
	ParentID    sql.NullInt64 `db:"parent_id"`
	Name        string        `db:"name"`
	Path        string        `db:"path"`
	Description string        `db:"description"`
	Level       int64         `db:"level"`
	Visibility  Visibility    `db:"visibility"`

	User     *User
	Parent   *Namespace
	Children []*Namespace
}

type NamespaceStore struct {
	*Store

	user      *User
	namespace *Namespace
}

func NewNamespaceStore(s *Store) *NamespaceStore {
	return &NamespaceStore{
		Store: s,
	}
}

func (ns NamespaceStore) New() *Namespace {
	n := &Namespace{
		model: model{
			DB: ns.DB,
		},
	}

	if ns.user != nil {
		n.UserID = ns.user.ID
		n.User = ns.user
	}

	if ns.namespace != nil {
		n.ParentID = sql.NullInt64{
			Int64: ns.namespace.ID,
			Valid: true,
		}
		n.Parent = ns.namespace
	}

	return n
}

func (ns NamespaceStore) Find(id int64) (*Namespace, error) {
	n := &Namespace{
		model: model{
			DB: ns.DB,
		},
	}

	query := "SELECT * FROM namespaces WHERE id = $1"
	args := []interface{}{id}

	if ns.user != nil {
		query += " AND user_id = $2"
		args = append(args, ns.user.ID)

		n.User = ns.user
	}

	if ns.namespace != nil {
		if ns.user != nil {
			query += " AND parent_id = $3"
		} else {
			query += " AND parent_id = $2"
		}

		args = append(args, ns.namespace.ID)

		n.Parent = ns.namespace
	}

	err := ns.Get(n, query, args...)

	if err == sql.ErrNoRows {
		err = nil

		n.CreatedAt = nil
		n.UpdatedAt = nil
	}

	return n, errors.Err(err)
}

func (ns NamespaceStore) GetByParentID(id int64) ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	query := "SELECT * FROM namespaces WHERE parent_id = $1"
	args := []interface{}{id}

	if ns.user != nil {
		query += " AND user_id = $2"
		args = append(args, ns.user.ID)
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
	}

	query := "SELECT * FROM namespaces WHERE path = $1"
	args := []interface{}{path}

	if ns.user != nil {
		query += " AND user_id = $2"
		args = append(args, ns.user.ID)

		n.User = ns.user
	}

	if ns.namespace != nil {
		if ns.user != nil {
			query += " AND parent_id = $3"
		} else {
			query += " AND parent_id = $2"
		}

		args = append(args, ns.namespace.ID)

		n.Parent = ns.namespace
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
			n.ParentID = sql.NullInt64{
				Int64: parent.ID,
				Valid: true,
			}

			n.Path = strings.Join([]string{parent.Path, n.Name}, "/")
		}

		if err := n.Create(); err != nil {
			return n, errors.Err(err)
		}

		parent = n
	}

	if ns.user != nil {
		n.User = ns.user
	}

	return n, nil
}

func (ns NamespaceStore) All() ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	query := "SELECT * FROM namespaces"
	args := []interface{}{}

	if ns.user != nil {
		query += " WHERE user_id = $1"
		args = append(args, ns.user.ID)
	}

	if ns.namespace != nil {
		if ns.user != nil {
			query += " AND WHERE parent_id = $2"
		} else {
			query += " WHERE parent_id = $1"
		}

		args = append(args, ns.namespace.ID)
	}

	query += " ORDER BY path ASC"

	err := ns.Select(&nn, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, n := range nn {
		n.DB = ns.DB

		if ns.user != nil {
			n.User = ns.user
		}

		if ns.namespace != nil {
			n.Parent = ns.namespace
		}
	}

	return nn, errors.Err(err)
}

func (ns NamespaceStore) Like(like string) ([]*Namespace, error) {
	nn := make([]*Namespace, 0)

	query := "SELECT * FROM namespaces WHERE path LIKE $1"
	args := []interface{}{"%" + like + "%"}

	if ns.user != nil {
		query += " AND user_id = $2"
		args = append(args, ns.user.ID)
	}

	if ns.namespace != nil {
		if ns.user != nil {
			query += " AND parent_id = $3"
		} else {
			query += " AND parent_id = $2"
		}

		args = append(args, ns.namespace.ID)
	}

	err := ns.Select(&nn, query, args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, n := range nn {
		n.DB = ns.DB

		if ns.user != nil {
			n.User = ns.user
		}

		if ns.namespace != nil {
			n.Parent = ns.namespace
		}
	}

	return nn, errors.Err(err)
}

func (ns NamespaceStore) LoadRelations(nn []*Namespace) error {
	if len(nn) == 0 {
		return nil
	}

	userIds := make([]int64, len(nn), len(nn))

	for i, n := range nn {
		userIds[i] = n.UserID
	}

	query, args, err := sqlx.In("SELECT * FROM users WHERE id IN (?)", userIds)

	if err != nil {
		return errors.Err(err)
	}

	uu := make([]*User, 0)

	err = ns.Select(&uu, ns.Rebind(query), args...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, n := range nn {
		for _, u := range uu {
			if u.ID == n.UserID && n.User == nil {
				n.User = u
				n.User.Email = strings.TrimSpace(n.User.Email)
				n.User.Username = strings.TrimSpace(n.User.Username)
			}
		}
	}

	return errors.Err(err)
}

func (n *Namespace) BuildStore() BuildStore {
	return BuildStore{
		Store: &Store{
			DB: n.DB,
		},
		namespace: n,
	}
}

func (n *Namespace) NamespaceStore() NamespaceStore {
	return NamespaceStore{
		Store: &Store{
			DB: n.DB,
		},
		namespace: n,
	}
}

func (n *Namespace) Create() error {
	stmt, err := n.Prepare(`
		INSERT INTO namespaces (user_id, parent_id, name, path, description, level, visibility)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(
		n.UserID,
		n.ParentID,
		n.Name,
		n.Path,
		n.Description,
		n.Level,
		n.Visibility,
	)

	return errors.Err(row.Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt))
}

func (n *Namespace) Update() error {
	stmt, err := n.Prepare(`
		UPDATE namespaces
		SET name = $1, path = $2, description = $3, visibility = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	row := stmt.QueryRow(n.Name, n.Path, n.Description, n.Visibility, n.ID)

	return errors.Err(row.Scan(&n.UpdatedAt))
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
		Store: &Store{
			DB: n.DB,
		},
		user: n.User,
	}

	n.Children, err = namespaces.GetByParentID(n.ID)

	return errors.Err(err)
}

func (n *Namespace) LoadParent() error {
	if !n.ParentID.Valid {
		n.Parent = &Namespace{}

		return nil
	}

	var err error

	namespaces := NamespaceStore{
		Store: &Store{
			DB: n.DB,
		},
		user: n.User,
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
