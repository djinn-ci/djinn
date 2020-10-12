package handler

import (
	"net/http"

	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/image"
	"github.com/andrewpillar/djinn/key"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/object"
	"github.com/andrewpillar/djinn/variable"
	"github.com/andrewpillar/djinn/web"

	"github.com/gorilla/mux"
)

type Collaborator struct {
	web.Handler
}

func (h Collaborator) DeleteModel(r *http.Request) error {
	n, ok := namespace.FromContext(r.Context())

	if !ok {
		return errors.New("no namespace in request context")
	}

	err := namespace.NewCollaboratorStore(h.DB, n).Delete(
		mux.Vars(r)["collaborator"],
		image.NewStore(h.DB).Chown,
		key.NewStore(h.DB).Chown,
		object.NewStore(h.DB).Chown,
		variable.NewStore(h.DB).Chown,
	)
	return errors.Err(err)
}
