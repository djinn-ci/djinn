package handler

import "net/http"

type Page struct {
	Handler
}

func NewPage(h Handler) Page {
	return Page{Handler: h}
}

func (h Page) Home(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Thrall CI server - Home\n"))
}
