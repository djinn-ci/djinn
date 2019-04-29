package web

import (
	"net"
	"net/http"
)

type SecureRedirect struct {
	listen string
}

func NewSecureRedirect(listen string) SecureRedirect {
	return SecureRedirect{
		listen:  listen,
	}
}

func (h SecureRedirect) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	url := "https://"

	_, securePort, _ := net.SplitHostPort(h.listen)

	host, _, err := net.SplitHostPort(r.Host)

	if err != nil {
		host = r.Host
	}

	url += host

	if securePort != "" {
		url += ":"
		url += securePort
	}

	url += r.URL.RequestURI()

	w.Header().Set("Connection", "close")

	http.Redirect(w, r, url, http.StatusMovedPermanently)
}
