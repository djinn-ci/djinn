package handler

import (
	"net/http"

	"djinn-ci.com/crypto"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/oauth2"
	oauth2template "djinn-ci.com/oauth2/template"
	"djinn-ci.com/template"
	"djinn-ci.com/user"
	usertemplate "djinn-ci.com/user/template"
	"djinn-ci.com/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

// App handles serving responses to managing an OAuth app that a user has
// created for OAuth access to the API.
type App struct {
	web.Handler

	block *crypto.Block
	apps  *oauth2.AppStore
}

func NewApp(h web.Handler, block *crypto.Block) App {
	return App{
		Handler: h,
		block:   block,
		apps:    oauth2.NewAppStore(h.DB),
	}
}

func (h App) appFromRequest(r *http.Request) (*oauth2.App, error) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		return nil, errors.New("failed to get user from context")
	}

	clientId := mux.Vars(r)["app"]

	a, err := oauth2.NewAppStore(h.DB, u).Get(query.Where("client_id", "=", query.Arg(clientId)))

	if err != nil {
		return a, errors.Err(err)
	}

	if a.IsZero() {
		return a, database.ErrNotFound
	}
	return a, nil
}

// Index serves the HTML response detailing the list of OAuth apps for the
// current user.
func (h App) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	aa, err := oauth2.NewAppStore(h.DB, u).All()

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrf := csrf.TemplateField(r)

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}

	p := &usertemplate.Settings{
		BasePage: bp,
		Section: &oauth2template.AppIndex{
			BasePage: bp,
			Apps:     aa,
		},
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Create serves the HTML response for creating a new OAuth app.
func (h App) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	csrf := csrf.TemplateField(r)

	p := &oauth2template.AppForm{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Store validates the form submitted in the given request for creating an
// OAuth app. If validation fails then the user is redirected back to the
// request referer, otherwise they are redirect back to the OAuth app index.
func (h App) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	apps := oauth2.NewAppStoreWithBlock(h.DB, h.block, u)

	f := &oauth2.AppForm{
		Apps: apps,
	}

	if err := webutil.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to create app",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	a, err := apps.Create(f.Name, f.Description, f.HomepageURI, f.RedirectURI)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to create app",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Created OAuth app " + a.Name,
	}, "alert")
	h.Redirect(w, r, "/settings/apps")
}

// Show serves the individual HTML response for viewing an individual OAuth
// app.
func (h App) Show(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	a, err := h.appFromRequest(r)

	if err != nil {
		if err == database.ErrNotFound {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	dec, err := h.block.Decrypt(a.ClientSecret)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	a.ClientSecret = dec

	csrf := csrf.TemplateField(r)

	p := &oauth2template.AppForm{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
		App: a,
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Update validates the form submitted in the given request for updating an
// OAuth app. If validation fails then the user is redirect back to the
// request's referer, otherwise they are redirected back to the updated OAuth
// app. If the base of the requested URL path is "/revoke" then this will
// revoke all of the access tokens for this app. If the base of the path is
// "/reset" then it will generate a new client secret.
func (h App) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	a, err := h.appFromRequest(r)

	if err != nil {
		if err == database.ErrNotFound {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to update app",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	base := webutil.BasePath(r.URL.Path)

	if base == "revoke" {
		if err := oauth2.NewTokenStore(h.DB).Revoke(a.ID); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to revoke tokens",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}

		sess.AddFlash(template.Alert{
			Level:   template.Success,
			Close:   true,
			Message: "App tokens revoked",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	if base == "reset" {
		if err := oauth2.NewAppStoreWithBlock(h.DB, h.block).Reset(a.ID); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to reset secret",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}

		sess.AddFlash(template.Alert{
			Level:   template.Success,
			Close:   true,
			Message: "App client secret reset",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	f := &oauth2.AppForm{
		Apps: oauth2.NewAppStore(h.DB, u),
		App:  a,
	}

	if err := webutil.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to update app",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := h.apps.Update(a.ID, f.Name, f.Description, f.HomepageURI, f.RedirectURI); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to update app",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "App changes have been saved",
	}, "alert")
	h.Redirect(w, r, a.Endpoint())
}
