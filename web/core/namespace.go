package core

import (
	"database/sql"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"
)

type Namespace struct {
	web.Handler

	Namespaces model.NamespaceStore
}

var NamespaceMaxDepth int64 = 20

func (h Namespace) Namespace(r *http.Request) *model.Namespace {
	val := r.Context().Value("namespace")

	n, _ := val.(*model.Namespace)

	return n
}

// Get a namespace by the given path. If the namespace does not exist then it
// will be created on the fly along with each sub-namespace. A namespace for a
// specific user can be retrieved like this if there is an '@' in the path. For
// example 'parent/child@user'.
func (h Namespace) Get(path string, u *model.User) (*model.Namespace, error) {
	namespaces := u.NamespaceStore()

	if strings.Contains(path, "@") {
		parts := strings.Split(path, "@")

		username := parts[1]
		path = parts[0]

		u, err := h.Users.FindByUsername(username)

		if err != nil {
			return &model.Namespace{}, errors.Err(err)
		}

		if !u.IsZero() {
			namespaces = u.NamespaceStore()
		}
	}

	n, err := namespaces.FindByPath(path)

	if err != nil {
		return n, errors.Err(err)
	}

	if !n.IsZero() {
		err = n.LoadCollaborators()

		return n, errors.Err(err)
	}

	p := &model.Namespace{}

	parts := strings.Split(path, "/")

	for _, name := range parts {
		if p.Level + 1 > NamespaceMaxDepth {
			break
		}

		if matched, err := regexp.Match("^[a-zA-Z0-9]+$", []byte(name)); !matched || err != nil {
			break
		}

		n = namespaces.New()
		n.Name = name
		n.Path = name
		n.Level = p.Level + 1

		if !p.IsZero() {
			n.RootID = p.RootID
			n.ParentID = sql.NullInt64{
				Int64: p.ID,
				Valid: true,
			}

			n.Path = strings.Join([]string{p.Path, n.Name}, "/")
		}

		if err := namespaces.Create(n); err != nil {
			return n, errors.Err(err)
		}

		if p.IsZero() {
			n.RootID = sql.NullInt64{
				Int64: n.ID,
				Valid: true,
			}

			if err := namespaces.Update(n); err != nil {
				return n, errors.Err(err)
			}
		}

		p = n
	}

	return n, nil
}

func (h Namespace) Destroy(r *http.Request) error {
	n := h.Namespace(r)

	err := h.Namespaces.Delete(n)

	return errors.Err(err)
}

func (h Namespace) Index(namespaces model.NamespaceStore, r *http.Request, opts ...query.Option) ([]*model.Namespace, model.Paginator, error) {
	u := h.User(r)

	q := r.URL.Query()

	search := q.Get("search")

	page, err := strconv.ParseInt(q.Get("page"), 10, 64)

	if err != nil {
		page = 1
	}

	index := []query.Option{
		model.Search("path", search),
		model.NamespaceSharedWith(u),
	}

	paginator, err := namespaces.Paginate(page, append(index, opts...)...)

	if err != nil {
		return []*model.Namespace{}, paginator, errors.Err(err)
	}

	index = append(
		index,
		query.OrderAsc("path"),
		query.Limit(model.PageLimit),
		query.Offset(paginator.Offset),
	)

	nn, err := namespaces.All(index...)

	if err != nil {
		return nn, paginator, errors.Err(err)
	}

	if err := namespaces.LoadUsers(nn); err != nil {
		return nn, paginator, errors.Err(err)
	}

	err = namespaces.LoadLastBuild(nn)

	return nn, paginator, errors.Err(err)
}

func (h Namespace) Store(w http.ResponseWriter, r *http.Request) (*model.Namespace, error) {
	u := h.User(r)

	namespaces := u.NamespaceStore()

	f := &form.Namespace{
		Namespaces: namespaces,
		UserID:     u.ID,
	}

	if err := h.ValidateForm(f, w, r); err != nil {
		if _, ok := err.(form.Errors); ok {
			return &model.Namespace{}, ErrValidationFailed
		}

		return &model.Namespace{}, errors.Err(err)
	}

	parent, err := namespaces.FindByPath(f.Parent)

	if err != nil {
		return &model.Namespace{}, errors.Err(err)
	}

	n := namespaces.New()
	n.Name = f.Name
	n.Path = f.Name
	n.Description = f.Description
	n.Visibility = f.Visibility

	if !parent.IsZero() {
		n.RootID = parent.RootID
		n.ParentID = sql.NullInt64{
			Int64: parent.ID,
			Valid: true,
		}
		n.Path = strings.Join([]string{parent.Path, f.Name}, "/")
		n.Level = parent.Level + 1
		n.Visibility = parent.Visibility
	}

	if n.Level >= NamespaceMaxDepth {
		return n, ErrNamespaceTooDeep
	}

	if err := namespaces.Create(n); err != nil {
		return n, errors.Err(err)
	}

	if parent.IsZero() {
		n.RootID = sql.NullInt64{
			Int64: n.ID,
			Valid: true,
		}

		err := namespaces.Update(n)

		return n, errors.Err(err)
	}

	return n, nil
}

func (h Namespace) Update(w http.ResponseWriter, r *http.Request) (*model.Namespace, error) {
	u := h.User(r)
	n := h.Namespace(r)

	f := &form.Namespace{
		Namespaces: u.NamespaceStore(),
		Namespace:  n,
		UserID:     n.UserID,
	}

	if err := h.ValidateForm(f, w, r); err != nil {
		if _, ok := err.(form.Errors); ok {
			return &model.Namespace{}, ErrValidationFailed
		}

		return &model.Namespace{}, errors.Err(err)
	}

	if err := n.LoadParent(); err != nil {
		return &model.Namespace{}, errors.Err(err)
	}

	n.Name = f.Name
	n.Description = f.Description
	n.Visibility = f.Visibility

	if !n.Parent.IsZero() {
		n.Visibility = n.Parent.Visibility
	} else {
		if err := n.CascadeVisibility(); err != nil {
			return n, errors.Err(err)
		}
	}

	err := h.Namespaces.Update(n)

	return n, errors.Err(err)
}
