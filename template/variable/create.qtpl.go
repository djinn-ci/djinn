// This file is automatically generated by qtc from "create.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/variable/create.qtpl:2
package variable

//line template/variable/create.qtpl:2
import "github.com/andrewpillar/thrall/template"

//line template/variable/create.qtpl:5
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/variable/create.qtpl:5
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/variable/create.qtpl:6
type CreatePage struct {
	template.Page
	template.Form
}

//line template/variable/create.qtpl:13
func (p *CreatePage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/variable/create.qtpl:13
	qw422016.N().S(` Create Variable - Thrall `)
//line template/variable/create.qtpl:15
}

//line template/variable/create.qtpl:15
func (p *CreatePage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/variable/create.qtpl:15
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/variable/create.qtpl:15
	p.StreamTitle(qw422016)
	//line template/variable/create.qtpl:15
	qt422016.ReleaseWriter(qw422016)
//line template/variable/create.qtpl:15
}

//line template/variable/create.qtpl:15
func (p *CreatePage) Title() string {
	//line template/variable/create.qtpl:15
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/variable/create.qtpl:15
	p.WriteTitle(qb422016)
	//line template/variable/create.qtpl:15
	qs422016 := string(qb422016.B)
	//line template/variable/create.qtpl:15
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/variable/create.qtpl:15
	return qs422016
//line template/variable/create.qtpl:15
}

//line template/variable/create.qtpl:17
func (p *CreatePage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/variable/create.qtpl:17
	qw422016.N().S(` <div class="panel"> <form class="panel-body slim" method="POST" action="/variables"> `)
	//line template/variable/create.qtpl:20
	qw422016.N().S(string(p.CSRF))
	//line template/variable/create.qtpl:20
	qw422016.N().S(` <div class="form-field"> <label class="label" for="key">Key</label> <input class="form-text" type="text" id="key" name="key" value="`)
	//line template/variable/create.qtpl:23
	qw422016.E().S(p.Field("key"))
	//line template/variable/create.qtpl:23
	qw422016.N().S(`" autocomplete="off"/> `)
	//line template/variable/create.qtpl:24
	p.StreamError(qw422016, "key")
	//line template/variable/create.qtpl:24
	qw422016.N().S(` </div> <div class="form-field"> <label class="label" for="value">Value</label> <input class="form-text" type="text" id="value" name="value" value="`)
	//line template/variable/create.qtpl:28
	qw422016.E().S(p.Field("value"))
	//line template/variable/create.qtpl:28
	qw422016.N().S(`" autocomplete="off"/> `)
	//line template/variable/create.qtpl:29
	p.StreamError(qw422016, "value")
	//line template/variable/create.qtpl:29
	qw422016.N().S(` </div> <div class="form-field"> <button type="submit" class="btn btn-primary">Create</button> </div> </form> </div> `)
//line template/variable/create.qtpl:36
}

//line template/variable/create.qtpl:36
func (p *CreatePage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/variable/create.qtpl:36
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/variable/create.qtpl:36
	p.StreamBody(qw422016)
	//line template/variable/create.qtpl:36
	qt422016.ReleaseWriter(qw422016)
//line template/variable/create.qtpl:36
}

//line template/variable/create.qtpl:36
func (p *CreatePage) Body() string {
	//line template/variable/create.qtpl:36
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/variable/create.qtpl:36
	p.WriteBody(qb422016)
	//line template/variable/create.qtpl:36
	qs422016 := string(qb422016.B)
	//line template/variable/create.qtpl:36
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/variable/create.qtpl:36
	return qs422016
//line template/variable/create.qtpl:36
}

//line template/variable/create.qtpl:38
func (p *CreatePage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/variable/create.qtpl:38
	qw422016.N().S(` <a href="/variables" class="back">`)
	//line template/variable/create.qtpl:39
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
	//line template/variable/create.qtpl:39
	qw422016.N().S(`</a> Create Variable `)
//line template/variable/create.qtpl:40
}

//line template/variable/create.qtpl:40
func (p *CreatePage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/variable/create.qtpl:40
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/variable/create.qtpl:40
	p.StreamHeader(qw422016)
	//line template/variable/create.qtpl:40
	qt422016.ReleaseWriter(qw422016)
//line template/variable/create.qtpl:40
}

//line template/variable/create.qtpl:40
func (p *CreatePage) Header() string {
	//line template/variable/create.qtpl:40
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/variable/create.qtpl:40
	p.WriteHeader(qb422016)
	//line template/variable/create.qtpl:40
	qs422016 := string(qb422016.B)
	//line template/variable/create.qtpl:40
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/variable/create.qtpl:40
	return qs422016
//line template/variable/create.qtpl:40
}

//line template/variable/create.qtpl:42
func (p *CreatePage) StreamActions(qw422016 *qt422016.Writer) {
//line template/variable/create.qtpl:42
}

//line template/variable/create.qtpl:42
func (p *CreatePage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/variable/create.qtpl:42
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/variable/create.qtpl:42
	p.StreamActions(qw422016)
	//line template/variable/create.qtpl:42
	qt422016.ReleaseWriter(qw422016)
//line template/variable/create.qtpl:42
}

//line template/variable/create.qtpl:42
func (p *CreatePage) Actions() string {
	//line template/variable/create.qtpl:42
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/variable/create.qtpl:42
	p.WriteActions(qb422016)
	//line template/variable/create.qtpl:42
	qs422016 := string(qb422016.B)
	//line template/variable/create.qtpl:42
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/variable/create.qtpl:42
	return qs422016
//line template/variable/create.qtpl:42
}

//line template/variable/create.qtpl:43
func (p *CreatePage) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/variable/create.qtpl:43
}

//line template/variable/create.qtpl:43
func (p *CreatePage) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/variable/create.qtpl:43
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/variable/create.qtpl:43
	p.StreamNavigation(qw422016)
	//line template/variable/create.qtpl:43
	qt422016.ReleaseWriter(qw422016)
//line template/variable/create.qtpl:43
}

//line template/variable/create.qtpl:43
func (p *CreatePage) Navigation() string {
	//line template/variable/create.qtpl:43
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/variable/create.qtpl:43
	p.WriteNavigation(qb422016)
	//line template/variable/create.qtpl:43
	qs422016 := string(qb422016.B)
	//line template/variable/create.qtpl:43
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/variable/create.qtpl:43
	return qs422016
//line template/variable/create.qtpl:43
}
