package namespace

import (
	"database/sql"
	"strconv"
	"time"

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

	Inviter   *user.User
	Invitee   *user.User
	Namespace *Namespace
}

var _ database.Model = (*Invite)(nil)

func InviteRelations(db database.Pool) []database.RelationFunc {
	users := user.Store{Pool: db}
	namespaces := Store{Pool: db}

	return []database.RelationFunc{
		database.Relation("inviter_id", "id", users),
		database.Relation("invitee_id", "id", users),
		database.Relation("namespace_id", "id", namespaces),
	}
}

func LoadInviteRelations(db database.Pool, ii ...*Invite) error {
	mm := make([]database.Model, 0, len(ii))

	for _, i := range ii {
		mm = append(mm, i)
	}

	if err := database.LoadRelations(mm, InviteRelations(db)...); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (i *Invite) Dest() []interface{} {
	return []interface{}{
		&i.ID,
		&i.NamespaceID,
		&i.InviteeID,
		&i.InviterID,
		&i.CreatedAt,
	}
}

func (i *Invite) Bind(m database.Model) {
	switch v := m.(type) {
	case *user.User:
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

func (i *Invite) Endpoint(_ ...string) string { return "/invites/" + strconv.FormatInt(i.ID, 10) }

func (i *Invite) JSON(addr string) map[string]interface{} {
	if i == nil {
		return nil
	}

	json := map[string]interface{}{
		"id":           i.ID,
		"namespace_id": i.NamespaceID,
		"invitee_id":   i.InviteeID,
		"inviter_id":   i.InviterID,
		"url":          addr + i.Endpoint(),
		"invitee":      i.Invitee.JSON(addr),
		"inviter":      i.Inviter.JSON(addr),
		"namespace":    i.Namespace.JSON(addr),
	}

	if i.Invitee != nil {
		json["invitee"] = i.Invitee.JSON(addr)
	}

	if i.Inviter != nil {
		json["inviter"] = i.Inviter.JSON(addr)
	}

	if i.Namespace != nil {
		json["namespace"] = i.Namespace.JSON(addr)
	}
	return json
}

func (i *Invite) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":           i.ID,
		"namespace_id": i.NamespaceID,
		"invitee_id":   i.InviteeID,
		"inviter_id":   i.InviterID,
		"created_at":   i.CreatedAt,
	}
}

type InviteEvent struct {
	db  database.Pool
	dis event.Dispatcher

	Action    string
	Namespace *Namespace
	Invitee   *user.User
	Inviter   *user.User
}

var _ queue.Job = (*InviteEvent)(nil)

func InitInviteEvent(db database.Pool, dis event.Dispatcher) queue.InitFunc {
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
		users := user.Store{
			Pool: e.db,
		}

		u, _, err := users.Get(query.Where("id", "=", query.Arg(e.Namespace.UserID)))

		if err != nil {
			return errors.Err(err)
		}
		e.Namespace.User = u
	}

	payload := map[string]interface{}{
		"namespace": e.Namespace.JSON(env.DJINN_API_SERVER),
	}

	typs := map[string]event.Type{
		"sent":     event.InviteSent,
		"accepted": event.InviteAccepted,
		"rejected": event.InviteRejected,
	}

	switch e.Action {
	case "sent":
		payload["inviter"] = e.Inviter.JSON(env.DJINN_API_SERVER)
		payload["invitee"] = e.Invitee.JSON(env.DJINN_API_SERVER)
	case "accepted":
		payload["invitee"] = e.Invitee.JSON(env.DJINN_API_SERVER)
	case "rejected":
		payload["invitee"] = e.Invitee.JSON(env.DJINN_API_SERVER)
	default:
		return errors.New("invalid invite action " + e.Action)
	}

	namespaceId := sql.NullInt64{
		Int64: e.Namespace.ID,
		Valid: true,
	}

	ev := event.New(namespaceId, typs[e.Action], payload)

	if err := e.dis.Dispatch(ev); err != nil {
		return errors.Err(err)
	}
	return nil
}

type InviteStore struct {
	database.Pool
}

var inviteTable = "namespace_invites"

func (s InviteStore) Get(opts ...query.Option) (*Invite, bool, error) {
	var i Invite

	ok, err := s.Pool.Get(inviteTable, &i, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &i, ok, nil
}

func (s InviteStore) All(opts ...query.Option) ([]*Invite, error) {
	ii := make([]*Invite, 0)

	new := func() database.Model {
		i := &Invite{}
		ii = append(ii, i)
		return i
	}

	if err := s.Pool.All(inviteTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return ii, nil
}

func getInviteeAndInviter(db database.Pool, inviteeId, inviterId int64) (*user.User, *user.User, error) {
	users := user.Store{
		Pool: db,
	}

	uu, err := users.All(
		query.Where("id", "=", query.Arg(inviteeId)),
		query.OrWhere("id", "=", query.Arg(inviterId)),
		query.Where("deleted_at", "IS", query.Lit("NULL")),
	)

	if err != nil {
		return nil, nil, errors.Err(err)
	}

	var (
		inviter *user.User
		invitee *user.User
	)

	if uu[0].ID == inviterId {
		inviter = uu[0]
		invitee = uu[1]
	} else {
		inviter = uu[1]
		invitee = uu[0]
	}
	return invitee, inviter, nil
}

func (s InviteStore) Accept(id int64) (*Namespace, *user.User, *user.User, error) {
	i, ok, err := s.Get(query.Where("id", "=", query.Arg(id)))

	if err != nil {
		return nil, nil, nil, errors.Err(err)
	}

	if !ok {
		return nil, nil, nil, database.ErrNotFound
	}

	invitee, inviter, err := getInviteeAndInviter(s.Pool, i.InviteeID, i.InviterID)

	if err != nil {
		return nil, nil, nil, errors.Err(err)
	}

	namespaces := Store{
		Pool: s.Pool,
	}

	n, ok, err := namespaces.Get(query.Where("id", "=", query.Arg(i.NamespaceID)))

	if err != nil {
		return nil, nil, nil, errors.Err(err)
	}

	if !ok {
		return nil, nil, nil, database.ErrNotFound
	}

	if n.RootID.Int64 != n.ID {
		n, ok, err = namespaces.Get(query.Where("id", "=", query.Arg(n.RootID)))

		if err != nil {
			return nil, nil, nil, errors.Err(err)
		}

		if !ok {
			return nil, nil, nil, database.ErrNotFound
		}
	}

	q := query.Insert(
		collaboratorTable,
		query.Columns("namespace_id", "user_id", "created_at"),
		query.Values(n.RootID, invitee.ID, time.Now()),
	)

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return nil, nil, nil, errors.Err(err)
	}

	if err := s.Delete(i.ID); err != nil {
		return nil, nil, nil, errors.Err(err)
	}
	return n, inviter, invitee, nil
}

type InviteParams struct {
	NamespaceID int64
	InviterID   int64
	InviteeID   int64
}

func (s InviteStore) Create(p InviteParams) (*Invite, error) {
	ownerId, err := getNamespaceOwnerId(s.Pool, p.NamespaceID)

	if err != nil {
		return nil, errors.Err(err)
	}

	if p.InviterID != ownerId {
		return nil, ErrPermission
	}

	invitee, inviter, err := getInviteeAndInviter(s.Pool, p.InviteeID, p.InviterID)

	now := time.Now()

	q := query.Insert(
		inviteTable,
		query.Columns("namespace_id", "invitee_id", "inviter_id", "created_at"),
		query.Values(p.NamespaceID, p.InviteeID, p.InviterID, now),
		query.Returning("id"),
	)

	var id int64

	if err := s.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}
	return &Invite{
		ID:          id,
		NamespaceID: p.NamespaceID,
		InviteeID:   p.InviteeID,
		InviterID:   p.InviterID,
		CreatedAt:   now,
		Invitee:     invitee,
		Inviter:     inviter,
	}, nil
}

func (s InviteStore) Delete(id int64) error {
	q := query.Delete(inviteTable, query.Where("id", "=", query.Arg(id)))

	if _, err := s.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}
	return nil
}
