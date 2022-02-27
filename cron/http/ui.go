package http

import (
	"fmt"
	"net/http"
	"regexp"

	"djinn-ci.com/alert"
	buildtemplate "djinn-ci.com/build/template"
	"djinn-ci.com/cron"
	crontemplate "djinn-ci.com/cron/template"
	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
)

type UI struct {
	*Handler
}

func (h UI) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	cc, paginator, err := h.IndexWithRelations(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := cron.LoadNamespaces(h.DB, cc...); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
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
		Search:    r.URL.Query().Get("search"),
	}

	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Create(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrf := csrf.TemplateField(r)

	p := &crontemplate.Form{
		Form: template.Form{
			CSRF:   csrf,
			Fields: webutil.FormFields(sess),
			Errors: webutil.FormErrors(sess),
		},
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

var msgfmt = "Cron job has been added: %s it will next trigger on %s"

func (h UI) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	c, f, err := h.StoreModel(u, r)

	if err != nil {
		cause := errors.Cause(err)

		errs := webutil.NewValidationErrors()

		switch err := cause.(type) {
		case webutil.ValidationErrors:
			if errs, ok := err["fatal"]; ok {
				h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
				alert.Flash(sess, alert.Danger, "Failed to create cron job")
				h.RedirectBack(w, r)
				return
			}
			errs = err
		case *namespace.PathError:
			errs.Add("namespace", err)
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to create cron job")
			h.RedirectBack(w, r)
			return
		}

		webutil.FlashFormWithErrors(sess, f, errs)
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, fmt.Sprintf(msgfmt, c.Name, c.NextRun.Format("Mon, 2 Jan 15:04 2006")))
	h.Redirect(w, r, "/cron")
}

func (h UI) Show(u *user.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if err := cron.LoadRelations(h.DB, c); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := namespace.Load(h.DB, c); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	q := r.URL.Query()

	bb, paginator, err := h.Builds.Index(q, query.Where("id", "IN", cron.SelectBuildIDs(c.ID)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
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
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Edit(u *user.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrf := csrf.TemplateField(r)

	p := &crontemplate.Form{
		Form: template.Form{
			CSRF:   csrf,
			Fields: webutil.FormFields(sess),
			Errors: webutil.FormErrors(sess),
		},
		Cron: c,
	}
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Update(u *user.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	c, f, err := h.UpdateModel(c, r)

	if err != nil {
		cause := errors.Cause(err)

		if verrs, ok := cause.(webutil.ValidationErrors); ok {
			if errs, ok := verrs["fatal"]; ok {
				h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
				alert.Flash(sess, alert.Danger, "Failed to update cron job")
				h.RedirectBack(w, r)
				return
			}

			webutil.FlashFormWithErrors(sess, f, verrs)
			h.RedirectBack(w, r)
			return
		}

		errs := webutil.NewValidationErrors()

		switch cause {
		case namespace.ErrName:
			errs.Add("namespace", cause)

			sess.AddFlash(errs, "form_errors")
			h.RedirectBack(w, r)
			return
		case namespace.ErrPermission, namespace.ErrOwner:
			alert.Flash(sess, alert.Danger, "Failed to update cron job: could not add to namespace")
			h.RedirectBack(w, r)
			return
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to update cron job")
			h.RedirectBack(w, r)
			return
		}
	}

	alert.Flash(sess, alert.Success, "Cron job has been updated")
	h.Redirect(w, r, c.Endpoint())
}

var recronuri = regexp.MustCompile("/cron/[0-9]+")

func (h UI) Destroy(u *user.User, c *cron.Cron, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteModel(r.Context(), c); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to delete cron job")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Deleted cron job")

	if recronuri.Match([]byte(r.Header.Get("Referer"))) {
		h.Redirect(w, r, "/cron")
		return
	}
	h.RedirectBack(w, r)
}

func RegisterUI(srv *server.Server) {
	user := userhttp.NewHandler(srv)

	ui := UI{
		Handler: NewHandler(srv),
	}

	sr := srv.Router.PathPrefix("/cron").Subrouter()
	sr.HandleFunc("", user.WithUser(ui.Index)).Methods("GET")
	sr.HandleFunc("/create", user.WithUser(ui.Create)).Methods("GET")
	sr.HandleFunc("", user.WithUser(ui.Store)).Methods("POST")
	sr.HandleFunc("/{cron:[0-9]+}", user.WithUser(ui.WithCron(ui.Show))).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}/edit", user.WithUser(ui.WithCron(ui.Edit))).Methods("GET")
	sr.HandleFunc("/{cron:[0-9]+}", user.WithUser(ui.WithCron(ui.Update))).Methods("PATCH")
	sr.HandleFunc("/{cron:[0-9]+}", user.WithUser(ui.WithCron(ui.Destroy))).Methods("DELETE")
	sr.Use(srv.CSRF)
}
