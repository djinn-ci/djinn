package handler

import (
	"encoding/hex"
	"net/http"

	"github.com/andrewpillar/djinn/crypto"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/oauth2"
	oauth2template "github.com/andrewpillar/djinn/oauth2/template"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	usertemplate "github.com/andrewpillar/djinn/user/template"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

// App handles serving responses to managing an OAuth app that a user has
// created for OAuth access to the API.
type App struct {
	web.Handler

	block *crypto.Block
	apps  *oauth2.AppStore

	//	// Block is the block cipher to use for encrypting client secrets.
	//	Block *crypto.Block
	//
	//	// Apps is the app store to use for the deletion of OAuth apps.
	//	Apps  *oauth2.AppStore
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

	clientId, err := hex.DecodeString(mux.Vars(r)["app"])

	if err != nil {
		return nil, database.ErrNotFound
	}

	a, err := oauth2.NewAppStore(h.DB, u).Get(query.Where("client_id", "=", clientId))

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

	csrf := string(csrf.TemplateField(r))

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
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Create serves the HTML response for creating a new OAuth app.
func (h App) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	csrfField := string(csrf.TemplateField(r))

	p := &oauth2template.AppForm{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: web.FormErrors(sess),
			Fields: web.FormFields(sess),
		},
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
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

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create app"), "alert")
		h.RedirectBack(w, r)
		return
	}

	a, err := apps.Create(f.Name, f.Description, f.HomepageURI, f.RedirectURI)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create app"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Created OAuth App "+a.Name), "alert")
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

	csrfField := string(csrf.TemplateField(r))

	p := &oauth2template.AppForm{
		Form: template.Form{
			CSRF:   csrfField,
			Errors: web.FormErrors(sess),
			Fields: web.FormFields(sess),
		},
		App: a,
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrfField)
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
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
		sess.AddFlash(template.Danger("Failed to update app"), "alert")
		h.RedirectBack(w, r)
		return
	}

	base := web.BasePath(r.URL.Path)

	if base == "revoke" {
		if err := oauth2.NewTokenStore(h.DB).Revoke(a.ID); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to revoke tokens"), "alert")
			h.RedirectBack(w, r)
			return
		}

		sess.AddFlash(template.Success("App tokens revoked"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if base == "reset" {
		if err := oauth2.NewAppStoreWithBlock(h.DB, h.block).Reset(a.ID); err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to reset secret"), "alert")
			h.RedirectBack(w, r)
			return
		}

		sess.AddFlash(template.Success("App client secret reset"), "alert")
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

	if err := form.UnmarshalAndValidate(f, r); err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update app"), "alert")
		h.RedirectBack(w, r)
		return
	}

	if err := h.apps.Update(a.ID, f.Name, f.Description, f.HomepageURI, f.RedirectURI); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to update app"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("App changes have been saved"), "alert")
	h.Redirect(w, r, a.Endpoint())
}
