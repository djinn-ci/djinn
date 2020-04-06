package web

import (
	"net"
	"net/http"

	"github.com/andrewpillar/thrall/log"
)

type Log struct {
	http.Handler
}

type logResponseWriter struct {
	http.ResponseWriter

	status int
	size   int
}

func NewLog(h http.Handler) Log {
	return Log{Handler: h}
}

func newLogResponseWriter(w http.ResponseWriter) *logResponseWriter {
	return &logResponseWriter{ResponseWriter: w}
}


func (l *logResponseWriter) Write(b []byte) (int, error) {
	n, err := l.ResponseWriter.Write(b)
	l.size += n
	return n, err
}

func (l *logResponseWriter) WriteHeader(status int) {
	l.ResponseWriter.WriteHeader(status)
	l.status = status
}

func (h Log) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lrw := newLogResponseWriter(w)

	h.Handler.ServeHTTP(lrw, r)

	username := "-"

	if r.URL.User != nil {
		if name := r.URL.User.Username(); name != "" {
			username = name
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)

	if err != nil {
		host = r.RemoteAddr
	}

	if r.Header.Get("X-Real-Ip") != "" {
		host = r.Header.Get("X-Real-Ip")
	}

	uri := r.RequestURI

	if r.ProtoMajor == 2 && r.Method == "CONNECT" {
		uri = r.Host
	}

	if uri == "" {
		uri = r.URL.RequestURI()
	}
	log.Info.Println(host, "-", username, r.Method, uri, r.Proto, lrw.status, lrw.size)
}
