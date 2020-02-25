// Code generated by qtc from "index.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/token/index.qtpl:2
package token

//line template/token/index.qtpl:2
import (
	"fmt"

	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

//line template/token/index.qtpl:10
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/token/index.qtpl:10
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/token/index.qtpl:11
type IndexPage struct {
	template.BasePage

	CSRF   string
	Tokens []*model.Token
}

//line template/token/index.qtpl:20
func (p *IndexPage) StreamTitle(qw422016 *qt422016.Writer) {
//line template/token/index.qtpl:20
	qw422016.N().S(` Settings - Access Tokens `)
//line template/token/index.qtpl:22
}

//line template/token/index.qtpl:22
func (p *IndexPage) WriteTitle(qq422016 qtio422016.Writer) {
//line template/token/index.qtpl:22
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/token/index.qtpl:22
	p.StreamTitle(qw422016)
//line template/token/index.qtpl:22
	qt422016.ReleaseWriter(qw422016)
//line template/token/index.qtpl:22
}

//line template/token/index.qtpl:22
func (p *IndexPage) Title() string {
//line template/token/index.qtpl:22
	qb422016 := qt422016.AcquireByteBuffer()
//line template/token/index.qtpl:22
	p.WriteTitle(qb422016)
//line template/token/index.qtpl:22
	qs422016 := string(qb422016.B)
//line template/token/index.qtpl:22
	qt422016.ReleaseByteBuffer(qb422016)
//line template/token/index.qtpl:22
	return qs422016
//line template/token/index.qtpl:22
}

//line template/token/index.qtpl:24
func (p *IndexPage) StreamBody(qw422016 *qt422016.Writer) {
//line template/token/index.qtpl:24
	qw422016.N().S(` <div class="panel"> `)
//line template/token/index.qtpl:26
	if len(p.Tokens) == 0 {
//line template/token/index.qtpl:26
		qw422016.N().S(` <div class="panel-message muted">No access tokens created.</div> `)
//line template/token/index.qtpl:28
	} else {
//line template/token/index.qtpl:28
		qw422016.N().S(` <table class="table"> <tbody> `)
//line template/token/index.qtpl:31
		for _, t := range p.Tokens {
//line template/token/index.qtpl:31
			qw422016.N().S(` <tr> <td> <strong><a href="`)
//line template/token/index.qtpl:34
			qw422016.E().S(t.UIEndpoint())
//line template/token/index.qtpl:34
			qw422016.N().S(`">`)
//line template/token/index.qtpl:34
			qw422016.E().S(t.Name)
//line template/token/index.qtpl:34
			qw422016.N().S(`</a></strong> `)
//line template/token/index.qtpl:35
			if t.Token != nil {
//line template/token/index.qtpl:35
				qw422016.N().S(` - <span class="muted">`)
//line template/token/index.qtpl:36
				qw422016.E().S(fmt.Sprintf("%x", t.Token))
//line template/token/index.qtpl:36
				qw422016.N().S(`</span> `)
//line template/token/index.qtpl:37
			}
//line template/token/index.qtpl:37
			qw422016.N().S(` </td> <td class="align-right"> <form method="POST" action="/settings/tokens/`)
//line template/token/index.qtpl:40
			qw422016.E().V(t.ID)
//line template/token/index.qtpl:40
			qw422016.N().S(`"> `)
//line template/token/index.qtpl:41
			qw422016.N().S(p.CSRF)
//line template/token/index.qtpl:41
			qw422016.N().S(` <input type="hidden" name="_method" value="DELETE"/> <button type="submit" class="btn btn-danger">Delete</button> </form> </td> </tr> `)
//line template/token/index.qtpl:47
		}
//line template/token/index.qtpl:47
		qw422016.N().S(` </tbody> </table> `)
//line template/token/index.qtpl:50
	}
//line template/token/index.qtpl:50
	qw422016.N().S(` </div> `)
//line template/token/index.qtpl:52
}

//line template/token/index.qtpl:52
func (p *IndexPage) WriteBody(qq422016 qtio422016.Writer) {
//line template/token/index.qtpl:52
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/token/index.qtpl:52
	p.StreamBody(qw422016)
//line template/token/index.qtpl:52
	qt422016.ReleaseWriter(qw422016)
//line template/token/index.qtpl:52
}

//line template/token/index.qtpl:52
func (p *IndexPage) Body() string {
//line template/token/index.qtpl:52
	qb422016 := qt422016.AcquireByteBuffer()
//line template/token/index.qtpl:52
	p.WriteBody(qb422016)
//line template/token/index.qtpl:52
	qs422016 := string(qb422016.B)
//line template/token/index.qtpl:52
	qt422016.ReleaseByteBuffer(qb422016)
//line template/token/index.qtpl:52
	return qs422016
//line template/token/index.qtpl:52
}

//line template/token/index.qtpl:54
func (p *IndexPage) StreamHeader(qw422016 *qt422016.Writer) {
//line template/token/index.qtpl:54
	qw422016.N().S(` Tokens `)
//line template/token/index.qtpl:56
}

//line template/token/index.qtpl:56
func (p *IndexPage) WriteHeader(qq422016 qtio422016.Writer) {
//line template/token/index.qtpl:56
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/token/index.qtpl:56
	p.StreamHeader(qw422016)
//line template/token/index.qtpl:56
	qt422016.ReleaseWriter(qw422016)
//line template/token/index.qtpl:56
}

//line template/token/index.qtpl:56
func (p *IndexPage) Header() string {
//line template/token/index.qtpl:56
	qb422016 := qt422016.AcquireByteBuffer()
//line template/token/index.qtpl:56
	p.WriteHeader(qb422016)
//line template/token/index.qtpl:56
	qs422016 := string(qb422016.B)
//line template/token/index.qtpl:56
	qt422016.ReleaseByteBuffer(qb422016)
//line template/token/index.qtpl:56
	return qs422016
//line template/token/index.qtpl:56
}

//line template/token/index.qtpl:58
func (p *IndexPage) StreamActions(qw422016 *qt422016.Writer) {
//line template/token/index.qtpl:58
	qw422016.N().S(` <li> <form method="POST" action="/settings/tokens/revoke"> `)
//line template/token/index.qtpl:61
	qw422016.N().S(p.CSRF)
//line template/token/index.qtpl:61
	qw422016.N().S(` <input type="hidden" name="_method" value="DELETE"/> <button type="submit" class="btn btn-danger">Revoke All</button> </form> </li> <li><a href="/settings/tokens/create" class="btn btn-primary">Create</a></li> `)
//line template/token/index.qtpl:67
}

//line template/token/index.qtpl:67
func (p *IndexPage) WriteActions(qq422016 qtio422016.Writer) {
//line template/token/index.qtpl:67
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/token/index.qtpl:67
	p.StreamActions(qw422016)
//line template/token/index.qtpl:67
	qt422016.ReleaseWriter(qw422016)
//line template/token/index.qtpl:67
}

//line template/token/index.qtpl:67
func (p *IndexPage) Actions() string {
//line template/token/index.qtpl:67
	qb422016 := qt422016.AcquireByteBuffer()
//line template/token/index.qtpl:67
	p.WriteActions(qb422016)
//line template/token/index.qtpl:67
	qs422016 := string(qb422016.B)
//line template/token/index.qtpl:67
	qt422016.ReleaseByteBuffer(qb422016)
//line template/token/index.qtpl:67
	return qs422016
//line template/token/index.qtpl:67
}

//line template/token/index.qtpl:69
func (p *IndexPage) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/token/index.qtpl:69
}

//line template/token/index.qtpl:69
func (p *IndexPage) WriteNavigation(qq422016 qtio422016.Writer) {
//line template/token/index.qtpl:69
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/token/index.qtpl:69
	p.StreamNavigation(qw422016)
//line template/token/index.qtpl:69
	qt422016.ReleaseWriter(qw422016)
//line template/token/index.qtpl:69
}

//line template/token/index.qtpl:69
func (p *IndexPage) Navigation() string {
//line template/token/index.qtpl:69
	qb422016 := qt422016.AcquireByteBuffer()
//line template/token/index.qtpl:69
	p.WriteNavigation(qb422016)
//line template/token/index.qtpl:69
	qs422016 := string(qb422016.B)
//line template/token/index.qtpl:69
	qt422016.ReleaseByteBuffer(qb422016)
//line template/token/index.qtpl:69
	return qs422016
//line template/token/index.qtpl:69
}