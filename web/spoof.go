package web

import (
	"net/http"
	"strings"
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
			method := r.PostFormValue("_method")

			if method == "PATCH" || method == "PUT" || method == "DELETE" {
				r.Method = method
			}
		}
	}

	h.Handler.ServeHTTP(w, r)
}
