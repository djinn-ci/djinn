// Package web provides utility for working with HTTP requests, responses, and
// middleware.
package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"djinn-ci.com/database"
	"djinn-ci.com/template"

	"github.com/andrewpillar/webutil"

	"github.com/gorilla/sessions"
)

// Alert returns the first alert that was flashed to the session. If no alert
// exists, then an empty alert is returned instead.
func Alert(sess *sessions.Session) template.Alert {
	val := sess.Flashes("alert")

	if val == nil {
		return template.Alert{}
	}
	return val[0].(template.Alert)
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
	addr := webutil.BaseAddress(r)

	prev := PageURL(r.URL, addr, p.Prev)
	next := PageURL(r.URL, addr, p.Next)

	return "<" + prev + ">; rel=\"prev\", <" + next + ">; rel=\"next\""
}

// HTMLError renders an HTML error page with the given message, and the given
// status code.
func HTMLError(w http.ResponseWriter, message string, status int) {
	p := &template.Error{
		Code:    status,
		Message: message,
	}
	webutil.HTML(w, template.Render(p), status)
}

// JSONError takes the given message, and encodes it to a JSON object with the
// given status code.
func JSONError(w http.ResponseWriter, message string, status int) {
	webutil.JSON(w, map[string]string{"message": message}, status)
}
