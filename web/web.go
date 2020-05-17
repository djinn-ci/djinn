// Package web provides utility for working with HTTP requests, responses, and
// middleware.
package web

import (
	"encoding/json"
	"net/http"

	"github.com/andrewpillar/thrall/template"
)

// HTML sets the Content-Type of the given ResponseWriter to text/html, and
// writes the given content with the given status code to the writer.
func HTML(w http.ResponseWriter, content string, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(content))
}

// HTMLError renders an HTML error page with the given message, and the given
// status code.
func HTMLError(w http.ResponseWriter, message string, status int) {
	p := &template.Error{
		Code:    status,
		Message: message,
	}
	HTML(w, template.Render(p), status)
}

// JSON sets the Content-Type of the given ResponseWriter to application/json,
// and encodes the given interface to JSON to the given writer, with the given
// status code.
func JSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// JSONError takes the given message, and encodes it to a JSON object with the
// given status code.
func JSONError(w http.ResponseWriter, message string, status int) {
	JSON(w, map[string]string{"message": message}, status)
}

// Text sets the Content-Type of the given ResponseWriter to text/plain, and
// writes the given content with the given status code to the writer.
func Text(w http.ResponseWriter, content string, status int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(content))
}
