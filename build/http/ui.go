package http

import (
	"net/http"
	"strconv"
	"strings"

	"djinn-ci.com/alert"
	"djinn-ci.com/build"
	buildtemplate "djinn-ci.com/build/template"
	"djinn-ci.com/database"
	"djinn-ci.com/driver"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/namespace"
	"djinn-ci.com/server"
	"djinn-ci.com/template"
	"djinn-ci.com/user"
	userhttp "djinn-ci.com/user/http"
	"djinn-ci.com/variable"
	variablehttp "djinn-ci.com/variable/http"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
)

type UI struct {
	*Handler
}

func (h UI) Index(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	bb, paginator, err := h.IndexWithRelations(u, r)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := build.LoadNamespaces(h.DB, bb...); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
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
	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf.TemplateField(r))
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Create(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	csrf := csrf.TemplateField(r)

	p := &buildtemplate.Create{
		Form: template.Form{
			CSRF:   csrf,
			Errors: webutil.FormErrors(sess),
			Fields: webutil.FormFields(sess),
		},
	}

	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Store(u *user.User, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	b, f, err := h.StoreModel(u, r)

	if err != nil {
		cause := errors.Cause(err)

		errs := webutil.NewValidationErrors()

		switch err := cause.(type) {
		case webutil.ValidationErrors:
			if errs, ok := err["fatal"]; ok {
				h.Log.Error.Println(r.Method, r.URL, errors.Slice(errs))
				alert.Flash(sess, alert.Danger, "Failed to submit build")
				h.RedirectBack(w, r)
				return
			}
			errs = err
		case *namespace.PathError:
			errs.Add("namespace", err)
		case *driver.Error:
			errs.Add("manifest", err)
		default:
			h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
			alert.Flash(sess, alert.Danger, "Failed to submit build")
			h.RedirectBack(w, r)
			return
		}

		webutil.FlashFormWithErrors(sess, f, errs)
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Build submitted: #"+strconv.FormatInt(b.Number, 10))
	h.Redirect(w, r, b.Endpoint())
}

func (h UI) Show(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	if err := build.LoadRelations(h.DB, b); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if err := build.LoadNamespaces(h.DB, b); err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	base := webutil.BasePath(r.URL.Path)
	csrf := csrf.TemplateField(r)

	p := &buildtemplate.Show{
		BasePage: template.BasePage{
			URL:  r.URL,
			User: u,
		},
		Build: b,
		CSRF:  csrf,
	}

	switch base {
	case "manifest":
		p.Section = &buildtemplate.Manifest{Build: b}
	case "raw":
		parts := strings.Split(r.URL.Path, "/")
		attr := parts[len(parts)-2]

		if attr == "manifest" {
			webutil.Text(w, b.Manifest.String(), http.StatusOK)
			return
		}
		if attr == "output" {
			webutil.Text(w, b.Output.String, http.StatusOK)
			return
		}
	case "objects":
		oo, err := h.objectsWithRelations(b)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		p.Section = &buildtemplate.Objects{
			Build:   b,
			Objects: oo,
		}
	case "artifacts":
		search := r.URL.Query().Get("search")

		aa, err := h.Artifacts.All(
			query.Where("build_id", "=", query.Arg(b.ID)),
			database.Search("name", search),
			query.OrderAsc("name"),
		)

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		for _, a := range aa {
			a.Build = b
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
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		unmasked := variablehttp.Unmasked(sess)

		for _, v := range vv {
			if _, ok := unmasked[v.VariableID.Int64]; ok && v.Masked {
				if err := variable.Unmask(h.AESGCM, v.Variable); err != nil {
					alert.Flash(sess, alert.Danger, "Could not unmask variable")
					h.Log.Error.Println(r.Method, r.URL, "could not unmask variable", errors.Err(err))
				}
				continue
			}

			if v.Masked {
				v.Value = variable.MaskString
			}
		}

		p.Section = &buildtemplate.Variables{
			CSRF:      csrf,
			User:      u,
			Build:     b,
			Unmasked:  unmasked,
			Variables: vv,
		}
	case "keys":
		kk, err := h.Keys.All(query.Where("build_id", "=", query.Arg(b.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		p.Section = &buildtemplate.Keys{
			Build: b,
			Keys:  kk,
		}
	case "tags":
		canModify := u.ID > 0 && u.ID == b.UserID

		if b.NamespaceID.Valid {
			err := b.Namespace.IsCollaborator(h.DB, u.ID)

			if err != nil {
				if !errors.Is(err, namespace.ErrPermission) {
					h.InternalServerError(w, r, errors.Err(err))
					return
				}
			}
			canModify = !errors.Is(err, namespace.ErrPermission)
		}

		tt, err := h.Tags.All(query.Where("build_id", "=", query.Arg(b.ID)))

		if err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		mm := make([]database.Model, 0, len(tt))

		for _, t := range tt {
			t.Build = b
			mm = append(mm, t)
		}

		if err := h.Users.Load("user_id", "id", mm...); err != nil {
			h.InternalServerError(w, r, errors.Err(err))
			return
		}

		p.Section = &buildtemplate.Tags{
			CSRF:      csrf,
			User:      u,
			Build:     b,
			CanModify: canModify,
			Tags:      tt,
		}
	}

	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf)
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Destroy(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := b.Kill(h.Redis); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to kill build")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Build killed")
	h.RedirectBack(w, r)
}

func (h UI) ShowJob(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	sess, save := h.Session(r)

	j, ok, err := h.Jobs.Get(
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("name", "=", query.Arg(mux.Vars(r)["name"])),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	if webutil.BasePath(r.URL.Path) == "raw" {
		webutil.Text(w, j.Output.String, http.StatusOK)
		return
	}

	j.Stage, _, err = h.Stages.Get(query.Where("id", "=", query.Arg(j.StageID)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	b.User, _, err = h.Users.Get(query.Where("id", "=", query.Arg(b.UserID)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	b.Trigger, _, err = h.Triggers.Get(query.Where("build_id", "=", query.Arg(b.ID)))

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	j.Build = b

	p := &buildtemplate.Job{
		BasePage: template.BasePage{URL: r.URL},
		Build:    b,
		Job:      j,
	}

	d := template.NewDashboard(p, r.URL, u, alert.First(sess), csrf.TemplateField(r))
	save(r, w)
	webutil.HTML(w, template.Render(d), http.StatusOK)
}

func (h UI) Download(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	a, ok, err := h.Artifacts.Get(
		query.Where("build_id", "=", query.Arg(b.ID)),
		query.Where("name", "=", query.Arg(name)),
	)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	if !ok {
		h.NotFound(w, r)
		return
	}

	store, err := h.Artifacts.Partition(b.UserID)

	if err != nil {
		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	rec, err := store.Open(a.Hash)

	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			h.NotFound(w, r)
			return
		}

		h.InternalServerError(w, r, errors.Err(err))
		return
	}

	defer rec.Close()
	http.ServeContent(w, r, a.Name, a.CreatedAt, rec)
}

func (h UI) StoreTag(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if _, err := h.StoreTagModel(u, b, r); err != nil {
		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to tag build")
		h.RedirectBack(w, r)
		return
	}
	h.RedirectBack(w, r)
}

func (h UI) DestroyTag(u *user.User, b *build.Build, w http.ResponseWriter, r *http.Request) {
	sess, _ := h.Session(r)

	if err := h.DeleteTagModel(b, mux.Vars(r)); err != nil {
		if errors.Is(err, database.ErrNotFound) {
			h.NotFound(w, r)
			return
		}

		h.Log.Error.Println(r.Method, r.URL, errors.Err(err))
		alert.Flash(sess, alert.Danger, "Failed to delete tag")
		h.RedirectBack(w, r)
		return
	}

	alert.Flash(sess, alert.Success, "Tag has been deleted")
	h.RedirectBack(w, r)
}

func RegisterUI(srv *server.Server) {
	user := userhttp.NewHandler(srv)

	ui := UI{
		Handler: NewHandler(srv),
	}

	auth := srv.Router.PathPrefix("/").Subrouter()
	auth.HandleFunc("/builds", user.WithUser(ui.Index)).Methods("GET")
	auth.HandleFunc("/builds/create", user.WithUser(ui.Create)).Methods("GET")
	auth.HandleFunc("/builds", user.WithUser(ui.Store)).Methods("POST")
	auth.Use(srv.CSRF)

	sr := srv.Router.PathPrefix("/b/{username}/{build:[0-9]+}").Subrouter()
	sr.HandleFunc("", user.WithOptionalUser(ui.WithBuild(ui.Show))).Methods("GET")
	sr.HandleFunc("", user.WithUser(ui.WithBuild(ui.Destroy))).Methods("DELETE")
	sr.HandleFunc("/manifest", user.WithOptionalUser(ui.WithBuild(ui.Show))).Methods("GET")
	sr.HandleFunc("/manifest/raw", user.WithOptionalUser(ui.WithBuild(ui.Show))).Methods("GET")
	sr.HandleFunc("/output/raw", user.WithOptionalUser(ui.WithBuild(ui.Show))).Methods("GET")
	sr.HandleFunc("/objects", user.WithOptionalUser(ui.WithBuild(ui.Show))).Methods("GET")
	sr.HandleFunc("/variables", user.WithOptionalUser(ui.WithBuild(ui.Show))).Methods("GET")
	sr.HandleFunc("/keys", user.WithOptionalUser(ui.WithBuild(ui.Show))).Methods("GET")
	sr.HandleFunc("/jobs/{name}", user.WithOptionalUser(ui.WithBuild(ui.ShowJob))).Methods("GET")
	sr.HandleFunc("/jobs/{name}/output/raw", user.WithOptionalUser(ui.WithBuild(ui.ShowJob))).Methods("GET")
	sr.HandleFunc("/artifacts", user.WithOptionalUser(ui.WithBuild(ui.Show))).Methods("GET")
	sr.HandleFunc("/artifacts/{name}", user.WithOptionalUser(ui.WithBuild(ui.Download))).Methods("GET")
	sr.HandleFunc("/tags", user.WithOptionalUser(ui.WithBuild(ui.Show))).Methods("GET")
	sr.HandleFunc("/tags", user.WithUser(ui.WithBuild(ui.StoreTag))).Methods("POST")
	sr.HandleFunc("/tags/{name:.+}", user.WithUser(ui.WithBuild(ui.DestroyTag))).Methods("DELETE")
	sr.Use(srv.CSRF)
}
