package handler

import (
	"net/http"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/image"
	"github.com/andrewpillar/djinn/key"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/object"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/variable"
	"github.com/andrewpillar/djinn/web"

	"github.com/gorilla/mux"
)

// Collaborator is the base handler that provides shared logic for the UI and
// API handlers for working with namespace collaborators.
type Collaborator struct {
	web.Handler
}

// DeleteModel deletes the specified collaborator in the request from the
// namespace in the given request context. Upon deletion this will modify the
// ownership of any resources created by the original collaborator to the owner
// of the namespace.
func (h Collaborator) DeleteModel(r *http.Request) error {
	ctx := r.Context()

	u, ok := user.FromContext(ctx)

	if !ok {
		return errors.New("no user in request context")
	}

	n, ok := namespace.FromContext(r.Context())

	if !ok {
		return errors.New("no namespace in request context")
	}

	collaborator := mux.Vars(r)["collaborator"]

	if collaborator == u.Username {
		return namespace.ErrDeleteSelf
	}

	err := namespace.NewCollaboratorStore(h.DB, n).Delete(
		collaborator,
		image.NewStore(h.DB).Chown,
		key.NewStore(h.DB).Chown,
		object.NewStore(h.DB).Chown,
		variable.NewStore(h.DB).Chown,
	)
	return errors.Err(err)
}
