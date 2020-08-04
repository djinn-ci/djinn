package handler

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/andrewpillar/thrall/block"
	"github.com/andrewpillar/thrall/build"
	buildtemplate "github.com/andrewpillar/thrall/build/template"
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/namespace"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/user"
	"github.com/andrewpillar/thrall/web"

	"github.com/andrewpillar/query"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type UI struct {
	Build

	Job       JobUI
	Tag       TagUI
	Artifacts block.Store
}

type JobUI struct {
	Job
}

type TagUI struct {
	Tag
}

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

	if err := build.NewStoreWithHasher(h.DB, h.Hasher).Submit(h.Queues[b.Manifest.Driver["type"]], b); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Danger("Failed to create build"), "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Success("Build submitted: #"+strconv.FormatInt(b.ID, 10)), "alert")
	h.Redirect(w, r, b.Endpoint())
}

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
			Build: b,
			Tags:  tt,
		}
	}

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrfField))
	save(r, w)
	web.HTML(w, template.Render(d), http.StatusOK)
}

// Download will serve the contents of an artifact.
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
