package handler

import (
	"net/http"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

type Handler struct {
	sc    *securecookie.SecureCookie
	store  sessions.Store
}

func New(sc *securecookie.SecureCookie, store sessions.Store) Handler {
	return Handler{sc: sc, store: store}
}

func (h Handler) Home(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Thrall CI server\n"))
}
