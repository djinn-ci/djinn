package handler

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/andrewpillar/djinn/block"
	"github.com/andrewpillar/djinn/build"
	buildtemplate "github.com/andrewpillar/djinn/build/template"
	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/form"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

// UI is the handler for handling UI requests made for build creation,
// submission, and retrieval.
type UI struct {
	Build

	// Artifacts is the block storage to use for retrieving artifacts that
	// are downloaded through the web frontend.
	Artifacts block.Store
}

// JobUI is the handler for handling UI requests made for managing build jobs.
type JobUI struct {
	Job
}

// TagUI is the handler for handling UI requests made for managing build tags.
type TagUI struct {
	Tag
}

// Index serves the HTML response detailing the list of builds.
func (h UI) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "no user in request context")
	}

	bb, paginator, err := h.IndexWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()

	p := &buildtemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Paginator: paginator,
		Builds:    bb,
		Search:    q.Get("search"),
		Status:    q.Get("status"),
		Tag:       q.Get("tag"),
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Create serves the HTML response for submitting builds via the web frontend.
func (h UI) Create(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	p := &buildtemplate.Create{
		Form: template.Form{
			CSRF:   string(csrf.TemplateField(r)),
			Errors: web.FormErrors(sess),
			Fields: web.FormFields(sess),
		},
	}

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Store validates the form submitted in the given request for submitting a
// build. If validation fails then the user is redirected back to the previous
// request, otherwise they are redirect back to the newly submitted build.
func (h UI) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	b, f, err := h.StoreModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(form.Errors); ok {
			web.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		switch cause {
		case build.ErrDriver:
			errs := form.NewErrors()
			errs.Put("manifest", cause)

			sess.AddFlash(errs, "form_errors")
			h.RedirectBack(w, r)
			return
		case namespace.ErrName:
			errs := form.NewErrors()
			errs.Put("manifest", errors.New("Namespace name can only contain letters and numbers"))

			sess.AddFlash(f.Fields(), "form_fields")
			sess.AddFlash(errs, "form_errors")
			h.RedirectBack(w, r)
			return
		case namespace.ErrPermission:
			sess.AddFlash(template.Danger("Failed to create build: could not add to namespace"), "alert")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Danger("Failed to create build"), "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	if err := build.NewStoreWithHasher(h.DB, h.Hasher).Submit(h.Queues[b.Manifest.Driver["type"]], web.BaseAddress(r), b); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create build"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Build submitted: #"+strconv.FormatInt(b.ID, 10)), "alert")
	h.Redirect(w, r, b.Endpoint())
}

// Show serves the HTML response for viewing an individual build in the given
// request. This serves different responses based on the base path of the
// request URL.
func (h UI) Show(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	b, err := h.ShowWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	base := web.BasePath(r.URL.Path)
	csrfField := csrf.TemplateField(r)

	p := &buildtemplate.Show{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Build: b,
		CSRF:  string(csrf.TemplateField(r)),
	}

	switch base {
	case "manifest":
		p.Section = &buildtemplate.Manifest{Build: b}
	case "raw":
		parts := strings.Split(r.URL.Path, "/")
		attr := parts[len(parts)-2]

		if attr == "manifest" {
			web.Text(w, b.Manifest.String(), http.StatusOK)
			return
		}
		if attr == "output" {
			web.Text(w, b.Output.String, http.StatusOK)
			return
		}
	case "objects":
		oo, err := h.objectsWithRelations(b)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &buildtemplate.Objects{
			Build:   b,
			Objects: oo,
		}
	case "artifacts":
		search := r.URL.Query().Get("search")

		aa, err := build.NewArtifactStore(h.DB, b).All(database.Search("name", search))

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &buildtemplate.Artifacts{
			URL:       r.URL,
			Search:    search,
			Build:     b,
			Artifacts: aa,
		}
	case "variables":
		vv, err := h.variablesWithRelations(b)

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &buildtemplate.Variables{
			Build:     b,
			Variables: vv,
		}
	case "keys":
		kk, err := build.NewKeyStore(h.DB, b).All()

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &buildtemplate.Keys{
			Build: b,
			Keys:  kk,
		}
	case "tags":
		tt, err := build.NewTagStore(h.DB, b).All()

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		mm := database.ModelSlice(len(tt), build.TagModel(tt))

		err = h.Users.Load("id", database.MapKey("user_id", mm), database.Bind("user_id", "id", mm...))

		if err != nil {
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
			return
		}

		p.Section = &buildtemplate.Tags{
			CSRF:  string(csrfField),
			User:  u,
			Build: b,
			Tags:  tt,
		}
	}

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrfField))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Download will serve the contents of a build's artifact in the given request.
// Depending on the MIME type of the artifact will depend on whether the content
// is served in the browser directly, or downloaded.
func (h UI) Download(w http.ResponseWriter, r *http.Request) {
	b, ok := build.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "Failed to get build from request context")
	}

	id, _ := strconv.ParseInt(mux.Vars(r)["artifact"], 10, 64)

	a, err := build.NewArtifactStore(h.DB, b).Get(query.Where("id", "=", id))

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if a.IsZero() || a.Name != mux.Vars(r)["name"] {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	rec, err := h.Artifacts.Open(a.Hash)

	if err != nil {
		if os.IsNotExist(errors.Cause(err)) {
			web.HTMLError(w, "Not found", http.StatusNotFound)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	defer rec.Close()
	http.ServeContent(w, r, a.Name, a.CreatedAt, rec)
}

// Destroy kills the build. On success this will redirect back, on failure it
// will redirect back with an error message.
func (h UI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.Kill(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to kill build"), "alert")
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Success("Build killed"), "alert")
	h.RedirectBack(w, r)
}

// Store adds the tags in the given form submitted in the request. Upon success
// this will redirect back to the previous page. On a failure this will redirect
// back with an error message.
func (h TagUI) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if _, err := h.StoreModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to tag build"), "alert")
		h.RedirectBack(w, r)
		return
	}
	h.RedirectBack(w, r)
}

// Destroy will remove the specified tag from the build in the given request.
// On success this will redirect back, otherwise it will redirect back with an
// error message.
func (h TagUI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to tag build"), "alert")
		h.RedirectBack(w, r)
		return
	}
	sess.AddFlash(template.Success("Tag has been deleted"), "alert")
	h.RedirectBack(w, r)
}

// Show will serve the HTML response for an individual build job. If the base
// of the URL path is "/raw", then a "text/plain" response is served with the
// output of the job.
func (h JobUI) Show(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	j, err := h.ShowWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if j.IsZero() {
		web.HTMLError(w, "Not found", http.StatusNotFound)
		return
	}

	if web.BasePath(r.URL.Path) == "raw" {
		web.Text(w, j.Output.String, http.StatusOK)
		return
	}

	p := &buildtemplate.Job{
		BasePage: template.BasePage{URL: r.URL},
		Job:      j,
	}

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrf.TemplateField(r)))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}
