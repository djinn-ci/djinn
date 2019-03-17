// This file is automatically generated by qtc from "index.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/namespace/index.qtpl:2
package namespace

//line template/namespace/index.qtpl:2
import (
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

//line template/namespace/index.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/namespace/index.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/namespace/index.qtpl:9
type IndexPage struct {
	*template.Page

	Namespaces []*model.Namespace
	Search     string
}

//line template/namespace/index.qtpl:19
func (p *IndexPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/namespace/index.qtpl:19
	qw422016.N().S(` Namespaces - Thrall `)
//line template/namespace/index.qtpl:21
}

//line template/namespace/index.qtpl:21
func (p *IndexPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/namespace/index.qtpl:21
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/index.qtpl:21
	p.StreamTitle(qw422016)
	//line template/namespace/index.qtpl:21
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/index.qtpl:21
}

//line template/namespace/index.qtpl:21
func (p *IndexPage) Title() string {
	//line template/namespace/index.qtpl:21
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/index.qtpl:21
	p.WriteTitle(qb422016)
	//line template/namespace/index.qtpl:21
	qs422016 := string(qb422016.B)
	//line template/namespace/index.qtpl:21
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/index.qtpl:21
	return qs422016
//line template/namespace/index.qtpl:21
}

//line template/namespace/index.qtpl:23
func (p *IndexPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/namespace/index.qtpl:23
	qw422016.N().S(` `)
	//line template/namespace/index.qtpl:24
	streamrenderNamespaces(qw422016, p.Namespaces, p.URI, p.Search)
	//line template/namespace/index.qtpl:24
	qw422016.N().S(` `)
//line template/namespace/index.qtpl:25
}

//line template/namespace/index.qtpl:25
func (p *IndexPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/namespace/index.qtpl:25
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/index.qtpl:25
	p.StreamBody(qw422016)
	//line template/namespace/index.qtpl:25
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/index.qtpl:25
}

//line template/namespace/index.qtpl:25
func (p *IndexPage) Body() string {
	//line template/namespace/index.qtpl:25
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/index.qtpl:25
	p.WriteBody(qb422016)
	//line template/namespace/index.qtpl:25
	qs422016 := string(qb422016.B)
	//line template/namespace/index.qtpl:25
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/index.qtpl:25
	return qs422016
//line template/namespace/index.qtpl:25
}

//line template/namespace/index.qtpl:27
func (p *IndexPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/namespace/index.qtpl:27
	qw422016.N().S(` Namespaces `)
//line template/namespace/index.qtpl:29
}

//line template/namespace/index.qtpl:29
func (p *IndexPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/namespace/index.qtpl:29
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/index.qtpl:29
	p.StreamHeader(qw422016)
	//line template/namespace/index.qtpl:29
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/index.qtpl:29
}

//line template/namespace/index.qtpl:29
func (p *IndexPage) Header() string {
	//line template/namespace/index.qtpl:29
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/index.qtpl:29
	p.WriteHeader(qb422016)
	//line template/namespace/index.qtpl:29
	qs422016 := string(qb422016.B)
	//line template/namespace/index.qtpl:29
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/index.qtpl:29
	return qs422016
//line template/namespace/index.qtpl:29
}

//line template/namespace/index.qtpl:31
func (p *IndexPage) StreamActions(qw422016 *qt422016.Writer) {
	//line template/namespace/index.qtpl:31
	qw422016.N().S(` <li><a href="/namespaces/create" class="btn btn-primary">Create</a></li> `)
//line template/namespace/index.qtpl:33
}

//line template/namespace/index.qtpl:33
func (p *IndexPage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/namespace/index.qtpl:33
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/index.qtpl:33
	p.StreamActions(qw422016)
	//line template/namespace/index.qtpl:33
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/index.qtpl:33
}

//line template/namespace/index.qtpl:33
func (p *IndexPage) Actions() string {
	//line template/namespace/index.qtpl:33
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/index.qtpl:33
	p.WriteActions(qb422016)
	//line template/namespace/index.qtpl:33
	qs422016 := string(qb422016.B)
	//line template/namespace/index.qtpl:33
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/index.qtpl:33
	return qs422016
//line template/namespace/index.qtpl:33
}

//line template/namespace/index.qtpl:35
func (p *IndexPage) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/namespace/index.qtpl:35
}

//line template/namespace/index.qtpl:35
func (p *IndexPage) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/namespace/index.qtpl:35
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/index.qtpl:35
	p.StreamNavigation(qw422016)
	//line template/namespace/index.qtpl:35
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/index.qtpl:35
}

//line template/namespace/index.qtpl:35
func (p *IndexPage) Navigation() string {
	//line template/namespace/index.qtpl:35
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/index.qtpl:35
	p.WriteNavigation(qb422016)
	//line template/namespace/index.qtpl:35
	qs422016 := string(qb422016.B)
	//line template/namespace/index.qtpl:35
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/index.qtpl:35
	return qs422016
//line template/namespace/index.qtpl:35
}
