// This file is automatically generated by qtc from "error.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/error.qtpl:3
package template

//line template/error.qtpl:3
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/error.qtpl:3
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/error.qtpl:4
type Error struct {
	*Page

	Code    int
	Message string
}

//line template/error.qtpl:12
func (p *Error) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/error.qtpl:12
	qw422016.N().S(`
Thrall - Error
`)
//line template/error.qtpl:14
}

//line template/error.qtpl:14
func (p *Error) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/error.qtpl:14
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/error.qtpl:14
	p.StreamTitle(qw422016)
	//line template/error.qtpl:14
	qt422016.ReleaseWriter(qw422016)
//line template/error.qtpl:14
}

//line template/error.qtpl:14
func (p *Error) Title() string {
	//line template/error.qtpl:14
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/error.qtpl:14
	p.WriteTitle(qb422016)
	//line template/error.qtpl:14
	qs422016 := string(qb422016.B)
	//line template/error.qtpl:14
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/error.qtpl:14
	return qs422016
//line template/error.qtpl:14
}

//line template/error.qtpl:16
func (p *Error) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/error.qtpl:16
	qw422016.N().S(`
<link rel="stylesheet" type="text/css" href="/assets/css/error.css">
`)
//line template/error.qtpl:18
}

//line template/error.qtpl:18
func (p *Error) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/error.qtpl:18
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/error.qtpl:18
	p.StreamHeader(qw422016)
	//line template/error.qtpl:18
	qt422016.ReleaseWriter(qw422016)
//line template/error.qtpl:18
}

//line template/error.qtpl:18
func (p *Error) Header() string {
	//line template/error.qtpl:18
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/error.qtpl:18
	p.WriteHeader(qb422016)
	//line template/error.qtpl:18
	qs422016 := string(qb422016.B)
	//line template/error.qtpl:18
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/error.qtpl:18
	return qs422016
//line template/error.qtpl:18
}

//line template/error.qtpl:20
func (p *Error) StreamBody(qw422016 *qt422016.Writer) {
	//line template/error.qtpl:20
	qw422016.N().S(`
<div class="error">
	<h1>`)
	//line template/error.qtpl:22
	qw422016.E().V(p.Code)
	//line template/error.qtpl:22
	qw422016.N().S(`</h1>
	<h2>`)
	//line template/error.qtpl:23
	qw422016.E().S(p.Message)
	//line template/error.qtpl:23
	qw422016.N().S(`</h2>
</div>
`)
//line template/error.qtpl:25
}

//line template/error.qtpl:25
func (p *Error) WriteBody(qq422016 qtio422016.Writer) {
	//line template/error.qtpl:25
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/error.qtpl:25
	p.StreamBody(qw422016)
	//line template/error.qtpl:25
	qt422016.ReleaseWriter(qw422016)
//line template/error.qtpl:25
}

//line template/error.qtpl:25
func (p *Error) Body() string {
	//line template/error.qtpl:25
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/error.qtpl:25
	p.WriteBody(qb422016)
	//line template/error.qtpl:25
	qs422016 := string(qb422016.B)
	//line template/error.qtpl:25
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/error.qtpl:25
	return qs422016
//line template/error.qtpl:25
}