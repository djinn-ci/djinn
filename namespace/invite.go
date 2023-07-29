package namespace

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/env"
	"djinn-ci.com/errors"
	"djinn-ci.com/event"
	"djinn-ci.com/queue"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"
)

type Invite struct {
	ID          int64
	NamespaceID int64
	InviteeID   int64
	InviterID   int64
	CreatedAt   time.Time

	Inviter   *auth.User
	Invitee   *auth.User
	Namespace *Namespace
}

func LoadInviteRelations(ctx context.Context, pool *database.Pool, ii ...*Invite) error {
	if len(ii) == 0 {
		return nil
	}

	rels := []database.Relation{
		{
			From: "namespace_id",
			To:   "id",
			Loader: database.ModelLoader(pool, table, func() database.Model {
				return &Namespace{}
			}),
		},
		{
			From:   "inviter_id",
			To:     "id",
			Loader: user.Loader(pool),
		},
		{
			From:   "invitee_id",
			To:     "id",
			Loader: user.Loader(pool),
		},
	}

	if err := database.LoadRelations[*Invite](ctx, ii, rels...); err != nil {
		return errors.Err(err)
	}
	return nil
}

var _ database.Model = (*Invite)(nil)

func (i *Invite) Primary() (string, any) { return "id", i.ID }

func (i *Invite) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":           &i.ID,
		"namespace_id": &i.NamespaceID,
		"invitee_id":   &i.InviteeID,
		"inviter_id":   &i.InviterID,
		"created_at":   &i.CreatedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (i *Invite) Params() database.Params {
	params := database.Params{
		"id":           database.ImmutableParam(i.ID),
		"namespace_id": database.CreateOnlyParam(i.NamespaceID),
		"invitee_id":   database.CreateOnlyParam(i.InviteeID),
		"inviter_id":   database.CreateOnlyParam(i.InviterID),
		"created_at":   database.CreateOnlyParam(i.CreatedAt),
	}

	return params
}

func (i *Invite) Bind(m database.Model) {
	switch v := m.(type) {
	case *auth.User:
		if v.ID == i.InviterID {
			i.Inviter = v
		}

		if v.ID == i.InviteeID {
			i.Invitee = v
		}
	case *Namespace:
		if i.NamespaceID == v.ID {
			i.Namespace = v
		}
	}
}

func (i *Invite) Endpoint(...string) string {
	return "/invites/" + strconv.FormatInt(i.ID, 10)
}

func (i *Invite) MarshalJSON() ([]byte, error) {
	if i == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"id":           i.ID,
		"namespace_id": i.NamespaceID,
		"invitee_id":   i.InviteeID,
		"inviter_id":   i.InviterID,
		"url":          env.DJINN_API_SERVER + i.Endpoint(),
		"invitee":      i.Invitee,
		"inviter":      i.Inviter,
		"namespace":    i.Namespace,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

type InviteEvent struct {
	db  *database.Pool
	dis event.Dispatcher

	Action    string
	Namespace *Namespace
	Invitee   *auth.User
	Inviter   *auth.User
}

var _ queue.Job = (*InviteEvent)(nil)

func InitInviteEvent(db *database.Pool, dis event.Dispatcher) queue.InitFunc {
	return func(j queue.Job) {
		if ev, ok := j.(*InviteEvent); ok {
			ev.db = db
			ev.dis = dis
		}
	}
}

func (e *InviteEvent) Name() string {
	switch e.Action {
	case "accepted":
		return "event:" + event.InviteAccepted.String()
	case "rejected":
		return "event:" + event.InviteRejected.String()
	case "sent":
		return "event:" + event.InviteSent.String()
	default:
		return "event:invite"
	}
}

func (e *InviteEvent) Perform() error {
	if e.dis == nil {
		return event.ErrNilDispatcher
	}

	if e.Namespace.User == nil {
		u, _, err := user.NewStore(e.db).Get(
			context.Background(),
			query.Where("id", "=", query.Arg(e.Namespace.UserID)),
		)

		if err != nil {
			return errors.Err(err)
		}
		e.Namespace.User = u
	}

	payload := map[string]any{
		"namespace": e.Namespace,
	}

	typs := map[string]event.Type{
		"sent":     event.InviteSent,
		"accepted": event.InviteAccepted,
		"rejected": event.InviteRejected,
	}

	switch e.Action {
	case "sent":
		payload["inviter"] = e.Inviter
		payload["invitee"] = e.Invitee
	case "accepted", "rejected":
		payload["invitee"] = e.Invitee
	default:
		return errors.New("namespace: invalid invite action " + e.Action)
	}

	namespaceId := database.Null[int64]{
		Elem:  e.Namespace.ID,
		Valid: true,
	}

	ev := event.New(namespaceId, typs[e.Action], payload)

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

type InviteStore struct {
	*database.Store[*Invite]
}

const inviteTable = "namespace_invites"

func NewInviteStore(pool *database.Pool) InviteStore {
	return InviteStore{
		Store: database.NewStore[*Invite](pool, inviteTable, func() *Invite {
			return &Invite{}
		}),
	}
}

func (s InviteStore) Accept(ctx context.Context, u *auth.User, i *Invite) (*Namespace, *auth.User, *auth.User, error) {
	if i.InviteeID != u.ID {
		return nil, nil, nil, database.ErrNoRows
	}

	users := user.NewStore(s.Pool)

	invitee, ok, err := users.Get(ctx, query.Where("id", "=", query.Arg(i.InviteeID)))

	if err != nil {
		return nil, nil, nil, errors.Err(err)
	}

	if !ok {
		return nil, nil, nil, database.ErrNoRows
	}

	inviter, ok, err := users.Get(ctx, query.Where("id", "=", query.Arg(i.InviterID)))

	if err != nil {
		return nil, nil, nil, errors.Err(err)
	}

	if !ok {
		return nil, nil, nil, database.ErrNoRows
	}

	namespaces := NewStore(s.Pool)

	n, ok, err := namespaces.Get(ctx, query.Where("id", "=", query.Arg(i.NamespaceID)))

	if err != nil {
		return nil, nil, nil, errors.Err(err)
	}

	if !ok {
		return nil, nil, nil, database.ErrNoRows
	}

	if n.RootID.Elem != n.ID {
		n, ok, err = namespaces.Get(ctx, query.Where("id", "=", query.Arg(n.RootID)))

		if err != nil {
			return nil, nil, nil, errors.Err(err)
		}

		if !ok {
			return nil, nil, nil, database.ErrNoRows
		}
	}

	c := Collaborator{
		NamespaceID: n.ID,
		UserID:      invitee.ID,
		CreatedAt:   time.Now(),
	}

	if err := NewCollaboratorStore(s.Pool).Create(ctx, &c); err != nil {
		return nil, nil, nil, errors.Err(err)
	}

	if err := s.Delete(ctx, i); err != nil {
		return nil, nil, nil, errors.Err(err)
	}
	return n, inviter, invitee, nil
}

type InviteParams struct {
	Handle    string
	Inviter   *auth.User
	Namespace *Namespace
}

var (
	ErrSelfInvite   = errors.New("namespace: cannot invite self")
	ErrInviteSent   = errors.New("namespace: invite sent")
	ErrCollaborator = errors.New("namespace: already a collaborator")
)

func (s InviteStore) Create(ctx context.Context, p *InviteParams) (*Invite, error) {
	if p.Inviter.ID != p.Namespace.UserID {
		return nil, auth.ErrPermission
	}

	u, ok, err := user.NewStore(s.Pool).Get(ctx, user.WhereHandle(p.Handle))

	if err != nil {
		return nil, errors.Err(err)
	}

	if !ok {
		return nil, database.ErrNoRows
	}

	if u.ID == p.Inviter.ID {
		return nil, ErrSelfInvite
	}

	_, ok, err = s.Get(ctx, query.Where("invitee_id", "=", query.Arg(u.ID)))

	if err != nil {
		return nil, errors.Err(err)
	}

	if ok {
		return nil, ErrInviteSent
	}

	_, ok, err = NewCollaboratorStore(s.Pool).Get(ctx, query.Where("user_id", "=", query.Arg(u.ID)))

	if err != nil {
		return nil, errors.Err(err)
	}

	if ok {
		return nil, ErrCollaborator
	}

	i := Invite{
		NamespaceID: p.Namespace.ID,
		InviterID:   p.Inviter.ID,
		InviteeID:   u.ID,
		CreatedAt:   time.Now(),
		Namespace:   p.Namespace,
		Inviter:     p.Inviter,
		Invitee:     u,
	}

	if err := s.Store.Create(ctx, &i); err != nil {
		return nil, errors.Err(err)
	}
	return &i, nil
}
