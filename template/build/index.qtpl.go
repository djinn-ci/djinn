// This file is automatically generated by qtc from "index.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/build/index.qtpl:2
package build

//line template/build/index.qtpl:2
import (
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

//line template/build/index.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/build/index.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/build/index.qtpl:9
type IndexPage struct {
	template.BasePage

	Builds []*model.Build
	Search string
	Status string
	Tag    string
}

//line template/build/index.qtpl:20
func (p *IndexPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:20
	qw422016.N().S(` Builds - Thrall `)
//line template/build/index.qtpl:22
}

//line template/build/index.qtpl:22
func (p *IndexPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:22
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:22
	p.StreamTitle(qw422016)
	//line template/build/index.qtpl:22
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:22
}

//line template/build/index.qtpl:22
func (p *IndexPage) Title() string {
	//line template/build/index.qtpl:22
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:22
	p.WriteTitle(qb422016)
	//line template/build/index.qtpl:22
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:22
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:22
	return qs422016
//line template/build/index.qtpl:22
}

//line template/build/index.qtpl:24
func (p *IndexPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:24
	qw422016.N().S(` <div class="panel">`)
	//line template/build/index.qtpl:25
	StreamRenderIndex(qw422016, p.Builds, p.URL.Path, p.Status, p.Search)
	//line template/build/index.qtpl:25
	qw422016.N().S(`</div> `)
//line template/build/index.qtpl:26
}

//line template/build/index.qtpl:26
func (p *IndexPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:26
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:26
	p.StreamBody(qw422016)
	//line template/build/index.qtpl:26
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:26
}

//line template/build/index.qtpl:26
func (p *IndexPage) Body() string {
	//line template/build/index.qtpl:26
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:26
	p.WriteBody(qb422016)
	//line template/build/index.qtpl:26
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:26
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:26
	return qs422016
//line template/build/index.qtpl:26
}

//line template/build/index.qtpl:28
func (p *IndexPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:28
	qw422016.N().S(` Builds `)
	//line template/build/index.qtpl:30
	if p.Tag != "" {
		//line template/build/index.qtpl:30
		qw422016.N().S(` <span class="pill pill-light">`)
		//line template/build/index.qtpl:31
		qw422016.E().S(p.Tag)
		//line template/build/index.qtpl:31
		qw422016.N().S(`<a href="`)
		//line template/build/index.qtpl:31
		qw422016.E().S(p.URL.Path)
		//line template/build/index.qtpl:31
		qw422016.N().S(`">`)
		//line template/build/index.qtpl:31
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M18.984 6.422l-5.578 5.578 5.578 5.578-1.406 1.406-5.578-5.578-5.578 5.578-1.406-1.406 5.578-5.578-5.578-5.578 1.406-1.406 5.578 5.578 5.578-5.578z"></path>
</svg>
`)
		//line template/build/index.qtpl:31
		qw422016.N().S(`</a></span> `)
		//line template/build/index.qtpl:32
	}
	//line template/build/index.qtpl:32
	qw422016.N().S(` `)
//line template/build/index.qtpl:33
}

//line template/build/index.qtpl:33
func (p *IndexPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:33
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:33
	p.StreamHeader(qw422016)
	//line template/build/index.qtpl:33
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:33
}

//line template/build/index.qtpl:33
func (p *IndexPage) Header() string {
	//line template/build/index.qtpl:33
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:33
	p.WriteHeader(qb422016)
	//line template/build/index.qtpl:33
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:33
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:33
	return qs422016
//line template/build/index.qtpl:33
}

//line template/build/index.qtpl:35
func (p *IndexPage) StreamActions(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:35
	qw422016.N().S(` <li><a href="/builds/create" class="btn btn-primary">Submit</a></li> `)
//line template/build/index.qtpl:37
}

//line template/build/index.qtpl:37
func (p *IndexPage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:37
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:37
	p.StreamActions(qw422016)
	//line template/build/index.qtpl:37
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:37
}

//line template/build/index.qtpl:37
func (p *IndexPage) Actions() string {
	//line template/build/index.qtpl:37
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:37
	p.WriteActions(qb422016)
	//line template/build/index.qtpl:37
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:37
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:37
	return qs422016
//line template/build/index.qtpl:37
}

//line template/build/index.qtpl:39
func (p *IndexPage) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/build/index.qtpl:39
}

//line template/build/index.qtpl:39
func (p *IndexPage) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:39
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:39
	p.StreamNavigation(qw422016)
	//line template/build/index.qtpl:39
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:39
}

//line template/build/index.qtpl:39
func (p *IndexPage) Navigation() string {
	//line template/build/index.qtpl:39
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:39
	p.WriteNavigation(qb422016)
	//line template/build/index.qtpl:39
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:39
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:39
	return qs422016
//line template/build/index.qtpl:39
}
