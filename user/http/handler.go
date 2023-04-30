package http

import (
	"djinn-ci.com/database"
	"djinn-ci.com/oauth2"
	"djinn-ci.com/provider"
	"djinn-ci.com/server"
	"djinn-ci.com/user"
)

type Handler struct {
	*server.Server

	Users     user.Store
	Tokens    *database.Store[*oauth2.Token]
	Providers *provider.Store
}

func NewHandler(srv *server.Server) *Handler {
	return &Handler{
		Server: srv,
		Users:  user.Store{Store: user.NewStore(srv.DB)},
		Tokens: oauth2.NewTokenStore(srv.DB),
		Providers: &provider.Store{
			Store:  provider.NewStore(srv.DB),
			AESGCM: srv.AESGCM,
		},
	}
}
