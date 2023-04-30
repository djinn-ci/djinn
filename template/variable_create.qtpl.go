// Code generated by qtc from "variable_create.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/variable_create.qtpl:2
package template

//line template/variable_create.qtpl:2
import (
	"djinn-ci.com/template/form"
	"djinn-ci.com/variable"
)

//line template/variable_create.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/variable_create.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/variable_create.qtpl:9
type VariableCreate struct {
	*form.Form
}

//line template/variable_create.qtpl:15
func (p *VariableCreate) StreamTitle(qw422016 *qt422016.Writer) {
//line template/variable_create.qtpl:15
	qw422016.N().S(`Create Variable`)
//line template/variable_create.qtpl:15
}

//line template/variable_create.qtpl:15
func (p *VariableCreate) WriteTitle(qq422016 qtio422016.Writer) {
//line template/variable_create.qtpl:15
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/variable_create.qtpl:15
	p.StreamTitle(qw422016)
//line template/variable_create.qtpl:15
	qt422016.ReleaseWriter(qw422016)
//line template/variable_create.qtpl:15
}

//line template/variable_create.qtpl:15
func (p *VariableCreate) Title() string {
//line template/variable_create.qtpl:15
	qb422016 := qt422016.AcquireByteBuffer()
//line template/variable_create.qtpl:15
	p.WriteTitle(qb422016)
//line template/variable_create.qtpl:15
	qs422016 := string(qb422016.B)
//line template/variable_create.qtpl:15
	qt422016.ReleaseByteBuffer(qb422016)
//line template/variable_create.qtpl:15
	return qs422016
//line template/variable_create.qtpl:15
}

//line template/variable_create.qtpl:17
func (p *VariableCreate) StreamHeader(qw422016 *qt422016.Writer) {
//line template/variable_create.qtpl:17
	qw422016.N().S(` <a class="back" href="/variables">`)
//line template/variable_create.qtpl:18
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
//line template/variable_create.qtpl:18
	qw422016.N().S(`</a> `)
//line template/variable_create.qtpl:18
	p.StreamTitle(qw422016)
//line template/variable_create.qtpl:18
	qw422016.N().S(` `)
//line template/variable_create.qtpl:19
}

//line template/variable_create.qtpl:19
func (p *VariableCreate) WriteHeader(qq422016 qtio422016.Writer) {
//line template/variable_create.qtpl:19
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/variable_create.qtpl:19
	p.StreamHeader(qw422016)
//line template/variable_create.qtpl:19
	qt422016.ReleaseWriter(qw422016)
//line template/variable_create.qtpl:19
}

//line template/variable_create.qtpl:19
func (p *VariableCreate) Header() string {
//line template/variable_create.qtpl:19
	qb422016 := qt422016.AcquireByteBuffer()
//line template/variable_create.qtpl:19
	p.WriteHeader(qb422016)
//line template/variable_create.qtpl:19
	qs422016 := string(qb422016.B)
//line template/variable_create.qtpl:19
	qt422016.ReleaseByteBuffer(qb422016)
//line template/variable_create.qtpl:19
	return qs422016
//line template/variable_create.qtpl:19
}

//line template/variable_create.qtpl:21
func (p *VariableCreate) StreamActions(qw422016 *qt422016.Writer) {
//line template/variable_create.qtpl:21
}

//line template/variable_create.qtpl:21
func (p *VariableCreate) WriteActions(qq422016 qtio422016.Writer) {
//line template/variable_create.qtpl:21
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/variable_create.qtpl:21
	p.StreamActions(qw422016)
//line template/variable_create.qtpl:21
	qt422016.ReleaseWriter(qw422016)
//line template/variable_create.qtpl:21
}

//line template/variable_create.qtpl:21
func (p *VariableCreate) Actions() string {
//line template/variable_create.qtpl:21
	qb422016 := qt422016.AcquireByteBuffer()
//line template/variable_create.qtpl:21
	p.WriteActions(qb422016)
//line template/variable_create.qtpl:21
	qs422016 := string(qb422016.B)
//line template/variable_create.qtpl:21
	qt422016.ReleaseByteBuffer(qb422016)
//line template/variable_create.qtpl:21
	return qs422016
//line template/variable_create.qtpl:21
}

//line template/variable_create.qtpl:22
func (p *VariableCreate) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/variable_create.qtpl:22
}

//line template/variable_create.qtpl:22
func (p *VariableCreate) WriteNavigation(qq422016 qtio422016.Writer) {
//line template/variable_create.qtpl:22
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/variable_create.qtpl:22
	p.StreamNavigation(qw422016)
//line template/variable_create.qtpl:22
	qt422016.ReleaseWriter(qw422016)
//line template/variable_create.qtpl:22
}

//line template/variable_create.qtpl:22
func (p *VariableCreate) Navigation() string {
//line template/variable_create.qtpl:22
	qb422016 := qt422016.AcquireByteBuffer()
//line template/variable_create.qtpl:22
	p.WriteNavigation(qb422016)
//line template/variable_create.qtpl:22
	qs422016 := string(qb422016.B)
//line template/variable_create.qtpl:22
	qt422016.ReleaseByteBuffer(qb422016)
//line template/variable_create.qtpl:22
	return qs422016
//line template/variable_create.qtpl:22
}

//line template/variable_create.qtpl:23
func (p *VariableCreate) StreamFooter(qw422016 *qt422016.Writer) {
//line template/variable_create.qtpl:23
}

//line template/variable_create.qtpl:23
func (p *VariableCreate) WriteFooter(qq422016 qtio422016.Writer) {
//line template/variable_create.qtpl:23
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/variable_create.qtpl:23
	p.StreamFooter(qw422016)
//line template/variable_create.qtpl:23
	qt422016.ReleaseWriter(qw422016)
//line template/variable_create.qtpl:23
}

//line template/variable_create.qtpl:23
func (p *VariableCreate) Footer() string {
//line template/variable_create.qtpl:23
	qb422016 := qt422016.AcquireByteBuffer()
//line template/variable_create.qtpl:23
	p.WriteFooter(qb422016)
//line template/variable_create.qtpl:23
	qs422016 := string(qb422016.B)
//line template/variable_create.qtpl:23
	qt422016.ReleaseByteBuffer(qb422016)
//line template/variable_create.qtpl:23
	return qs422016
//line template/variable_create.qtpl:23
}

//line template/variable_create.qtpl:25
func (p *VariableCreate) StreamBody(qw422016 *qt422016.Writer) {
//line template/variable_create.qtpl:25
	qw422016.N().S(` <div class="panel"> <form class="panel-body slim" method="POST" action="/variables"> `)
//line template/variable_create.qtpl:28
	qw422016.N().V(p.CSRF)
//line template/variable_create.qtpl:28
	qw422016.N().S(` `)
//line template/variable_create.qtpl:29
	p.StreamField(qw422016, form.Field{
		ID:       "namespace",
		Name:     "Namespace",
		Optional: true,
		Type:     form.Text,
	})
//line template/variable_create.qtpl:34
	qw422016.N().S(` `)
//line template/variable_create.qtpl:35
	p.StreamField(qw422016, form.Field{
		ID:   "key",
		Name: "Key",
		Type: form.Text,
	})
//line template/variable_create.qtpl:39
	qw422016.N().S(` `)
//line template/variable_create.qtpl:40
	p.StreamField(qw422016, form.Field{
		ID:   "value",
		Name: "Value",
		Type: form.Text,
	})
//line template/variable_create.qtpl:44
	qw422016.N().S(` `)
//line template/variable_create.qtpl:45
	p.StreamField(qw422016, form.Field{
		ID:   "mask",
		Name: "Mask variable",
		Desc: `Mask the variable and replace it with <span class="code">` + variable.MaskString + `</span> in the build logs`,
		Type: form.Checkbox,
	})
//line template/variable_create.qtpl:50
	qw422016.N().S(` <div class="form-field"> <button type="submit" class="btn btn-primary">Submit</button> </div> </form> </div> `)
//line template/variable_create.qtpl:56
}

//line template/variable_create.qtpl:56
func (p *VariableCreate) WriteBody(qq422016 qtio422016.Writer) {
//line template/variable_create.qtpl:56
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/variable_create.qtpl:56
	p.StreamBody(qw422016)
//line template/variable_create.qtpl:56
	qt422016.ReleaseWriter(qw422016)
//line template/variable_create.qtpl:56
}

//line template/variable_create.qtpl:56
func (p *VariableCreate) Body() string {
//line template/variable_create.qtpl:56
	qb422016 := qt422016.AcquireByteBuffer()
//line template/variable_create.qtpl:56
	p.WriteBody(qb422016)
//line template/variable_create.qtpl:56
	qs422016 := string(qb422016.B)
//line template/variable_create.qtpl:56
	qt422016.ReleaseByteBuffer(qb422016)
//line template/variable_create.qtpl:56
	return qs422016
//line template/variable_create.qtpl:56
}
