package namespace

import (
	"net"
	"net/url"
	"strings"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

// Form is the type that represents input data for creating a new Namespace.
type Form struct {
	Namespaces *Store     `schema:"-" json:"-"`
	Namespace  *Namespace `schema:"-" json:"-"`

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

	Owner  string `schema:"-" json:"-"`
	Handle string `schema:"handle" json:"handle"`
}

type WebhookForm struct {
	Webhooks *WebhookStore `schema:"-" json:"-"`
	Webhook  *Webhook      `schema:"-" json:"-"`

	PayloadURL   string   `schema:"payload_url" json:"payload_url"`
	Secret       string   `schema:"secret"      json:"secret"`
	RemoveSecret bool     `schema:"remove_secret" json:"remove_secret"`
	SSL          bool     `schema:"ssl"         json:"ssl"`
	Active       bool     `schema:"active"      json:"active"`
	Events       []string `schema:"events[]"    json:"events"`
}

var (
	_ webutil.Form = (*Form)(nil)
	_ webutil.Form = (*InviteForm)(nil)
	_ webutil.Form = (*WebhookForm)(nil)

	localhosts = map[string]struct{}{
		"localhost": {},
		"127.0.0.1": {},
		"::1":       {},
	}
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
	errs := webutil.NewErrors()

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
		errs.Put("name", webutil.ErrFieldRequired("Name"))
	}

	if len(f.Name) < 3 || len(f.Name) > 32 {
		errs.Put("name", webutil.ErrField("Name", errors.New("must be between 3 and 32 characters in length")))
	}

	if !rename.Match([]byte(f.Name)) {
		errs.Put("name", webutil.ErrField("Name", errors.New("can only contain letters and numbers")))
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
		n, err := f.Namespaces.Get(
			database.Where(f.Namespaces.User, "user_id"),
			query.Where("path", "=", query.Arg(f.Name)),
		)

		if err != nil {
			return errors.Err(err)
		}

		if !n.IsZero() {
			errs.Put("name", webutil.ErrFieldExists("Name"))
		}
	}

	if len(f.Description) > 255 {
		errs.Put("description", webutil.ErrField("Description", errors.New("must be shorter than 255 characters in length")))
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
	errs := webutil.NewErrors()

	if f.Handle == "" {
		errs.Put("handle", webutil.ErrFieldRequired("Username"))
	}

	invitee, err := f.Users.Get(user.WhereHandle(f.Handle))

	if err != nil {
		return errors.Err(err)
	}

	if invitee.IsZero() {
		errs.Put("handle", errors.New("No such user"))
	}

	inviter, err := f.Users.Get(query.Where("username", "=", query.Arg(f.Owner)))

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

	i, err := f.Invites.Get(query.Where("invitee_id", "=", query.Arg(invitee.ID)))

	if err != nil {
		return errors.Err(err)
	}

	if !i.IsZero() {
		errs.Put("handle", errors.New("User already invited"))
	}

	c, err := f.Collaborators.Get(query.Where("user_id", "=", query.Arg(invitee.ID)))

	if err != nil {
		return errors.Err(err)
	}

	if !c.IsZero() {
		errs.Put("handle", errors.New("User is already a collaborator"))
	}
	return errs.Err()
}

func (f WebhookForm) Fields() map[string]string {
	return map[string]string{
		"payload_url": f.PayloadURL,
		"events":      strings.Join(f.Events, " "),
	}
}

func (f WebhookForm) Validate() error {
	errs := webutil.NewErrors()

	if f.PayloadURL == "" {
		errs.Put("payload_url", webutil.ErrFieldRequired("Payload URL"))
	}

	w, err := f.Webhooks.Get(query.Where("payload_url", "=", query.Arg(strings.ToLower(f.PayloadURL))))

	if err != nil {
		return errors.Err(err)
	}

	if !w.IsZero() {
		if f.Webhook == nil || f.Webhook.PayloadURL.String() != f.PayloadURL {
			errs.Put("payload_url", errors.New("Webhook for URL already exists"))
		}
	}

	url, err := url.Parse(f.PayloadURL)

	if err != nil {
		errs.Put("payload_url", errors.New("Invalid payload URL"))
	}

	if url.Scheme != "http" && url.Scheme != "https" {
		errs.Put("payload_url", errors.New("Invalid payload URL"))
	}

	host, _, err := net.SplitHostPort(url.Host)

	if nerr, ok := err.(*net.AddrError); ok {
		if nerr.Err == "missing port in address" {
			host = url.Host
		}
	}

	if host == "" {
		errs.Put("payload_url", errors.New("Invalid payload URL, missing host"))
	}

	if _, ok := localhosts[host]; ok {
		errs.Put("payload_url", errors.New("Invalid payload URL, cannot be localhost"))
	}

	if f.SSL {
		if url.Scheme == "http" {
			errs.Put("payload_url", errors.New("Cannot use SSL for a plain http URL"))
		}
	}
	return errs.Err()
}
