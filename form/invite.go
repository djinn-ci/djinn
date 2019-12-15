package form

import (
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"

	"github.com/andrewpillar/query"
)

type Invite struct {
	Collaborators model.CollaboratorStore `schema:"-"`
	Invites       model.InviteStore       `schema:"-"`
	Users         model.UserStore         `schema:"-"`
	Inviter       *model.User             `schema:"-"`
	Invitee       *model.User             `schema:"-"`

	Handle string `schema:"handle"`
}

func (f *Invite) Validate() error {
	errs := NewErrors()

	if f.Handle == "" {
		errs.Put("handle", ErrFieldRequired("Username/Email"))
	}

	if f.Handle == f.Inviter.Email || f.Handle == f.Inviter.Username {
		errs.Put("handle", errors.New("You cannot add yourself as a collaborator"))
	}

	u, err := f.Users.Get(
		query.Where("username", "=", f.Handle),
		query.OrWhere("email", "=", f.Handle),
	)

	if err != nil {
		return errors.Err(err)
	}

	if u.IsZero() {
		errs.Put("handle", errors.New("Could not find user"))
	}

	i, err := f.Invites.Get(
		query.WhereQuery("invitee_id", "=",
			query.Select(
				query.Columns("id"),
				query.From(model.UserTable),
				query.Where("email", "=", f.Handle),
				query.OrWhere("username", "=", f.Handle),
			),
		),
	)

	if err != nil {
		return errors.Err(err)
	}

	if !i.IsZero() {
		errs.Put("handle", errors.New("User already invited"))
	}

	c, err := f.Collaborators.Get(
		query.WhereQuery("user_id", "=",
			query.Select(
				query.Columns("id"),
				query.From(model.UserTable),
				query.Where("email", "=", f.Handle),
				query.OrWhere("username", "=", f.Handle),
			),
		),
	)

	if err != nil {
		return errors.Err(err)
	}

	if !c.IsZero() {
		errs.Put("handle", errors.New("User is already a collaborator"))
	}

	f.Invitee = u

	return errs.Err()
}

func (f Invite) Fields() map[string]string {
	return map[string]string{
		"handle": f.Handle,
	}
}
