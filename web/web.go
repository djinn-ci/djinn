package web

import (
	"encoding/json"
	"net/http"

	"github.com/andrewpillar/thrall/template"
)

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

func JSONError(w http.ResponseWriter, message string, status int) {
	data := map[string]string{"message": message}

	JSON(w, data, status)
}

func JSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	enc := json.NewEncoder(w)
	enc.Encode(data)
}

func Text(w http.ResponseWriter, content string, status int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(content))
}
