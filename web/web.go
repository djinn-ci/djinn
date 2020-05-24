// Package web provides utility for working with HTTP requests, responses, and
// middleware.
package web

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

type lenEncoder struct {
	w io.Writer
	l int
}

func (w *lenEncoder) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.l += n
	return n, err
}

func (w *lenEncoder) len() int64 { return int64(w.l) }

// PageURL appends the given page value to the given URL as the page query
// parameter and returns the string.
func PageURL(url *url.URL, addr string, page int64) string {
	parts := strings.Split(addr, "://")

	q := url.Query()
	q.Set("page", strconv.FormatInt(page, 10))

	url.Scheme = parts[0]
	url.Host = parts[1]
	url.RawQuery = q.Encode()

	return url.String()
}

// EncodeToLink returns the given Paginator encoded to a Link header value
// using the given URL.
func EncodeToLink(p model.Paginator, r *http.Request) string {
	addr := BaseAddress(r)

	prev := PageURL(r.URL, addr, p.Prev)
	next := PageURL(r.URL, addr, p.Next)

	return "<" + prev + ">; rel=\"prev\", <" + next + ">; rel=\"next\""
}

// BaseAddress will return the HTTP address for the given HTTP Request. This
// will return the Scheme of the current Request (http, or https), concatenated
// with the host. If the X-Forwarded-Proto, and X-Forwarded-Host headers are
// present in the Request, then they will be used for the Scheme and Host
// respectively.
func BaseAddress(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	host := r.Header.Get("X-Forwarded-Host")

	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host
}

// HTML sets the Content-Type of the given ResponseWriter to text/html, and
// writes the given content with the given status code to the writer.
func HTML(w http.ResponseWriter, content string, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(content)), 10))
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
	enc := &lenEncoder{w: ioutil.Discard}
	json.NewEncoder(enc).Encode(data)

	w.Header().Set("Content-Length", strconv.FormatInt(enc.len(), 10))
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
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(content)), 10))
	w.WriteHeader(status)
	w.Write([]byte(content))
}
