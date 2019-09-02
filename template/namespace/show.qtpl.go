// This file is automatically generated by qtc from "show.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/namespace/show.qtpl:2
package namespace

//line template/namespace/show.qtpl:2
import (
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/build"
	"github.com/andrewpillar/thrall/template/key"
	"github.com/andrewpillar/thrall/template/object"
	"github.com/andrewpillar/thrall/template/variable"
)

//line template/namespace/show.qtpl:13
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/namespace/show.qtpl:13
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/namespace/show.qtpl:14
type ShowPage struct {
	template.BasePage

	Namespace *model.Namespace

	Status string
	Search string

	Builds []*model.Build
}

type ShowNamespaces struct {
	ShowPage

	Index IndexPage
}

type ShowObjects struct {
	ShowPage

	Index object.IndexPage
}

type ShowVariables struct {
	ShowPage

	Index variable.IndexPage
}

type ShowKeys struct {
	ShowPage

	Index key.IndexPage
}

type ShowCollaborators struct {
	ShowPage

	CSRF          string
	Fields        map[string]string
	Errors        form.Errors
	Collaborators []*model.Collaborator
}

//line template/namespace/show.qtpl:60
func (p *ShowPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:60
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:61
	qw422016.E().S(p.Namespace.Path)
	//line template/namespace/show.qtpl:61
	qw422016.N().S(` - Thrall `)
//line template/namespace/show.qtpl:62
}

//line template/namespace/show.qtpl:62
func (p *ShowPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:62
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:62
	p.StreamTitle(qw422016)
	//line template/namespace/show.qtpl:62
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:62
}

//line template/namespace/show.qtpl:62
func (p *ShowPage) Title() string {
	//line template/namespace/show.qtpl:62
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:62
	p.WriteTitle(qb422016)
	//line template/namespace/show.qtpl:62
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:62
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:62
	return qs422016
//line template/namespace/show.qtpl:62
}

//line template/namespace/show.qtpl:64
func (p *ShowNamespaces) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:64
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:65
	qw422016.E().S(p.Namespace.Path)
	//line template/namespace/show.qtpl:65
	qw422016.N().S(` - Namespaces `)
//line template/namespace/show.qtpl:66
}

//line template/namespace/show.qtpl:66
func (p *ShowNamespaces) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:66
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:66
	p.StreamTitle(qw422016)
	//line template/namespace/show.qtpl:66
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:66
}

//line template/namespace/show.qtpl:66
func (p *ShowNamespaces) Title() string {
	//line template/namespace/show.qtpl:66
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:66
	p.WriteTitle(qb422016)
	//line template/namespace/show.qtpl:66
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:66
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:66
	return qs422016
//line template/namespace/show.qtpl:66
}

//line template/namespace/show.qtpl:68
func (p *ShowObjects) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:68
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:69
	qw422016.E().S(p.Namespace.Path)
	//line template/namespace/show.qtpl:69
	qw422016.N().S(` - Objects `)
//line template/namespace/show.qtpl:70
}

//line template/namespace/show.qtpl:70
func (p *ShowObjects) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:70
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:70
	p.StreamTitle(qw422016)
	//line template/namespace/show.qtpl:70
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:70
}

//line template/namespace/show.qtpl:70
func (p *ShowObjects) Title() string {
	//line template/namespace/show.qtpl:70
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:70
	p.WriteTitle(qb422016)
	//line template/namespace/show.qtpl:70
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:70
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:70
	return qs422016
//line template/namespace/show.qtpl:70
}

//line template/namespace/show.qtpl:72
func (p *ShowVariables) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:72
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:73
	qw422016.E().S(p.Namespace.Path)
	//line template/namespace/show.qtpl:73
	qw422016.N().S(` - Variables `)
//line template/namespace/show.qtpl:74
}

//line template/namespace/show.qtpl:74
func (p *ShowVariables) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:74
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:74
	p.StreamTitle(qw422016)
	//line template/namespace/show.qtpl:74
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:74
}

//line template/namespace/show.qtpl:74
func (p *ShowVariables) Title() string {
	//line template/namespace/show.qtpl:74
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:74
	p.WriteTitle(qb422016)
	//line template/namespace/show.qtpl:74
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:74
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:74
	return qs422016
//line template/namespace/show.qtpl:74
}

//line template/namespace/show.qtpl:76
func (p *ShowKeys) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:76
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:77
	qw422016.E().S(p.Namespace.Path)
	//line template/namespace/show.qtpl:77
	qw422016.N().S(` - Keys `)
//line template/namespace/show.qtpl:78
}

//line template/namespace/show.qtpl:78
func (p *ShowKeys) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:78
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:78
	p.StreamTitle(qw422016)
	//line template/namespace/show.qtpl:78
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:78
}

//line template/namespace/show.qtpl:78
func (p *ShowKeys) Title() string {
	//line template/namespace/show.qtpl:78
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:78
	p.WriteTitle(qb422016)
	//line template/namespace/show.qtpl:78
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:78
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:78
	return qs422016
//line template/namespace/show.qtpl:78
}

//line template/namespace/show.qtpl:80
func (p *ShowCollaborators) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:80
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:81
	qw422016.E().S(p.Namespace.Path)
	//line template/namespace/show.qtpl:81
	qw422016.N().S(` - Collaborators `)
//line template/namespace/show.qtpl:82
}

//line template/namespace/show.qtpl:82
func (p *ShowCollaborators) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:82
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:82
	p.StreamTitle(qw422016)
	//line template/namespace/show.qtpl:82
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:82
}

//line template/namespace/show.qtpl:82
func (p *ShowCollaborators) Title() string {
	//line template/namespace/show.qtpl:82
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:82
	p.WriteTitle(qb422016)
	//line template/namespace/show.qtpl:82
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:82
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:82
	return qs422016
//line template/namespace/show.qtpl:82
}

//line template/namespace/show.qtpl:84
func (p *ShowPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:84
	qw422016.N().S(` <div class="panel">`)
	//line template/namespace/show.qtpl:85
	build.StreamRenderIndex(qw422016, p.Builds, p.URI, p.Status, p.Search)
	//line template/namespace/show.qtpl:85
	qw422016.N().S(`</div> `)
//line template/namespace/show.qtpl:86
}

//line template/namespace/show.qtpl:86
func (p *ShowPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:86
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:86
	p.StreamBody(qw422016)
	//line template/namespace/show.qtpl:86
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:86
}

//line template/namespace/show.qtpl:86
func (p *ShowPage) Body() string {
	//line template/namespace/show.qtpl:86
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:86
	p.WriteBody(qb422016)
	//line template/namespace/show.qtpl:86
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:86
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:86
	return qs422016
//line template/namespace/show.qtpl:86
}

//line template/namespace/show.qtpl:88
func (p *ShowNamespaces) StreamBody(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:88
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:89
	p.Index.StreamBody(qw422016)
	//line template/namespace/show.qtpl:89
	qw422016.N().S(` `)
//line template/namespace/show.qtpl:90
}

//line template/namespace/show.qtpl:90
func (p *ShowNamespaces) WriteBody(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:90
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:90
	p.StreamBody(qw422016)
	//line template/namespace/show.qtpl:90
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:90
}

//line template/namespace/show.qtpl:90
func (p *ShowNamespaces) Body() string {
	//line template/namespace/show.qtpl:90
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:90
	p.WriteBody(qb422016)
	//line template/namespace/show.qtpl:90
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:90
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:90
	return qs422016
//line template/namespace/show.qtpl:90
}

//line template/namespace/show.qtpl:92
func (p *ShowObjects) StreamBody(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:92
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:93
	p.Index.StreamBody(qw422016)
	//line template/namespace/show.qtpl:93
	qw422016.N().S(` `)
//line template/namespace/show.qtpl:94
}

//line template/namespace/show.qtpl:94
func (p *ShowObjects) WriteBody(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:94
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:94
	p.StreamBody(qw422016)
	//line template/namespace/show.qtpl:94
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:94
}

//line template/namespace/show.qtpl:94
func (p *ShowObjects) Body() string {
	//line template/namespace/show.qtpl:94
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:94
	p.WriteBody(qb422016)
	//line template/namespace/show.qtpl:94
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:94
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:94
	return qs422016
//line template/namespace/show.qtpl:94
}

//line template/namespace/show.qtpl:96
func (p *ShowVariables) StreamBody(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:96
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:97
	p.Index.StreamBody(qw422016)
	//line template/namespace/show.qtpl:97
	qw422016.N().S(` `)
//line template/namespace/show.qtpl:98
}

//line template/namespace/show.qtpl:98
func (p *ShowVariables) WriteBody(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:98
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:98
	p.StreamBody(qw422016)
	//line template/namespace/show.qtpl:98
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:98
}

//line template/namespace/show.qtpl:98
func (p *ShowVariables) Body() string {
	//line template/namespace/show.qtpl:98
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:98
	p.WriteBody(qb422016)
	//line template/namespace/show.qtpl:98
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:98
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:98
	return qs422016
//line template/namespace/show.qtpl:98
}

//line template/namespace/show.qtpl:100
func (p *ShowKeys) StreamBody(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:100
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:101
	p.Index.StreamBody(qw422016)
	//line template/namespace/show.qtpl:101
	qw422016.N().S(` `)
//line template/namespace/show.qtpl:102
}

//line template/namespace/show.qtpl:102
func (p *ShowKeys) WriteBody(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:102
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:102
	p.StreamBody(qw422016)
	//line template/namespace/show.qtpl:102
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:102
}

//line template/namespace/show.qtpl:102
func (p *ShowKeys) Body() string {
	//line template/namespace/show.qtpl:102
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:102
	p.WriteBody(qb422016)
	//line template/namespace/show.qtpl:102
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:102
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:102
	return qs422016
//line template/namespace/show.qtpl:102
}

//line template/namespace/show.qtpl:104
func (p *ShowCollaborators) StreamBody(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:104
	qw422016.N().S(` <div class="panel"> <div class="panel-header panel-body"> <form method="POST" action="`)
	//line template/namespace/show.qtpl:107
	qw422016.E().S(p.Namespace.UIEndpoint("-", "collaborators"))
	//line template/namespace/show.qtpl:107
	qw422016.N().S(`"> `)
	//line template/namespace/show.qtpl:108
	qw422016.N().S(p.CSRF)
	//line template/namespace/show.qtpl:108
	qw422016.N().S(` <div class="form-field form-field-inline"> <input type="text" class="form-text" name="handle" placeholder="Add a collaborator..." value="`)
	//line template/namespace/show.qtpl:110
	qw422016.E().S(p.Fields["handle"])
	//line template/namespace/show.qtpl:110
	qw422016.N().S(`" autocomplete="off"/> <button type="submit" class="btn btn-primary">Add</button> <span class="form-error">`)
	//line template/namespace/show.qtpl:112
	qw422016.E().S(p.Errors.First("handle"))
	//line template/namespace/show.qtpl:112
	qw422016.N().S(`</span> </div> </form> </div> `)
	//line template/namespace/show.qtpl:116
	if len(p.Collaborators) == 0 {
		//line template/namespace/show.qtpl:116
		qw422016.N().S(` <div class="panel-message muted">Share resources with other users by adding them as a collaborator.</div> `)
		//line template/namespace/show.qtpl:118
	} else {
		//line template/namespace/show.qtpl:118
		qw422016.N().S(` <table class="table"> <thead> <tr> <th>USER</th> <th></th> </tr> </thead> <tbody> `)
		//line template/namespace/show.qtpl:127
		for _, c := range p.Collaborators {
			//line template/namespace/show.qtpl:127
			qw422016.N().S(` <tr> <td>`)
			//line template/namespace/show.qtpl:129
			qw422016.E().S(c.User.Username)
			//line template/namespace/show.qtpl:129
			qw422016.N().S(` &lt;`)
			//line template/namespace/show.qtpl:129
			qw422016.E().S(c.User.Email)
			//line template/namespace/show.qtpl:129
			qw422016.N().S(`&gt;</td> `)
			//line template/namespace/show.qtpl:130
			if p.User != nil && !p.User.IsZero() {
				//line template/namespace/show.qtpl:130
				qw422016.N().S(` <td class="align-right"> <form method="POST" action="`)
				//line template/namespace/show.qtpl:132
				qw422016.E().S(c.UIEndpoint())
				//line template/namespace/show.qtpl:132
				qw422016.N().S(`"> `)
				//line template/namespace/show.qtpl:133
				qw422016.N().S(p.CSRF)
				//line template/namespace/show.qtpl:133
				qw422016.N().S(` <input type="hidden" name="_method" value="DELETE"/> <button type="submit" class="btn btn-danger">Delete</button> </form> </td> `)
				//line template/namespace/show.qtpl:138
			}
			//line template/namespace/show.qtpl:138
			qw422016.N().S(` </tr> `)
			//line template/namespace/show.qtpl:140
		}
		//line template/namespace/show.qtpl:140
		qw422016.N().S(` </tbody> </table> `)
		//line template/namespace/show.qtpl:143
	}
	//line template/namespace/show.qtpl:143
	qw422016.N().S(` </div> `)
//line template/namespace/show.qtpl:145
}

//line template/namespace/show.qtpl:145
func (p *ShowCollaborators) WriteBody(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:145
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:145
	p.StreamBody(qw422016)
	//line template/namespace/show.qtpl:145
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:145
}

//line template/namespace/show.qtpl:145
func (p *ShowCollaborators) Body() string {
	//line template/namespace/show.qtpl:145
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:145
	p.WriteBody(qb422016)
	//line template/namespace/show.qtpl:145
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:145
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:145
	return qs422016
//line template/namespace/show.qtpl:145
}

//line template/namespace/show.qtpl:147
func (p *ShowPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:147
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:148
	if p.Namespace.Parent != nil {
		//line template/namespace/show.qtpl:148
		qw422016.N().S(` <a class="back" href="`)
		//line template/namespace/show.qtpl:149
		qw422016.E().S(p.Namespace.Parent.UIEndpoint())
		//line template/namespace/show.qtpl:149
		qw422016.N().S(`">`)
		//line template/namespace/show.qtpl:149
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
		//line template/namespace/show.qtpl:149
		qw422016.N().S(`</a> `)
		//line template/namespace/show.qtpl:150
	} else {
		//line template/namespace/show.qtpl:150
		qw422016.N().S(` <a class="back" href="/namespaces">`)
		//line template/namespace/show.qtpl:151
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
		//line template/namespace/show.qtpl:151
		qw422016.N().S(`</a> `)
		//line template/namespace/show.qtpl:152
	}
	//line template/namespace/show.qtpl:152
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:153
	streamrenderPath(qw422016, p.Namespace.User.Username, p.Namespace.Path)
	//line template/namespace/show.qtpl:153
	qw422016.N().S(` <small>`)
	//line template/namespace/show.qtpl:154
	qw422016.E().S(p.Namespace.Description)
	//line template/namespace/show.qtpl:154
	qw422016.N().S(`</small> `)
//line template/namespace/show.qtpl:155
}

//line template/namespace/show.qtpl:155
func (p *ShowPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:155
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:155
	p.StreamHeader(qw422016)
	//line template/namespace/show.qtpl:155
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:155
}

//line template/namespace/show.qtpl:155
func (p *ShowPage) Header() string {
	//line template/namespace/show.qtpl:155
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:155
	p.WriteHeader(qb422016)
	//line template/namespace/show.qtpl:155
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:155
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:155
	return qs422016
//line template/namespace/show.qtpl:155
}

//line template/namespace/show.qtpl:157
func (p *ShowPage) StreamActions(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:157
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:158
	if p.User != nil && p.User.ID == p.Namespace.UserID {
		//line template/namespace/show.qtpl:158
		qw422016.N().S(` <li><a href="`)
		//line template/namespace/show.qtpl:159
		qw422016.E().S(p.Namespace.UIEndpoint())
		//line template/namespace/show.qtpl:159
		qw422016.N().S(`/-/edit" class="btn btn-primary">Edit</a></li> `)
		//line template/namespace/show.qtpl:160
		if p.Namespace.Level+1 < model.NamespaceMaxDepth {
			//line template/namespace/show.qtpl:160
			qw422016.N().S(` <li><a href="/namespaces/create?parent=`)
			//line template/namespace/show.qtpl:161
			qw422016.E().S(p.Namespace.Path)
			//line template/namespace/show.qtpl:161
			qw422016.N().S(`" class="btn btn-primary">Create</a></li> `)
			//line template/namespace/show.qtpl:162
		}
		//line template/namespace/show.qtpl:162
		qw422016.N().S(` `)
		//line template/namespace/show.qtpl:163
	}
	//line template/namespace/show.qtpl:163
	qw422016.N().S(` `)
//line template/namespace/show.qtpl:164
}

//line template/namespace/show.qtpl:164
func (p *ShowPage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:164
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:164
	p.StreamActions(qw422016)
	//line template/namespace/show.qtpl:164
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:164
}

//line template/namespace/show.qtpl:164
func (p *ShowPage) Actions() string {
	//line template/namespace/show.qtpl:164
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:164
	p.WriteActions(qb422016)
	//line template/namespace/show.qtpl:164
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:164
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:164
	return qs422016
//line template/namespace/show.qtpl:164
}

//line template/namespace/show.qtpl:166
func (p *ShowPage) StreamNavigation(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:166
	qw422016.N().S(` <li> <a href="`)
	//line template/namespace/show.qtpl:168
	qw422016.E().S(p.Namespace.UIEndpoint())
	//line template/namespace/show.qtpl:168
	qw422016.N().S(`" class="`)
	//line template/namespace/show.qtpl:168
	qw422016.E().S(template.Active(p.Namespace.UIEndpoint() == p.URI))
	//line template/namespace/show.qtpl:168
	qw422016.N().S(`"> `)
	//line template/namespace/show.qtpl:169
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M22.688 18.984c0.422 0.281 0.422 0.984-0.094 1.406l-2.297 2.297c-0.422 0.422-0.984 0.422-1.406 0l-9.094-9.094c-2.297 0.891-4.969 0.422-6.891-1.5-2.016-2.016-2.531-5.016-1.313-7.406l4.406 4.313 3-3-4.313-4.313c2.391-1.078 5.391-0.703 7.406 1.313 1.922 1.922 2.391 4.594 1.5 6.891z"></path>
</svg>
`)
	//line template/namespace/show.qtpl:169
	qw422016.N().S(`<span>Builds</span> </a> </li> <li> <a href="`)
	//line template/namespace/show.qtpl:173
	qw422016.E().S(p.Namespace.UIEndpoint("-", "namespaces"))
	//line template/namespace/show.qtpl:173
	qw422016.N().S(`" class="`)
	//line template/namespace/show.qtpl:173
	qw422016.E().S(template.Active(p.Namespace.UIEndpoint("-", "namespaces") == p.URI))
	//line template/namespace/show.qtpl:173
	qw422016.N().S(`"> `)
	//line template/namespace/show.qtpl:174
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M9.984 3.984l2.016 2.016h8.016c1.078 0 1.969 0.938 1.969 2.016v9.984c0 1.078-0.891 2.016-1.969 2.016h-16.031c-1.078 0-1.969-0.938-1.969-2.016v-12c0-1.078 0.891-2.016 1.969-2.016h6z"></path>
</svg>
`)
	//line template/namespace/show.qtpl:174
	qw422016.N().S(`<span>Namespaces</span> </a> </li> <li> <a href="`)
	//line template/namespace/show.qtpl:178
	qw422016.E().S(p.Namespace.UIEndpoint("-", "objects"))
	//line template/namespace/show.qtpl:178
	qw422016.N().S(`" class="`)
	//line template/namespace/show.qtpl:178
	qw422016.E().S(template.Active(p.Namespace.UIEndpoint("-", "objects") == p.URI))
	//line template/namespace/show.qtpl:178
	qw422016.N().S(`"> `)
	//line template/namespace/show.qtpl:179
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M5.016 18h13.969v2.016h-13.969v-2.016zM9 15.984v-6h-3.984l6.984-6.984 6.984 6.984h-3.984v6h-6z"></path>
</svg>
`)
	//line template/namespace/show.qtpl:179
	qw422016.N().S(`<span>Objects</span> </a> </li> <li> <a href="`)
	//line template/namespace/show.qtpl:183
	qw422016.E().S(p.Namespace.UIEndpoint("-", "variables"))
	//line template/namespace/show.qtpl:183
	qw422016.N().S(`" class="`)
	//line template/namespace/show.qtpl:183
	qw422016.E().S(template.Active(p.Namespace.UIEndpoint("-", "variables") == p.URI))
	//line template/namespace/show.qtpl:183
	qw422016.N().S(`"> `)
	//line template/namespace/show.qtpl:184
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M14.578 16.594l4.641-4.594-4.641-4.594 1.406-1.406 6 6-6 6zM9.422 16.594l-1.406 1.406-6-6 6-6 1.406 1.406-4.641 4.594z"></path>
</svg>
`)
	//line template/namespace/show.qtpl:184
	qw422016.N().S(`<span>Variables</span> </a> </li> <li> <a href="`)
	//line template/namespace/show.qtpl:188
	qw422016.E().S(p.Namespace.UIEndpoint("-", "keys"))
	//line template/namespace/show.qtpl:188
	qw422016.N().S(`" class="`)
	//line template/namespace/show.qtpl:188
	qw422016.E().S(template.Active(p.Namespace.UIEndpoint("-", "keys") == p.URI))
	//line template/namespace/show.qtpl:188
	qw422016.N().S(`"> `)
	//line template/namespace/show.qtpl:189
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M6.984 14.016c1.078 0 2.016-0.938 2.016-2.016s-0.938-2.016-2.016-2.016-1.969 0.938-1.969 2.016 0.891 2.016 1.969 2.016zM12.656 9.984h10.359v4.031h-2.016v3.984h-3.984v-3.984h-4.359c-0.797 2.344-3.047 3.984-5.672 3.984-3.328 0-6-2.672-6-6s2.672-6 6-6c2.625 0 4.875 1.641 5.672 3.984z"></path>
</svg>
`)
	//line template/namespace/show.qtpl:189
	qw422016.N().S(`<span>SSH Keys</span> </a> </li> <li> <a href="`)
	//line template/namespace/show.qtpl:193
	qw422016.E().S(p.Namespace.UIEndpoint("-", "collaborators"))
	//line template/namespace/show.qtpl:193
	qw422016.N().S(`" class="`)
	//line template/namespace/show.qtpl:193
	qw422016.E().S(template.Active(p.Namespace.UIEndpoint("-", "collaborators") == p.URI))
	//line template/namespace/show.qtpl:193
	qw422016.N().S(`"> `)
	//line template/namespace/show.qtpl:194
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M15.984 12.984c2.344 0 7.031 1.172 7.031 3.516v2.484h-6v-2.484c0-1.5-0.797-2.625-1.969-3.469 0.328-0.047 0.656-0.047 0.938-0.047zM8.016 12.984c2.344 0 6.984 1.172 6.984 3.516v2.484h-14.016v-2.484c0-2.344 4.688-3.516 7.031-3.516zM8.016 11.016c-1.641 0-3-1.359-3-3s1.359-3 3-3 2.953 1.359 2.953 3-1.313 3-2.953 3zM15.984 11.016c-1.641 0-3-1.359-3-3s1.359-3 3-3 3 1.359 3 3-1.359 3-3 3z"></path>
</svg>
`)
	//line template/namespace/show.qtpl:194
	qw422016.N().S(`<span>Collaborators</span> </a> </li> `)
//line template/namespace/show.qtpl:197
}

//line template/namespace/show.qtpl:197
func (p *ShowPage) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:197
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:197
	p.StreamNavigation(qw422016)
	//line template/namespace/show.qtpl:197
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:197
}

//line template/namespace/show.qtpl:197
func (p *ShowPage) Navigation() string {
	//line template/namespace/show.qtpl:197
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:197
	p.WriteNavigation(qb422016)
	//line template/namespace/show.qtpl:197
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:197
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:197
	return qs422016
//line template/namespace/show.qtpl:197
}
