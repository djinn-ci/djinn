// This file is automatically generated by qtc from "create.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/build/create.qtpl:2
package build

//line template/build/create.qtpl:2
import (
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/template"
)

//line template/build/create.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/build/create.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/build/create.qtpl:9
type CreatePage struct {
	*template.Page

	Errors form.Errors
	Form   form.Form
}

//line template/build/create.qtpl:17
func (p *CreatePage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/build/create.qtpl:17
	qw422016.N().S(`
Submit Build - Thrall
`)
//line template/build/create.qtpl:19
}

//line template/build/create.qtpl:19
func (p *CreatePage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/build/create.qtpl:19
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/create.qtpl:19
	p.StreamTitle(qw422016)
	//line template/build/create.qtpl:19
	qt422016.ReleaseWriter(qw422016)
//line template/build/create.qtpl:19
}

//line template/build/create.qtpl:19
func (p *CreatePage) Title() string {
	//line template/build/create.qtpl:19
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/create.qtpl:19
	p.WriteTitle(qb422016)
	//line template/build/create.qtpl:19
	qs422016 := string(qb422016.B)
	//line template/build/create.qtpl:19
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/create.qtpl:19
	return qs422016
//line template/build/create.qtpl:19
}

//line template/build/create.qtpl:22
func (p *CreatePage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/create.qtpl:22
	qw422016.N().S(` <div class="header"> <h1> <a href="/" class="back">`)
	//line template/build/create.qtpl:25
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
	//line template/build/create.qtpl:25
	qw422016.N().S(`</a> Submit Build </h1> </div> <div class="body"> <div class="panel"> <form class="slim" method="POST" action="/builds"> `)
	//line template/build/create.qtpl:31
	if p.Errors.First("build") != "" {
		//line template/build/create.qtpl:31
		qw422016.N().S(` <div class="form-error">Failed to submit build: `)
		//line template/build/create.qtpl:32
		qw422016.E().S(p.Errors.First("build"))
		//line template/build/create.qtpl:32
		qw422016.N().S(`</div> `)
		//line template/build/create.qtpl:33
	}
	//line template/build/create.qtpl:33
	qw422016.N().S(` <div class="form-field"> <label class="label">Namespace <small>(optional)</small></label> <input class="text" type="text" name="namespace" autocomplete="off"/> </div> <div class="form-field"> <label class="label">Manifest</label> <textarea class="text" name="manifest"></textarea> <div class="error">`)
	//line template/build/create.qtpl:41
	qw422016.E().S(p.Errors.First("manifest"))
	//line template/build/create.qtpl:41
	qw422016.N().S(`</div> </div> <div class="form-field"> <label class="label">Tags <small>(optional)</small></label> <input class="text" type="text" name="tags" autocomplete="off"/> </div> <div class="form-field"> <button type="submit" class="button button-primary">Submit</button> </div> </form> </div> </div> `)
//line template/build/create.qtpl:53
}

//line template/build/create.qtpl:53
func (p *CreatePage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/create.qtpl:53
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/create.qtpl:53
	p.StreamBody(qw422016)
	//line template/build/create.qtpl:53
	qt422016.ReleaseWriter(qw422016)
//line template/build/create.qtpl:53
}

//line template/build/create.qtpl:53
func (p *CreatePage) Body() string {
	//line template/build/create.qtpl:53
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/create.qtpl:53
	p.WriteBody(qb422016)
	//line template/build/create.qtpl:53
	qs422016 := string(qb422016.B)
	//line template/build/create.qtpl:53
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/create.qtpl:53
	return qs422016
//line template/build/create.qtpl:53
}
