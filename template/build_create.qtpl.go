// Code generated by qtc from "build_create.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/build_create.qtpl:1
package template

//line template/build_create.qtpl:1
import "djinn-ci.com/template/form"

//line template/build_create.qtpl:3
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/build_create.qtpl:3
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/build_create.qtpl:4
type BuildCreate struct {
	*form.Form
}

//line template/build_create.qtpl:10
func (p *BuildCreate) StreamTitle(qw422016 *qt422016.Writer) {
//line template/build_create.qtpl:10
	qw422016.N().S(`Submit Build`)
//line template/build_create.qtpl:10
}

//line template/build_create.qtpl:10
func (p *BuildCreate) WriteTitle(qq422016 qtio422016.Writer) {
//line template/build_create.qtpl:10
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/build_create.qtpl:10
	p.StreamTitle(qw422016)
//line template/build_create.qtpl:10
	qt422016.ReleaseWriter(qw422016)
//line template/build_create.qtpl:10
}

//line template/build_create.qtpl:10
func (p *BuildCreate) Title() string {
//line template/build_create.qtpl:10
	qb422016 := qt422016.AcquireByteBuffer()
//line template/build_create.qtpl:10
	p.WriteTitle(qb422016)
//line template/build_create.qtpl:10
	qs422016 := string(qb422016.B)
//line template/build_create.qtpl:10
	qt422016.ReleaseByteBuffer(qb422016)
//line template/build_create.qtpl:10
	return qs422016
//line template/build_create.qtpl:10
}

//line template/build_create.qtpl:12
func (p *BuildCreate) StreamHeader(qw422016 *qt422016.Writer) {
//line template/build_create.qtpl:12
	qw422016.N().S(` <a class="back" href="/builds">`)
//line template/build_create.qtpl:13
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
//line template/build_create.qtpl:13
	qw422016.N().S(`</a> `)
//line template/build_create.qtpl:13
	p.StreamTitle(qw422016)
//line template/build_create.qtpl:13
	qw422016.N().S(` `)
//line template/build_create.qtpl:14
}

//line template/build_create.qtpl:14
func (p *BuildCreate) WriteHeader(qq422016 qtio422016.Writer) {
//line template/build_create.qtpl:14
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/build_create.qtpl:14
	p.StreamHeader(qw422016)
//line template/build_create.qtpl:14
	qt422016.ReleaseWriter(qw422016)
//line template/build_create.qtpl:14
}

//line template/build_create.qtpl:14
func (p *BuildCreate) Header() string {
//line template/build_create.qtpl:14
	qb422016 := qt422016.AcquireByteBuffer()
//line template/build_create.qtpl:14
	p.WriteHeader(qb422016)
//line template/build_create.qtpl:14
	qs422016 := string(qb422016.B)
//line template/build_create.qtpl:14
	qt422016.ReleaseByteBuffer(qb422016)
//line template/build_create.qtpl:14
	return qs422016
//line template/build_create.qtpl:14
}

//line template/build_create.qtpl:16
func (p *BuildCreate) StreamActions(qw422016 *qt422016.Writer) {
//line template/build_create.qtpl:16
}

//line template/build_create.qtpl:16
func (p *BuildCreate) WriteActions(qq422016 qtio422016.Writer) {
//line template/build_create.qtpl:16
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/build_create.qtpl:16
	p.StreamActions(qw422016)
//line template/build_create.qtpl:16
	qt422016.ReleaseWriter(qw422016)
//line template/build_create.qtpl:16
}

//line template/build_create.qtpl:16
func (p *BuildCreate) Actions() string {
//line template/build_create.qtpl:16
	qb422016 := qt422016.AcquireByteBuffer()
//line template/build_create.qtpl:16
	p.WriteActions(qb422016)
//line template/build_create.qtpl:16
	qs422016 := string(qb422016.B)
//line template/build_create.qtpl:16
	qt422016.ReleaseByteBuffer(qb422016)
//line template/build_create.qtpl:16
	return qs422016
//line template/build_create.qtpl:16
}

//line template/build_create.qtpl:17
func (p *BuildCreate) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/build_create.qtpl:17
}

//line template/build_create.qtpl:17
func (p *BuildCreate) WriteNavigation(qq422016 qtio422016.Writer) {
//line template/build_create.qtpl:17
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/build_create.qtpl:17
	p.StreamNavigation(qw422016)
//line template/build_create.qtpl:17
	qt422016.ReleaseWriter(qw422016)
//line template/build_create.qtpl:17
}

//line template/build_create.qtpl:17
func (p *BuildCreate) Navigation() string {
//line template/build_create.qtpl:17
	qb422016 := qt422016.AcquireByteBuffer()
//line template/build_create.qtpl:17
	p.WriteNavigation(qb422016)
//line template/build_create.qtpl:17
	qs422016 := string(qb422016.B)
//line template/build_create.qtpl:17
	qt422016.ReleaseByteBuffer(qb422016)
//line template/build_create.qtpl:17
	return qs422016
//line template/build_create.qtpl:17
}

//line template/build_create.qtpl:18
func (p *BuildCreate) StreamFooter(qw422016 *qt422016.Writer) {
//line template/build_create.qtpl:18
}

//line template/build_create.qtpl:18
func (p *BuildCreate) WriteFooter(qq422016 qtio422016.Writer) {
//line template/build_create.qtpl:18
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/build_create.qtpl:18
	p.StreamFooter(qw422016)
//line template/build_create.qtpl:18
	qt422016.ReleaseWriter(qw422016)
//line template/build_create.qtpl:18
}

//line template/build_create.qtpl:18
func (p *BuildCreate) Footer() string {
//line template/build_create.qtpl:18
	qb422016 := qt422016.AcquireByteBuffer()
//line template/build_create.qtpl:18
	p.WriteFooter(qb422016)
//line template/build_create.qtpl:18
	qs422016 := string(qb422016.B)
//line template/build_create.qtpl:18
	qt422016.ReleaseByteBuffer(qb422016)
//line template/build_create.qtpl:18
	return qs422016
//line template/build_create.qtpl:18
}

//line template/build_create.qtpl:20
func (p *BuildCreate) StreamBody(qw422016 *qt422016.Writer) {
//line template/build_create.qtpl:20
	qw422016.N().S(` <div class="panel"> <form action="/builds" class="panel-body slim" method="POST"> `)
//line template/build_create.qtpl:23
	qw422016.N().V(p.CSRF)
//line template/build_create.qtpl:23
	qw422016.N().S(` `)
//line template/build_create.qtpl:24
	p.StreamField(qw422016, form.Field{
		ID:   "manifest",
		Name: "Manifest",
		Type: form.Textarea,
	})
//line template/build_create.qtpl:28
	qw422016.N().S(` `)
//line template/build_create.qtpl:29
	p.StreamField(qw422016, form.Field{
		ID:       "comment",
		Name:     "Comment",
		Type:     form.Textarea,
		Optional: true,
	})
//line template/build_create.qtpl:34
	qw422016.N().S(` <div class="form-field"> <button type="submit" class="btn btn-primary">Submit</button> </div> </form> </div> `)
//line template/build_create.qtpl:40
}

//line template/build_create.qtpl:40
func (p *BuildCreate) WriteBody(qq422016 qtio422016.Writer) {
//line template/build_create.qtpl:40
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/build_create.qtpl:40
	p.StreamBody(qw422016)
//line template/build_create.qtpl:40
	qt422016.ReleaseWriter(qw422016)
//line template/build_create.qtpl:40
}

//line template/build_create.qtpl:40
func (p *BuildCreate) Body() string {
//line template/build_create.qtpl:40
	qb422016 := qt422016.AcquireByteBuffer()
//line template/build_create.qtpl:40
	p.WriteBody(qb422016)
//line template/build_create.qtpl:40
	qs422016 := string(qb422016.B)
//line template/build_create.qtpl:40
	qt422016.ReleaseByteBuffer(qb422016)
//line template/build_create.qtpl:40
	return qs422016
//line template/build_create.qtpl:40
}
