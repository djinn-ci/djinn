package handler

import (
	"encoding/hex"
	"net/http"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/oauth2"
	oauth2template "github.com/andrewpillar/djinn/oauth2/template"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	usertemplate "github.com/andrewpillar/djinn/user/template"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

// Connection handles serving the responses to managing the OAuth apps that
// have connected to a user's account.
type Connection struct {
	web.Handler

	apps   *oauth2.AppStore
	tokens *oauth2.TokenStore
}

func NewConnection(h web.Handler) Connection {
	return Connection{
		Handler: h,
		apps:    oauth2.NewAppStore(h.DB),
		tokens:  oauth2.NewTokenStore(h.DB),
	}
}

func (h Connection) getToken(r *http.Request) (*oauth2.Token, error) {
	clientId, err := hex.DecodeString(mux.Vars(r)["id"])

	if err != nil {
		return nil, errors.New("invalid hex encoding")
	}

	a, err := h.apps.Get(query.Where("client_id", "=", query.Arg(clientId)))

	if err != nil {
		return nil, errors.Err(err)
	}

	if a.IsZero() {
		return nil, errors.Err(err)
	}

	u, err := h.Users.Get(query.Where("id", "=", query.Arg(a.UserID)))

	if err != nil {
		return nil, errors.Err(err)
	}

	a.User = u

	t, err := oauth2.NewTokenStore(h.DB, a).Get()
	return t, errors.Err(err)
}

// Index serves the HTML response detailing the list of connected apps.
func (h Connection) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	tt, err := oauth2.NewTokenStore(h.DB, u).All(query.Where("app_id", "IS NOT", query.Lit("NULL")))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	mm := database.ModelSlice(len(tt), oauth2.TokenModel(tt))

	aa, err := h.apps.All(query.Where("id", "IN", database.List(database.MapKey("app_id", mm)...)))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	uu, err := h.Users.All(query.Where("id", "IN", database.List(database.MapKey("user_id", mm)...)))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	apps := make(map[int64]*oauth2.App)
	users := make(map[int64]*user.User)

	for _, u := range uu {
		users[u.ID] = u
	}

	for _, a := range aa {
		a.User = users[a.UserID]
		apps[a.ID] = a
	}

	for _, t := range tt {
		t.App = apps[t.AppID.Int64]
	}

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}

	p := &usertemplate.Settings{
		BasePage: bp,
		Section: &oauth2template.ConnectionIndex{
			BasePage: bp,
			Tokens:   tt,
		},
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Show serves the HTML response detailing an individual connected app.
func (h Connection) Show(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	t, err := h.getToken(r)

	if err != nil {
		if err.Error() == "invalid hex encoding" {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if t.IsZero() {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	csrf := csrf.TemplateField(r)

	p := &oauth2template.Connection{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		CSRF:  csrf,
		Token: t,
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrf))
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Destroy will delete the connection from the user's account. This will delete
// the underlying access token that was generated for the user from the
// database.
func (h Connection) Destroy(w http.ResponseWriter, r *http.Request) {
	t, err := h.getToken(r)

	if err != nil {
		if err.Error() == "invalid hex encoding" {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := h.tokens.Delete(t.ID); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}
	h.Redirect(w, r, "/settings/connections")
}
