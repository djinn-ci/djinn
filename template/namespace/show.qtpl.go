// This file is automatically generated by qtc from "show.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/namespace/show.qtpl:2
package namespace

//line template/namespace/show.qtpl:2
import (
	"strings"

	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

//line template/namespace/show.qtpl:10
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/namespace/show.qtpl:10
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/namespace/show.qtpl:11
type ShowPage struct {
	*template.Page

	Namespace *model.Namespace
}

//line template/namespace/show.qtpl:19
func streamrenderFullName(qw422016 *qt422016.Writer, username, fullName string) {
	//line template/namespace/show.qtpl:19
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:21
	parts := strings.Split(fullName, "/")

	//line template/namespace/show.qtpl:22
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:23
	for i, p := range parts {
		//line template/namespace/show.qtpl:23
		qw422016.N().S(` <a href="/u/`)
		//line template/namespace/show.qtpl:24
		qw422016.E().S(username)
		//line template/namespace/show.qtpl:24
		qw422016.N().S(`/`)
		//line template/namespace/show.qtpl:24
		qw422016.E().S(strings.Join(parts[:i+1], "/"))
		//line template/namespace/show.qtpl:24
		qw422016.N().S(`">`)
		//line template/namespace/show.qtpl:24
		qw422016.E().S(p)
		//line template/namespace/show.qtpl:24
		qw422016.N().S(`</a> `)
		//line template/namespace/show.qtpl:25
		if i != len(parts)-1 {
			//line template/namespace/show.qtpl:25
			qw422016.N().S(`<span> / </span>`)
			//line template/namespace/show.qtpl:25
		}
		//line template/namespace/show.qtpl:25
		qw422016.N().S(` `)
		//line template/namespace/show.qtpl:26
	}
	//line template/namespace/show.qtpl:26
	qw422016.N().S(` `)
//line template/namespace/show.qtpl:27
}

//line template/namespace/show.qtpl:27
func writerenderFullName(qq422016 qtio422016.Writer, username, fullName string) {
	//line template/namespace/show.qtpl:27
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:27
	streamrenderFullName(qw422016, username, fullName)
	//line template/namespace/show.qtpl:27
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:27
}

//line template/namespace/show.qtpl:27
func renderFullName(username, fullName string) string {
	//line template/namespace/show.qtpl:27
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:27
	writerenderFullName(qb422016, username, fullName)
	//line template/namespace/show.qtpl:27
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:27
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:27
	return qs422016
//line template/namespace/show.qtpl:27
}

//line template/namespace/show.qtpl:31
func (p *ShowPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:31
	qw422016.N().S(` `)
	//line template/namespace/show.qtpl:32
	qw422016.E().S(p.Namespace.FullName)
	//line template/namespace/show.qtpl:32
	qw422016.N().S(` - Thrall `)
//line template/namespace/show.qtpl:33
}

//line template/namespace/show.qtpl:33
func (p *ShowPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:33
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:33
	p.StreamTitle(qw422016)
	//line template/namespace/show.qtpl:33
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:33
}

//line template/namespace/show.qtpl:33
func (p *ShowPage) Title() string {
	//line template/namespace/show.qtpl:33
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:33
	p.WriteTitle(qb422016)
	//line template/namespace/show.qtpl:33
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:33
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:33
	return qs422016
//line template/namespace/show.qtpl:33
}

//line template/namespace/show.qtpl:35
func (p *ShowPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:35
	qw422016.N().S(` <div class="dashboard-header"> <h1> <a href="/namespaces" class="dashboard-header-back">`)
	//line template/namespace/show.qtpl:38
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
	//line template/namespace/show.qtpl:38
	qw422016.N().S(`</a> `)
	//line template/namespace/show.qtpl:39
	streamrenderFullName(qw422016, p.Namespace.User.Username, p.Namespace.FullName)
	//line template/namespace/show.qtpl:39
	qw422016.N().S(`<br/> <small>`)
	//line template/namespace/show.qtpl:40
	qw422016.E().S(p.Namespace.Description)
	//line template/namespace/show.qtpl:40
	qw422016.N().S(`</small> </h1> <ul class="actions"> <li><a href="/u/`)
	//line template/namespace/show.qtpl:43
	qw422016.E().S(p.Namespace.User.Username)
	//line template/namespace/show.qtpl:43
	qw422016.N().S(`/`)
	//line template/namespace/show.qtpl:43
	qw422016.E().S(p.Namespace.FullName)
	//line template/namespace/show.qtpl:43
	qw422016.N().S(`/-/edit" class="button button-secondary">Edit</a></li> <li><a href="/namespaces/create?parent=`)
	//line template/namespace/show.qtpl:44
	qw422016.E().S(p.Namespace.FullName)
	//line template/namespace/show.qtpl:44
	qw422016.N().S(`" class="button button-primary">Create</a></li> </ul> </div> `)
//line template/namespace/show.qtpl:47
}

//line template/namespace/show.qtpl:47
func (p *ShowPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:47
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:47
	p.StreamBody(qw422016)
	//line template/namespace/show.qtpl:47
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:47
}

//line template/namespace/show.qtpl:47
func (p *ShowPage) Body() string {
	//line template/namespace/show.qtpl:47
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:47
	p.WriteBody(qb422016)
	//line template/namespace/show.qtpl:47
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:47
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:47
	return qs422016
//line template/namespace/show.qtpl:47
}
