package web

import (
	"net/http"
	"strings"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/log"
)

type Spoof struct {
	http.Handler
}

func NewSpoof(h http.Handler) Spoof {
	return Spoof{Handler: h}
}

func (h Spoof) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		typ := r.Header.Get("Content-Type")

		if strings.Contains(typ, "application/x-www-form-urlencoded") || strings.Contains(typ, "multipart/form-data") {
			// Shallow copy HTTP request so the original request body does not
			// get parsed. We don't want to accidentally discard
			// multipart/form-data requests, this can cause CSRF to fail.
			tmp := r.WithContext(r.Context())

			if err := tmp.ParseForm(); err != nil {
				log.Error.Println(errors.Err(err))
				h.Handler.ServeHTTP(w, r)
				return
			}

			method := tmp.Form.Get("_method")

			if method == "PATCH" || method == "PUT" || method == "DELETE" {
				r.Method = method
			}
		}
	}

	h.Handler.ServeHTTP(w, r)
}
