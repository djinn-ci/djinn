package http

import (
	"net/http"

	"djinn-ci.com/server"
	"djinn-ci.com/user"

	"github.com/andrewpillar/webutil"
)

func RegisterAPI(prefix string, srv *server.Server) {
	h := NewHandler(srv)

	auth := srv.Router.PathPrefix("/").Subrouter()
	auth.HandleFunc("/user", h.WithUser(func(u *user.User, w http.ResponseWriter, r *http.Request) {
		webutil.JSON(w, u.JSON(webutil.BaseAddress(r)+"/"+prefix), http.StatusOK)
	}))
}
