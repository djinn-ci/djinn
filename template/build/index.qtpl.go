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
	template.Page

	Builds []*model.Build
	Status string
}

//line template/build/index.qtpl:18
func (p *IndexPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:18
	qw422016.N().S(` Builds - Thrall `)
//line template/build/index.qtpl:20
}

//line template/build/index.qtpl:20
func (p *IndexPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:20
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:20
	p.StreamTitle(qw422016)
	//line template/build/index.qtpl:20
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:20
}

//line template/build/index.qtpl:20
func (p *IndexPage) Title() string {
	//line template/build/index.qtpl:20
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:20
	p.WriteTitle(qb422016)
	//line template/build/index.qtpl:20
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:20
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:20
	return qs422016
//line template/build/index.qtpl:20
}

//line template/build/index.qtpl:22
func (p *IndexPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:22
	qw422016.N().S(` `)
	//line template/build/index.qtpl:23
	StreamRenderBuildsTable(qw422016, p.Builds, p.Status, p.URI)
	//line template/build/index.qtpl:23
	qw422016.N().S(` `)
//line template/build/index.qtpl:24
}

//line template/build/index.qtpl:24
func (p *IndexPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:24
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:24
	p.StreamBody(qw422016)
	//line template/build/index.qtpl:24
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:24
}

//line template/build/index.qtpl:24
func (p *IndexPage) Body() string {
	//line template/build/index.qtpl:24
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:24
	p.WriteBody(qb422016)
	//line template/build/index.qtpl:24
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:24
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:24
	return qs422016
//line template/build/index.qtpl:24
}

//line template/build/index.qtpl:26
func (p *IndexPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:26
	qw422016.N().S(` Builds `)
//line template/build/index.qtpl:28
}

//line template/build/index.qtpl:28
func (p *IndexPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:28
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:28
	p.StreamHeader(qw422016)
	//line template/build/index.qtpl:28
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:28
}

//line template/build/index.qtpl:28
func (p *IndexPage) Header() string {
	//line template/build/index.qtpl:28
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:28
	p.WriteHeader(qb422016)
	//line template/build/index.qtpl:28
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:28
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:28
	return qs422016
//line template/build/index.qtpl:28
}

//line template/build/index.qtpl:30
func (p *IndexPage) StreamActions(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:30
	qw422016.N().S(` <li><a href="/builds/create" class="btn btn-primary">Submit</a></li> `)
//line template/build/index.qtpl:32
}

//line template/build/index.qtpl:32
func (p *IndexPage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:32
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:32
	p.StreamActions(qw422016)
	//line template/build/index.qtpl:32
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:32
}

//line template/build/index.qtpl:32
func (p *IndexPage) Actions() string {
	//line template/build/index.qtpl:32
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:32
	p.WriteActions(qb422016)
	//line template/build/index.qtpl:32
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:32
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:32
	return qs422016
//line template/build/index.qtpl:32
}

//line template/build/index.qtpl:34
func (p *IndexPage) StreamNavigation(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:34
	qw422016.N().S(` `)
//line template/build/index.qtpl:35
}

//line template/build/index.qtpl:35
func (p *IndexPage) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:35
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:35
	p.StreamNavigation(qw422016)
	//line template/build/index.qtpl:35
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:35
}

//line template/build/index.qtpl:35
func (p *IndexPage) Navigation() string {
	//line template/build/index.qtpl:35
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:35
	p.WriteNavigation(qb422016)
	//line template/build/index.qtpl:35
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:35
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:35
	return qs422016
//line template/build/index.qtpl:35
}
