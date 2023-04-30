package http

import (
	"net/http"

	"djinn-ci.com/auth"
	"djinn-ci.com/server"

	"github.com/andrewpillar/webutil/v2"
)

func RegisterAPI(a auth.Authenticator, srv *server.Server) {
	h := NewHandler(srv)

	user := func(u *auth.User, w http.ResponseWriter, r *http.Request) {
		webutil.JSON(w, u, http.StatusOK)
	}

	srv.Router.PathPrefix("/").Subrouter().HandleFunc("/user", h.Restrict(a, nil, user))
}
