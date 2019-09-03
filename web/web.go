package web

import (
	"net/http"

	"github.com/andrewpillar/thrall/template"
)

var PageLimit int64 = 1

func HTML(w http.ResponseWriter, content string, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(content))
}

func HTMLError(w http.ResponseWriter, message string, status int) {
	p := &template.Error{
		Code:    status,
		Message: message,
	}

	HTML(w, template.Render(p), status)
}

func Text(w http.ResponseWriter, content string, status int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(content))
}
