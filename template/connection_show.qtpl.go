// Code generated by qtc from "connection_show.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/connection_show.qtpl:2
package template

//line template/connection_show.qtpl:2
import (
	"djinn-ci.com/oauth2"
	"djinn-ci.com/template/form"
)

//line template/connection_show.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/connection_show.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/connection_show.qtpl:9
type ConnectionShow struct {
	*form.Form

	Token *oauth2.Token
}

//line template/connection_show.qtpl:17
func (p *ConnectionShow) StreamTitle(qw422016 *qt422016.Writer) {
//line template/connection_show.qtpl:17
	qw422016.N().S(`Connection to `)
//line template/connection_show.qtpl:17
	qw422016.E().S(p.Token.App.Name)
//line template/connection_show.qtpl:17
}

//line template/connection_show.qtpl:17
func (p *ConnectionShow) WriteTitle(qq422016 qtio422016.Writer) {
//line template/connection_show.qtpl:17
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/connection_show.qtpl:17
	p.StreamTitle(qw422016)
//line template/connection_show.qtpl:17
	qt422016.ReleaseWriter(qw422016)
//line template/connection_show.qtpl:17
}

//line template/connection_show.qtpl:17
func (p *ConnectionShow) Title() string {
//line template/connection_show.qtpl:17
	qb422016 := qt422016.AcquireByteBuffer()
//line template/connection_show.qtpl:17
	p.WriteTitle(qb422016)
//line template/connection_show.qtpl:17
	qs422016 := string(qb422016.B)
//line template/connection_show.qtpl:17
	qt422016.ReleaseByteBuffer(qb422016)
//line template/connection_show.qtpl:17
	return qs422016
//line template/connection_show.qtpl:17
}

//line template/connection_show.qtpl:19
func (p *ConnectionShow) StreamHeader(qw422016 *qt422016.Writer) {
//line template/connection_show.qtpl:19
	qw422016.N().S(` <a href="/settings/connections" class="back"> `)
//line template/connection_show.qtpl:21
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
//line template/connection_show.qtpl:21
	qw422016.N().S(` </a> Connection to `)
//line template/connection_show.qtpl:22
	qw422016.E().S(p.Token.App.Name)
//line template/connection_show.qtpl:22
	qw422016.N().S(` `)
//line template/connection_show.qtpl:23
}

//line template/connection_show.qtpl:23
func (p *ConnectionShow) WriteHeader(qq422016 qtio422016.Writer) {
//line template/connection_show.qtpl:23
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/connection_show.qtpl:23
	p.StreamHeader(qw422016)
//line template/connection_show.qtpl:23
	qt422016.ReleaseWriter(qw422016)
//line template/connection_show.qtpl:23
}

//line template/connection_show.qtpl:23
func (p *ConnectionShow) Header() string {
//line template/connection_show.qtpl:23
	qb422016 := qt422016.AcquireByteBuffer()
//line template/connection_show.qtpl:23
	p.WriteHeader(qb422016)
//line template/connection_show.qtpl:23
	qs422016 := string(qb422016.B)
//line template/connection_show.qtpl:23
	qt422016.ReleaseByteBuffer(qb422016)
//line template/connection_show.qtpl:23
	return qs422016
//line template/connection_show.qtpl:23
}

//line template/connection_show.qtpl:25
func (p *ConnectionShow) StreamActions(qw422016 *qt422016.Writer) {
//line template/connection_show.qtpl:25
	qw422016.N().S(` <li> <form method="POST" actions="/settings/connections/`)
//line template/connection_show.qtpl:27
	qw422016.E().S(p.Token.App.ClientID)
//line template/connection_show.qtpl:27
	qw422016.N().S(`"> `)
//line template/connection_show.qtpl:28
	form.StreamMethod(qw422016, "DELETE")
//line template/connection_show.qtpl:28
	qw422016.N().S(` `)
//line template/connection_show.qtpl:29
	qw422016.N().V(p.CSRF)
//line template/connection_show.qtpl:29
	qw422016.N().S(` <button type="submit" class="btn btn-danger">Revoke access</button> </form> </li> `)
//line template/connection_show.qtpl:33
}

//line template/connection_show.qtpl:33
func (p *ConnectionShow) WriteActions(qq422016 qtio422016.Writer) {
//line template/connection_show.qtpl:33
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/connection_show.qtpl:33
	p.StreamActions(qw422016)
//line template/connection_show.qtpl:33
	qt422016.ReleaseWriter(qw422016)
//line template/connection_show.qtpl:33
}

//line template/connection_show.qtpl:33
func (p *ConnectionShow) Actions() string {
//line template/connection_show.qtpl:33
	qb422016 := qt422016.AcquireByteBuffer()
//line template/connection_show.qtpl:33
	p.WriteActions(qb422016)
//line template/connection_show.qtpl:33
	qs422016 := string(qb422016.B)
//line template/connection_show.qtpl:33
	qt422016.ReleaseByteBuffer(qb422016)
//line template/connection_show.qtpl:33
	return qs422016
//line template/connection_show.qtpl:33
}

//line template/connection_show.qtpl:35
func (p *ConnectionShow) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/connection_show.qtpl:35
}

//line template/connection_show.qtpl:35
func (p *ConnectionShow) WriteNavigation(qq422016 qtio422016.Writer) {
//line template/connection_show.qtpl:35
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/connection_show.qtpl:35
	p.StreamNavigation(qw422016)
//line template/connection_show.qtpl:35
	qt422016.ReleaseWriter(qw422016)
//line template/connection_show.qtpl:35
}

//line template/connection_show.qtpl:35
func (p *ConnectionShow) Navigation() string {
//line template/connection_show.qtpl:35
	qb422016 := qt422016.AcquireByteBuffer()
//line template/connection_show.qtpl:35
	p.WriteNavigation(qb422016)
//line template/connection_show.qtpl:35
	qs422016 := string(qb422016.B)
//line template/connection_show.qtpl:35
	qt422016.ReleaseByteBuffer(qb422016)
//line template/connection_show.qtpl:35
	return qs422016
//line template/connection_show.qtpl:35
}

//line template/connection_show.qtpl:36
func (p *ConnectionShow) StreamFooter(qw422016 *qt422016.Writer) {
//line template/connection_show.qtpl:36
}

//line template/connection_show.qtpl:36
func (p *ConnectionShow) WriteFooter(qq422016 qtio422016.Writer) {
//line template/connection_show.qtpl:36
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/connection_show.qtpl:36
	p.StreamFooter(qw422016)
//line template/connection_show.qtpl:36
	qt422016.ReleaseWriter(qw422016)
//line template/connection_show.qtpl:36
}

//line template/connection_show.qtpl:36
func (p *ConnectionShow) Footer() string {
//line template/connection_show.qtpl:36
	qb422016 := qt422016.AcquireByteBuffer()
//line template/connection_show.qtpl:36
	p.WriteFooter(qb422016)
//line template/connection_show.qtpl:36
	qs422016 := string(qb422016.B)
//line template/connection_show.qtpl:36
	qt422016.ReleaseByteBuffer(qb422016)
//line template/connection_show.qtpl:36
	return qs422016
//line template/connection_show.qtpl:36
}

//line template/connection_show.qtpl:38
func (p *ConnectionShow) StreamBody(qw422016 *qt422016.Writer) {
//line template/connection_show.qtpl:38
	qw422016.N().S(` <div class="panel"> <div class="panel-body slim scope-list"> <strong>Authorized</strong> `)
//line template/connection_show.qtpl:41
	qw422016.E().S(p.Token.CreatedAt.Format("Mon 2, Jan 15:04 2006"))
//line template/connection_show.qtpl:41
	qw422016.N().S(`<br/> <strong>Homepage</strong> <a target="_blank" href="`)
//line template/connection_show.qtpl:43
	qw422016.E().S(p.Token.App.HomeURI)
//line template/connection_show.qtpl:43
	qw422016.N().S(`">`)
//line template/connection_show.qtpl:43
	qw422016.E().S(p.Token.App.HomeURI)
//line template/connection_show.qtpl:43
	qw422016.N().S(`</a><br/><br/> `)
//line template/connection_show.qtpl:44
	qw422016.E().S(p.Token.App.Description)
//line template/connection_show.qtpl:44
	qw422016.N().S(` <div class="separator"></div> <h2>Permissions</h2> `)
//line template/connection_show.qtpl:47
	for _, sc := range p.Token.Scope {
//line template/connection_show.qtpl:47
		qw422016.N().S(` `)
//line template/connection_show.qtpl:48
		streamrenderScopeItem(qw422016, sc.Resource, sc.Permission)
//line template/connection_show.qtpl:48
		qw422016.N().S(` `)
//line template/connection_show.qtpl:49
	}
//line template/connection_show.qtpl:49
	qw422016.N().S(` </div> </div> `)
//line template/connection_show.qtpl:52
}

//line template/connection_show.qtpl:52
func (p *ConnectionShow) WriteBody(qq422016 qtio422016.Writer) {
//line template/connection_show.qtpl:52
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/connection_show.qtpl:52
	p.StreamBody(qw422016)
//line template/connection_show.qtpl:52
	qt422016.ReleaseWriter(qw422016)
//line template/connection_show.qtpl:52
}

//line template/connection_show.qtpl:52
func (p *ConnectionShow) Body() string {
//line template/connection_show.qtpl:52
	qb422016 := qt422016.AcquireByteBuffer()
//line template/connection_show.qtpl:52
	p.WriteBody(qb422016)
//line template/connection_show.qtpl:52
	qs422016 := string(qb422016.B)
//line template/connection_show.qtpl:52
	qt422016.ReleaseByteBuffer(qb422016)
//line template/connection_show.qtpl:52
	return qs422016
//line template/connection_show.qtpl:52
}
