// This file is automatically generated by qtc from "edit.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/namespace/edit.qtpl:2
package namespace

//line template/namespace/edit.qtpl:2
import (
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

//line template/namespace/edit.qtpl:9
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/namespace/edit.qtpl:9
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/namespace/edit.qtpl:10
type EditPage struct {
	*template.Page

	Errors    form.Errors
	Form      form.Form
	Namespace *model.Namespace
}

//line template/namespace/edit.qtpl:19
func (p *EditPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/namespace/edit.qtpl:19
	qw422016.N().S(`
Edit Namespace - Thrall
`)
//line template/namespace/edit.qtpl:21
}

//line template/namespace/edit.qtpl:21
func (p *EditPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/namespace/edit.qtpl:21
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/edit.qtpl:21
	p.StreamTitle(qw422016)
	//line template/namespace/edit.qtpl:21
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/edit.qtpl:21
}

//line template/namespace/edit.qtpl:21
func (p *EditPage) Title() string {
	//line template/namespace/edit.qtpl:21
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/edit.qtpl:21
	p.WriteTitle(qb422016)
	//line template/namespace/edit.qtpl:21
	qs422016 := string(qb422016.B)
	//line template/namespace/edit.qtpl:21
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/edit.qtpl:21
	return qs422016
//line template/namespace/edit.qtpl:21
}

//line template/namespace/edit.qtpl:24
func (p *EditPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/namespace/edit.qtpl:24
	qw422016.N().S(` <div class="dashboard-header"> <h1> <a href="/u/`)
	//line template/namespace/edit.qtpl:27
	qw422016.E().S(p.Namespace.User.Username)
	//line template/namespace/edit.qtpl:27
	qw422016.N().S(`/`)
	//line template/namespace/edit.qtpl:27
	qw422016.E().S(p.Namespace.FullName)
	//line template/namespace/edit.qtpl:27
	qw422016.N().S(`" class="dashboard-header-back"> `)
	//line template/namespace/edit.qtpl:28
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<title>Back</title>
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
	//line template/namespace/edit.qtpl:28
	qw422016.N().S(` </a>`)
	//line template/namespace/edit.qtpl:29
	streamrenderFullName(qw422016, p.Namespace.User.Username, p.Namespace.FullName)
	//line template/namespace/edit.qtpl:29
	qw422016.N().S(` - Edit</h1> </div> <div class="dashboard-body"> <div class="panel panel-slim"> <form method="POST" action="/u/`)
	//line template/namespace/edit.qtpl:33
	qw422016.E().S(p.Namespace.User.Username)
	//line template/namespace/edit.qtpl:33
	qw422016.N().S(`/`)
	//line template/namespace/edit.qtpl:33
	qw422016.E().S(p.Namespace.FullName)
	//line template/namespace/edit.qtpl:33
	qw422016.N().S(`"> <input type="hidden" name="_method" value="PATCH"/> `)
	//line template/namespace/edit.qtpl:35
	if p.Errors.First("namespace") != "" {
		//line template/namespace/edit.qtpl:35
		qw422016.N().S(` <div class="form-error">Failed to create namespace: `)
		//line template/namespace/edit.qtpl:36
		qw422016.E().S(p.Errors.First("namespace"))
		//line template/namespace/edit.qtpl:36
		qw422016.N().S(`</div> `)
		//line template/namespace/edit.qtpl:37
	}
	//line template/namespace/edit.qtpl:37
	qw422016.N().S(` <div class="input-field"> <label class="input-field-label">Name</label> `)
	//line template/namespace/edit.qtpl:40
	if p.Form.Get("name") != "" {
		//line template/namespace/edit.qtpl:40
		qw422016.N().S(` <input class="input-text" type="text" name="name" value="`)
		//line template/namespace/edit.qtpl:41
		qw422016.E().S(p.Form.Get("name"))
		//line template/namespace/edit.qtpl:41
		qw422016.N().S(`" autocomplete="off"/> `)
		//line template/namespace/edit.qtpl:42
	} else {
		//line template/namespace/edit.qtpl:42
		qw422016.N().S(` <input class="input-text" type="text" name="name" value="`)
		//line template/namespace/edit.qtpl:43
		qw422016.E().S(p.Namespace.Name)
		//line template/namespace/edit.qtpl:43
		qw422016.N().S(`" autocomplete="off"/> `)
		//line template/namespace/edit.qtpl:44
	}
	//line template/namespace/edit.qtpl:44
	qw422016.N().S(` <span class="error">`)
	//line template/namespace/edit.qtpl:45
	qw422016.E().S(p.Errors.First("name"))
	//line template/namespace/edit.qtpl:45
	qw422016.N().S(`</span> </div> <div class="input-field"> <label class="input-field-label">Description</label> `)
	//line template/namespace/edit.qtpl:49
	if p.Form.Get("description") != "" {
		//line template/namespace/edit.qtpl:49
		qw422016.N().S(` <textarea class="input-text" name="description">`)
		//line template/namespace/edit.qtpl:50
		qw422016.E().S(p.Form.Get("description"))
		//line template/namespace/edit.qtpl:50
		qw422016.N().S(`</textarea> `)
		//line template/namespace/edit.qtpl:51
	} else {
		//line template/namespace/edit.qtpl:51
		qw422016.N().S(` <textarea class="input-text" name="description">`)
		//line template/namespace/edit.qtpl:52
		qw422016.E().S(p.Namespace.Description)
		//line template/namespace/edit.qtpl:52
		qw422016.N().S(`</textarea> `)
		//line template/namespace/edit.qtpl:53
	}
	//line template/namespace/edit.qtpl:53
	qw422016.N().S(` </div> <div class="input-field"> <label class="input-field-label">Visibility</label> <label class="input-option"> <input class="input-option-selector" type="radio" name="visibility" value="private" `)
	//line template/namespace/edit.qtpl:58
	if p.Namespace.Visibility == model.Private {
		//line template/namespace/edit.qtpl:58
		qw422016.N().S(`checked="true"`)
		//line template/namespace/edit.qtpl:58
	}
	//line template/namespace/edit.qtpl:58
	qw422016.N().S(`/> <div class="input-option-description"> Private<br/> Only you will be able to view builds in the namespace. </div> </label> <label class="input-option"> <input class="input-option-selector" type="radio" name="visibility" value="internal" `)
	//line template/namespace/edit.qtpl:65
	if p.Namespace.Visibility == model.Internal {
		//line template/namespace/edit.qtpl:65
		qw422016.N().S(`checked="true"`)
		//line template/namespace/edit.qtpl:65
	}
	//line template/namespace/edit.qtpl:65
	qw422016.N().S(`/> <div class="input-option-description"> Internal<br/> Anyone with an account will be able to view builds in the namespace </div> </label> <label class="input-option"> <input class="input-option-selector" type="radio" name="visibility" value="public" `)
	//line template/namespace/edit.qtpl:72
	if p.Namespace.Visibility == model.Public {
		//line template/namespace/edit.qtpl:72
		qw422016.N().S(`checked="true"`)
		//line template/namespace/edit.qtpl:72
	}
	//line template/namespace/edit.qtpl:72
	qw422016.N().S(`/> <div class="input-option-description"> Public<br/> Anyone will be able to view builds in the namespace. </div> </label> </div> <div class="input-field"> <button type="submit" class="button button-primary">Save</button> </div> </form> </div> </div> `)
//line template/namespace/edit.qtpl:85
}

//line template/namespace/edit.qtpl:85
func (p *EditPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/namespace/edit.qtpl:85
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/edit.qtpl:85
	p.StreamBody(qw422016)
	//line template/namespace/edit.qtpl:85
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/edit.qtpl:85
}

//line template/namespace/edit.qtpl:85
func (p *EditPage) Body() string {
	//line template/namespace/edit.qtpl:85
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/edit.qtpl:85
	p.WriteBody(qb422016)
	//line template/namespace/edit.qtpl:85
	qs422016 := string(qb422016.B)
	//line template/namespace/edit.qtpl:85
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/edit.qtpl:85
	return qs422016
//line template/namespace/edit.qtpl:85
}
