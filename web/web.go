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

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/template"

	"github.com/gorilla/sessions"
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

// Alert returns the first alert that was flashed to the session. If no alert
// exists, then an empty alert is returned instead.
func Alert(sess *sessions.Session) template.Alert {
	val := sess.Flashes("alert")

	if val == nil {
		return template.Alert{}
	}
	return val[0].(template.Alert)
}

// FormErrors returns the map that has been flashed to the session under the
// form_errors key. If the key does not exist, then an empty form.Errors is
// returned instead.
func FormErrors(sess *sessions.Session) form.Errors {
	val := sess.Flashes("form_errors")

	if val == nil {
		return form.NewErrors()
	}
	return val[0].(form.Errors)
}

// FormFields returns the map that has been flashed to the session under the
// form_fields key. If the key does not exist, then an empty map is returned
// instead.
func FormFields(sess *sessions.Session) map[string]string {
	val := sess.Flashes("form_fields")

	if val == nil {
		return map[string]string{}
	}
	return val[0].(map[string]string)
}

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
func EncodeToLink(p database.Paginator, r *http.Request) string {
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

// BasePath returns the last element of the given path. This will split the
// path using the "/" separator. If the path is empty BasePath returns "/".
func BasePath(path string) string {
	if path == "" {
		return "/"
	}

	parts := strings.Split(path, "/")
	return strings.TrimSuffix(parts[len(parts)-1], "/")
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

// FlashFormWithErrors will flash the given form.Form, and form.Errors to the
// given session under the "form_fields" and "form_errors" keys respectively.
func FlashFormWithErrors(sess *sessions.Session, f form.Form, errs form.Errors) {
	sess.AddFlash(f.Fields(), "form_fields")
	sess.AddFlash(errs, "form_errors")
}

// Validate form takes the given form interface and calls Validate on it. If
// a form.Errors type is returned from validation, then this will be flashed to
// the given session, along with the contents of the form. If the given session
// is nil, then no data will be flashed. This will return any errors from the
// subsequent Validate call.
func ValidateForm(f form.Form, sess *sessions.Session) error {
	if err := f.Validate(); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			if sess != nil {
				sess.AddFlash(ferrs, "form_errors")
				sess.AddFlash(f.Fields(), "form_fields")
			}
			return ferrs
		}

		if sess != nil {
			sess.AddFlash(template.Danger("Failed to validate form"), "alert")
		}
		return errors.Err(err)
	}
	return nil
}
