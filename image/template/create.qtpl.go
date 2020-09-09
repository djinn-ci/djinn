// Code generated by qtc from "create.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line image/template/create.qtpl:1
package template

//line image/template/create.qtpl:1
import "github.com/andrewpillar/djinn/template"

//line image/template/create.qtpl:3
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line image/template/create.qtpl:3
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line image/template/create.qtpl:4
type Create struct {
	template.BasePage
	template.Form
}

//line image/template/create.qtpl:11
func (p *Create) StreamTitle(qw422016 *qt422016.Writer) {
//line image/template/create.qtpl:11
	qw422016.N().S(` Add Image - Thrall `)
//line image/template/create.qtpl:13
}

//line image/template/create.qtpl:13
func (p *Create) WriteTitle(qq422016 qtio422016.Writer) {
//line image/template/create.qtpl:13
	qw422016 := qt422016.AcquireWriter(qq422016)
//line image/template/create.qtpl:13
	p.StreamTitle(qw422016)
//line image/template/create.qtpl:13
	qt422016.ReleaseWriter(qw422016)
//line image/template/create.qtpl:13
}

//line image/template/create.qtpl:13
func (p *Create) Title() string {
//line image/template/create.qtpl:13
	qb422016 := qt422016.AcquireByteBuffer()
//line image/template/create.qtpl:13
	p.WriteTitle(qb422016)
//line image/template/create.qtpl:13
	qs422016 := string(qb422016.B)
//line image/template/create.qtpl:13
	qt422016.ReleaseByteBuffer(qb422016)
//line image/template/create.qtpl:13
	return qs422016
//line image/template/create.qtpl:13
}

//line image/template/create.qtpl:15
func (p *Create) StreamBody(qw422016 *qt422016.Writer) {
//line image/template/create.qtpl:15
	qw422016.N().S(` <div class="panel"> <form class="panel-body slim" method="POST" action="/images" enctype="multipart/form-data"> `)
//line image/template/create.qtpl:18
	qw422016.N().S(string(p.CSRF))
//line image/template/create.qtpl:18
	qw422016.N().S(` <div class="form-field"> <label class="label" for="namespace">Namespace <small>(optional)</small></label> <input class="form-text" type="text" id="namescpace" name="namespace" value="`)
//line image/template/create.qtpl:21
	qw422016.E().S(p.Fields["namespace"])
//line image/template/create.qtpl:21
	qw422016.N().S(`" autocomplete="off"/> </div> <div class="form-field"> <label class="label" for="name">Name</label> <input class="form-text" type="text" id="name" name="name" value="`)
//line image/template/create.qtpl:25
	qw422016.E().S(p.Fields["name"])
//line image/template/create.qtpl:25
	qw422016.N().S(`" autocomplete="off"/> `)
//line image/template/create.qtpl:26
	p.StreamError(qw422016, "name")
//line image/template/create.qtpl:26
	qw422016.N().S(` </div> `)
//line image/template/create.qtpl:28
	template.StreamFileInput(qw422016, p.Form)
//line image/template/create.qtpl:28
	qw422016.N().S(` <div class="form-field"> <button type="submit" class="btn btn-primary">Create</button> </div> </form> </div> `)
//line image/template/create.qtpl:34
}

//line image/template/create.qtpl:34
func (p *Create) WriteBody(qq422016 qtio422016.Writer) {
//line image/template/create.qtpl:34
	qw422016 := qt422016.AcquireWriter(qq422016)
//line image/template/create.qtpl:34
	p.StreamBody(qw422016)
//line image/template/create.qtpl:34
	qt422016.ReleaseWriter(qw422016)
//line image/template/create.qtpl:34
}

//line image/template/create.qtpl:34
func (p *Create) Body() string {
//line image/template/create.qtpl:34
	qb422016 := qt422016.AcquireByteBuffer()
//line image/template/create.qtpl:34
	p.WriteBody(qb422016)
//line image/template/create.qtpl:34
	qs422016 := string(qb422016.B)
//line image/template/create.qtpl:34
	qt422016.ReleaseByteBuffer(qb422016)
//line image/template/create.qtpl:34
	return qs422016
//line image/template/create.qtpl:34
}

//line image/template/create.qtpl:36
func (p *Create) StreamHeader(qw422016 *qt422016.Writer) {
//line image/template/create.qtpl:36
	qw422016.N().S(` <a href="/images" class="back">`)
//line image/template/create.qtpl:37
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
//line image/template/create.qtpl:37
	qw422016.N().S(`</a> Add Image `)
//line image/template/create.qtpl:38
}

//line image/template/create.qtpl:38
func (p *Create) WriteHeader(qq422016 qtio422016.Writer) {
//line image/template/create.qtpl:38
	qw422016 := qt422016.AcquireWriter(qq422016)
//line image/template/create.qtpl:38
	p.StreamHeader(qw422016)
//line image/template/create.qtpl:38
	qt422016.ReleaseWriter(qw422016)
//line image/template/create.qtpl:38
}

//line image/template/create.qtpl:38
func (p *Create) Header() string {
//line image/template/create.qtpl:38
	qb422016 := qt422016.AcquireByteBuffer()
//line image/template/create.qtpl:38
	p.WriteHeader(qb422016)
//line image/template/create.qtpl:38
	qs422016 := string(qb422016.B)
//line image/template/create.qtpl:38
	qt422016.ReleaseByteBuffer(qb422016)
//line image/template/create.qtpl:38
	return qs422016
//line image/template/create.qtpl:38
}

//line image/template/create.qtpl:40
func (p *Create) StreamActions(qw422016 *qt422016.Writer) {
//line image/template/create.qtpl:40
}

//line image/template/create.qtpl:40
func (p *Create) WriteActions(qq422016 qtio422016.Writer) {
//line image/template/create.qtpl:40
	qw422016 := qt422016.AcquireWriter(qq422016)
//line image/template/create.qtpl:40
	p.StreamActions(qw422016)
//line image/template/create.qtpl:40
	qt422016.ReleaseWriter(qw422016)
//line image/template/create.qtpl:40
}

//line image/template/create.qtpl:40
func (p *Create) Actions() string {
//line image/template/create.qtpl:40
	qb422016 := qt422016.AcquireByteBuffer()
//line image/template/create.qtpl:40
	p.WriteActions(qb422016)
//line image/template/create.qtpl:40
	qs422016 := string(qb422016.B)
//line image/template/create.qtpl:40
	qt422016.ReleaseByteBuffer(qb422016)
//line image/template/create.qtpl:40
	return qs422016
//line image/template/create.qtpl:40
}

//line image/template/create.qtpl:41
func (p *Create) StreamNavigation(qw422016 *qt422016.Writer) {
//line image/template/create.qtpl:41
}

//line image/template/create.qtpl:41
func (p *Create) WriteNavigation(qq422016 qtio422016.Writer) {
//line image/template/create.qtpl:41
	qw422016 := qt422016.AcquireWriter(qq422016)
//line image/template/create.qtpl:41
	p.StreamNavigation(qw422016)
//line image/template/create.qtpl:41
	qt422016.ReleaseWriter(qw422016)
//line image/template/create.qtpl:41
}

//line image/template/create.qtpl:41
func (p *Create) Navigation() string {
//line image/template/create.qtpl:41
	qb422016 := qt422016.AcquireByteBuffer()
//line image/template/create.qtpl:41
	p.WriteNavigation(qb422016)
//line image/template/create.qtpl:41
	qs422016 := string(qb422016.B)
//line image/template/create.qtpl:41
	qt422016.ReleaseByteBuffer(qb422016)
//line image/template/create.qtpl:41
	return qs422016
//line image/template/create.qtpl:41
}
