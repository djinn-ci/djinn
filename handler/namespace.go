package handler

import (
	"net/http"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/namespace"
)

type Namespace struct {
	Handler
}

func NewNamespace(h Handler) Namespace {
	return Namespace{Handler: h}
}

func (h Namespace) Index(w http.ResponseWriter, r *http.Request) {
	u, err := h.UserFromRequest(r)

	if err != nil {
		log.Error.Println(errors.Err(err))
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	namespaces, err := u.Namespaces()

	if err != nil {
		log.Error.Println(errors.Err(err))
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	p := &namespace.IndexPage{
		Namespaces: namespaces,
	}

	d := template.NewDashboard(p, r.URL.RequestURI(), u)

	html(w, template.Render(d), http.StatusOK)
}
