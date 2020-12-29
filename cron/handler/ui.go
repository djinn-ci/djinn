package handler

import (
	"net/http"
	"regexp"

	"github.com/andrewpillar/djinn/build"
	buildtemplate "github.com/andrewpillar/djinn/build/template"
	"github.com/andrewpillar/djinn/cron"
	crontemplate "github.com/andrewpillar/djinn/cron/template"
	"github.com/andrewpillar/djinn/errors"
	"github.com/andrewpillar/djinn/namespace"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
	"github.com/andrewpillar/djinn/web"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
)

// UI is the handler for handling UI requests made for cron job creation,
// and management.
type UI struct {
	Cron
}

// Index serves the HTML response detailing the list of cron jobs.
func (h UI) Index(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	q := r.URL.Query()

	cc, paginator, err := h.IndexWithRelations(cron.NewStore(h.DB, u), q)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrf := csrf.TemplateField(r)

	p := &crontemplate.Index{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Paginator: paginator,
		Crons:     cc,
		CSRF:      csrf,
		Search:    q.Get("search"),
	}

	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrf))
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Create serves the HTML response for creating cron jobs via the web frontend.
func (h UI) Create(w http.ResponseWriter, r *http.Request) {
	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	sess, save := h.Session(r)

	csrf := string(csrf.TemplateField(r))

	p := &crontemplate.Form{
		Form: template.Form{
			CSRF:   csrf,
			Fields: webutil.FormFields(sess),
			Errors: webutil.FormErrors(sess),
		},
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Store validates the form submitted in the given request for creating a cron
// job. If validation fails then the user is redirected back to the request
// referer, otherwise they are redirect back to the cron job index.
func (h UI) Store(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	c, f, err := h.StoreModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		switch cause {
		case namespace.ErrName:
			errs := webutil.NewErrors()
			errs.Put("namespace", cause)

			sess.AddFlash(errs, "form_errors")
			h.RedirectBack(w, r)
			return
		case namespace.ErrPermission:
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message:  "Failed to create cron: could not add to namespace",
			}, "alert")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message:  "Failed to create cron",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Cron job has been added: " + c.Name + " it will next trigger on" + c.NextRun.Format("Mon, 2 Jan 15:04 2006"),
	}, "alert")
	h.Redirect(w, r, "/cron")
}

// Show serves the HTML response for viewing an individual cron job in the given
// request.
func (h UI) Show(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	c, err := h.ShowWithRelations(r)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	q := r.URL.Query()

	bb, paginator, err := build.NewStore(h.DB).Index(
		q,
		query.Where(
			"id", "IN", cron.SelectBuild("build_id", query.Where("cron_id", "=", query.Arg(c.ID))),
		),
	)

	if err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	if err := build.LoadRelations(h.loaders, bb...); err != nil {
		h.Log.Error.Println(r.Method, r.URL.Path, errors.Err(err))
		web.HTMLError(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	csrf := csrf.TemplateField(r)

	bp := template.BasePage{
		URL:  r.URL,
		User: u,
	}
	p := &crontemplate.Show{
		BasePage: bp,
		CSRF:     csrf,
		Cron:     c,
		Builds: &buildtemplate.Index{
			BasePage:  bp,
			Paginator: paginator,
			Builds:    bb,
			Search:    q.Get("search"),
			Status:    q.Get("status"),
			Tag:       q.Get("tag"),
		},
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrf))
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Edit serves the HTML response for editing the cron job in the given request
// context.
func (h UI) Edit(w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	u, ok := user.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get user from request context")
	}

	c, ok := cron.FromContext(r.Context())

	if !ok {
		h.Log.Error.Println(r.Method, r.URL, "failed to get cron from request context")
	}

	csrf := csrf.TemplateField(r)

	p := &crontemplate.Form{
		Form: template.Form{
			CSRF:   string(csrf),
			Fields: webutil.FormFields(sess),
			Errors: webutil.FormErrors(sess),
		},
		Cron: c,
	}
	d := template.NewDashboard(p, r.URL, u, web.Alert(sess), string(csrf))
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

// Update validates the form submitted in the given request for updating a cron
// job. If validation fails then the user is redirect back to the request's
// referer, otherwise they are redirected back to the updated cron job.
func (h UI) Update(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	c, f, err := h.UpdateModel(r)

	if err != nil {
		cause := errors.Cause(err)

		if ferrs, ok := cause.(*webutil.Errors); ok {
			webutil.FlashFormWithErrors(sess, f, ferrs)
			h.RedirectBack(w, r)
			return
		}

		switch cause {
		case namespace.ErrName:
			errs := webutil.NewErrors()
			errs.Put("namespace", cause)

			sess.AddFlash(errs, "form_errors")
			h.RedirectBack(w, r)
			return
		case namespace.ErrPermission:
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to create cron job: could not add to namespace",
			}, "alert")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			sess.AddFlash(template.Alert{
				Level:   template.Danger,
				Close:   true,
				Message: "Failed to create cron job",
			}, "alert")
			h.RedirectBack(w, r)
			return
		}
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Cron job has been updated",
	}, "alert")
	h.Redirect(w, r, c.Endpoint())
}

// Destroy removes the cron job in the given request context from the database.
// This redirects back to the cron index if this was done from an individual
// cron view, otherwise it redirects back to the request's referer.
func (h UI) Destroy(w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		sess.AddFlash(template.Alert{
			Level:   template.Danger,
			Close:   true,
			Message: "Failed to delete cron job",
		}, "alert")
		h.RedirectBack(w, r)
		return
	}

	sess.AddFlash(template.Alert{
		Level:   template.Success,
		Close:   true,
		Message: "Deleted cron job",
	}, "alert")

	if matched, _ := regexp.Match("/cron/[0-9]+", []byte(r.Header.Get("Referer"))); matched {
		h.Redirect(w, r, "/cron")
		return
	}
	h.RedirectBack(w, r)
}
