package namespace

import (
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/user"

	"github.com/andrewpillar/query"
)

// Form is the type that represents input data for creating a new Namespace.
type Form struct {
	Namespaces *Store     `schema:"-"`
	Namespace  *Namespace `schema:"-"`

	Parent      string     `schema:"parent"      json:"parent"`
	Name        string     `schema:"name"        json:"name"`
	Description string     `schema:"description" json:"description"`
	Visibility  Visibility `schema:"visibility"  json:"visibility"`
}

// InviteForm is the type that represents input data for sending an Invite.
type InviteForm struct {
	Namespaces    *Store             `schema:"-"`
	Collaborators *CollaboratorStore `schema:"-"`
	Invites       *InviteStore       `schema:"-"`
	Users         *user.Store        `schema:"-"`

	// Inviter is the original User who sent the invite.
	Inviter *user.User `schema:"-"`

	// Invitee is the User who received the invite.
	Invitee *user.User `schema:"-"`

	Namespace string `schema:"-"`
	Owner     string `schema:"-"`
	Handle    string `schema:"handle" json:"handle"`
}

var (
	_ form.Form = (*Form)(nil)
	_ form.Form = (*InviteForm)(nil)
)

// Fields returns a map of the Name and Description fields in the Namespace
// form.
func (f Form) Fields() map[string]string {
	return map[string]string{
		"name":        f.Name,
		"description": f.Description,
	}
}

// Validate checks to see if the Namespace Name is present, between 3 and 32
// characters in length, contains only letters and numbers, and is unique to
// the current Namespace. This uniqueness check is skipped if a Namespace is
// set, and the Name field already matches that name (assuming it's being
// edited). The description field is checked to see if it is less than 255
// characters in length, if present.
func (f Form) Validate() error {
	errs := form.NewErrors()

	if f.Namespace != nil {
		if f.Name == "" {
			f.Name = f.Namespace.Name
		}

		if f.Description == "" {
			f.Description = f.Namespace.Description
		}

		if f.Visibility == Visibility(0) {
			f.Visibility = f.Namespace.Visibility
		}
	}

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

// Fields returns a map of just the Handle field from the current InviteForm.
func (f *InviteForm) Fields() map[string]string {
	return map[string]string{
		"handle": f.Handle,
	}
}

// Validate checks to see if the current InviteForm has the handle of the User
// being invited available, if the User being invited is not the current User,
// and if that User actually exists, and is not already in the Namespace. If
// all of these checks pass, then the User being invited is set as the Invitee
// field on the InviteForm.
func (f *InviteForm) Validate() error {
	errs := form.NewErrors()

	if f.Handle == "" {
		errs.Put("handle", form.ErrFieldRequired("Username"))
	}

	invitee, err := f.Users.Get(user.WhereHandle(f.Handle))

	if err != nil {
		return errors.Err(err)
	}

	if invitee.IsZero() {
		errs.Put("handle", errors.New("No such user"))
	}

	inviter, err := f.Users.Get(query.Where("username", "=", f.Owner))

	if err != nil {
		return errors.Err(err)
	}

	if inviter.IsZero() {
		return ErrPermission
	}

	if invitee.ID == inviter.ID {
		errs.Put("handle", errors.New("Cannot invite yourself"))
	}

	f.Inviter = inviter
	f.Invitee = invitee

	i, err := f.Invites.Get(query.Where("invitee_id", "=", invitee.ID))

	if err != nil {
		return errors.Err(err)
	}

	if !i.IsZero() {
		errs.Put("handle", errors.New("User already invited"))
	}

	c, err := f.Collaborators.Get(query.Where("user_id", "=", invitee.ID))

	if err != nil {
		return errors.Err(err)
	}

	if !c.IsZero() {
		errs.Put("handle", errors.New("User is already a collaborator"))
	}
	return errs.Err()
}
