package ui

import (
	"net/http"

	"github.com/andrewpillar/thrall/web"
)

type Collaborator struct {
	web.Handler
}

func (h Collaborator) Store(w http.ResponseWriter, r *http.Request) {

}

func (h Collaborator) Destroy(w http.ResponseWriter, r *http.Request) {

}
