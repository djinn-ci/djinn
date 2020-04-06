package namespace

import (
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"
)

type Form struct {
	Namespaces Store      `schema:"-"`
	Namespace  *Namespace `schema:"-"`

	UserID      int64      `schema:"-"`
	Parent      string     `schema:"parent"`
	Name        string     `schema:"name"`
	Description string     `schema:"description"`
	Visibility  Visibility `schema:"visibility"`
}

type ResourceForm struct {
	User          *user.User        `schema:"-"`
	Namespaces    Store             `schema:"-"`
	Collaborators CollaboratorStore `schema:"-"`

	Namespace string `schema:"namespace"`
}

type InviteForm struct {
	Collaborators CollaboratorStore `schema:"-"`
	Invites       InviteStore       `schema:"-"`
	Users         user.Store        `schema:"-"`
	Inviter       *user.User        `schema:"-"`
	Invitee       *user.User        `schema:"-"`

	Handle string `schema:"handle"`
}

var (
	_ form.Form = (*Form)(nil)
	_ form.Form = (*ResourceForm)(nil)
	_ form.Form = (*InviteForm)(nil)
)

func (f Form) Fields() map[string]string {
	return map[string]string{
		"name":        f.Name,
		"description": f.Description,
	}
}

func (f Form) Validate() error {
	errs := form.NewErrors()

	if f.Name == "" {
		errs.Put("name", form.ErrFieldRequired("Name"))
	}

	if len(f.Name) < 3 || len(f.Name) > 32 {
		errs.Put("name", form.ErrFieldInvalid("Name", "must be between 3 and 32 characters in length"))
	}

	if !rename.Match([]byte(f.Name)) {
		errs.Put("name", form.ErrFieldInvalid("Name", "can only contain letters and numbers"))
	}

	checkUnique := true

	if !f.Namespace.IsZero() {
		parts := strings.Split(f.Namespace.Path, "/")
		parts[len(parts)-1] = f.Name

		if f.Namespace.Name == f.Name {
			checkUnique = false
		}
	} else if f.Parent != "" {
		f.Name = strings.Join([]string{f.Parent, f.Name}, "/")
	}

	if checkUnique {
		n, err := f.Namespaces.Get(query.Where("path", "=", f.Name))

		if err != nil {
			return errors.Err(err)
		}

		if !n.IsZero() {
			errs.Put("name", form.ErrFieldExists("Name"))
		}
	}

	if len(f.Description) > 255 {
		errs.Put("description", form.ErrFieldInvalid("Description", "must be shorter than 255 characters in length"))
	}
	return errs.Err()
}

func (f ResourceForm) Fields() map[string]string { return map[string]string{} }

func (f ResourceForm) Validate() error {
	n, err := f.Namespaces.GetByPath(f.Namespace)

	if err != nil {
		return errors.Err(err)
	}

	f.Collaborators.Bind(n)

	cc, err := f.Collaborators.All()

	if err != nil {
		return errors.Err(err)
	}

	n.LoadCollaborators(cc)

	if !n.CanAdd(f.User) {
		return ErrPermission
	}
	return nil
}

func (f InviteForm) Fields() map[string]string {
	return map[string]string{
		"handle": f.Handle,
	}
}

func (f InviteForm) Validate() error {
	errs := form.NewErrors()

	if f.Handle == "" {
		errs.Put("handle", form.ErrFieldRequired("Username"))
	}

	if f.Handle == f.Inviter.Email || f.Handle == f.Inviter.Username {
		errs.Put("handle", errors.New("You cannot add yourself as a collaborator"))
	}

	u, err := f.Users.Get(user.WhereHandle(f.Handle))

	if err != nil {
		return errors.Err(err)
	}

	if u.IsZero() {
		errs.Put("handle", errors.New("Could not find user"))
	}

	selectq := user.Select("id", user.WhereHandle(f.Handle))

	i, err := f.Invites.Get(query.WhereQuery("invitee_id", "=", selectq))

	if err != nil {
		return errors.Err(err)
	}

	if !i.IsZero() {
		errs.Put("handle", errors.New("User already invited"))
	}

	c, err := f.Collaborators.Get(query.WhereQuery("user_id", "=", selectq))

	if err != nil {
		return errors.Err(err)
	}

	if !c.IsZero() {
		errs.Put("handle", errors.New("User is already a collaborator"))
	}

	f.Invitee = u

	return errs.Err()
}
