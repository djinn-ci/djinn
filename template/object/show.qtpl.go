// This file is automatically generated by qtc from "show.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/object/show.qtpl:2
package object

//line template/object/show.qtpl:2
import (
	"fmt"

	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/build"
)

//line template/object/show.qtpl:11
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/object/show.qtpl:11
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/object/show.qtpl:12
type ShowPage struct {
	template.BasePage

	Object *model.Object
	Search string
	Status string
	Builds []*model.Build
}

//line template/object/show.qtpl:23
func (p *ShowPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/object/show.qtpl:23
	qw422016.N().S(` Objects `)
	//line template/object/show.qtpl:24
	qw422016.E().S(p.Object.Name)
	//line template/object/show.qtpl:24
	qw422016.N().S(` - Thrall `)
//line template/object/show.qtpl:25
}

//line template/object/show.qtpl:25
func (p *ShowPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/object/show.qtpl:25
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/object/show.qtpl:25
	p.StreamTitle(qw422016)
	//line template/object/show.qtpl:25
	qt422016.ReleaseWriter(qw422016)
//line template/object/show.qtpl:25
}

//line template/object/show.qtpl:25
func (p *ShowPage) Title() string {
	//line template/object/show.qtpl:25
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/object/show.qtpl:25
	p.WriteTitle(qb422016)
	//line template/object/show.qtpl:25
	qs422016 := string(qb422016.B)
	//line template/object/show.qtpl:25
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/object/show.qtpl:25
	return qs422016
//line template/object/show.qtpl:25
}

//line template/object/show.qtpl:27
func (p *ShowPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/object/show.qtpl:27
	qw422016.N().S(` <div class="panel"> <table class="table"> <tr> <td>Name</td> <td class="align-right">`)
	//line template/object/show.qtpl:32
	qw422016.E().S(p.Object.Name)
	//line template/object/show.qtpl:32
	qw422016.N().S(`</td> </tr> <tr> <td>Type</td> <td class="align-right"><span class="code">`)
	//line template/object/show.qtpl:36
	qw422016.E().S(p.Object.Type)
	//line template/object/show.qtpl:36
	qw422016.N().S(`</span></td> </tr> <tr> <td>Size</td> <td class="align-right">`)
	//line template/object/show.qtpl:40
	qw422016.E().S(template.RenderSize(p.Object.Size))
	//line template/object/show.qtpl:40
	qw422016.N().S(`</td> </tr> <tr> <td>MD5</td> <td class="align-right"><span class="code">`)
	//line template/object/show.qtpl:44
	qw422016.E().S(fmt.Sprintf("%x", p.Object.MD5))
	//line template/object/show.qtpl:44
	qw422016.N().S(`</span></td> </tr> <tr> <td>SHA256</td> <td class="align-right"><span class="code">`)
	//line template/object/show.qtpl:48
	qw422016.E().S(fmt.Sprintf("%x", p.Object.SHA256))
	//line template/object/show.qtpl:48
	qw422016.N().S(`</span></td> </tr> </table> </div> <div class="panel">`)
	//line template/object/show.qtpl:52
	build.StreamRenderIndex(qw422016, p.Builds, p.URL.Path, p.Status, p.Search)
	//line template/object/show.qtpl:52
	qw422016.N().S(`</div> `)
//line template/object/show.qtpl:53
}

//line template/object/show.qtpl:53
func (p *ShowPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/object/show.qtpl:53
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/object/show.qtpl:53
	p.StreamBody(qw422016)
	//line template/object/show.qtpl:53
	qt422016.ReleaseWriter(qw422016)
//line template/object/show.qtpl:53
}

//line template/object/show.qtpl:53
func (p *ShowPage) Body() string {
	//line template/object/show.qtpl:53
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/object/show.qtpl:53
	p.WriteBody(qb422016)
	//line template/object/show.qtpl:53
	qs422016 := string(qb422016.B)
	//line template/object/show.qtpl:53
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/object/show.qtpl:53
	return qs422016
//line template/object/show.qtpl:53
}

//line template/object/show.qtpl:55
func (p *ShowPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/object/show.qtpl:55
	qw422016.N().S(` <a href="/objects" class="back">`)
	//line template/object/show.qtpl:56
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
	//line template/object/show.qtpl:56
	qw422016.N().S(`</a> `)
	//line template/object/show.qtpl:57
	if !p.Object.Namespace.IsZero() {
		//line template/object/show.qtpl:57
		qw422016.N().S(` <a href="`)
		//line template/object/show.qtpl:58
		qw422016.E().S(p.Object.Namespace.UIEndpoint())
		//line template/object/show.qtpl:58
		qw422016.N().S(`">`)
		//line template/object/show.qtpl:58
		qw422016.E().S(p.Object.Namespace.Name)
		//line template/object/show.qtpl:58
		qw422016.N().S(`</a> / `)
		//line template/object/show.qtpl:59
	}
	//line template/object/show.qtpl:59
	qw422016.N().S(` `)
	//line template/object/show.qtpl:60
	qw422016.E().S(p.Object.Name)
	//line template/object/show.qtpl:60
	qw422016.N().S(` `)
//line template/object/show.qtpl:61
}

//line template/object/show.qtpl:61
func (p *ShowPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/object/show.qtpl:61
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/object/show.qtpl:61
	p.StreamHeader(qw422016)
	//line template/object/show.qtpl:61
	qt422016.ReleaseWriter(qw422016)
//line template/object/show.qtpl:61
}

//line template/object/show.qtpl:61
func (p *ShowPage) Header() string {
	//line template/object/show.qtpl:61
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/object/show.qtpl:61
	p.WriteHeader(qb422016)
	//line template/object/show.qtpl:61
	qs422016 := string(qb422016.B)
	//line template/object/show.qtpl:61
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/object/show.qtpl:61
	return qs422016
//line template/object/show.qtpl:61
}

//line template/object/show.qtpl:63
func (p *ShowPage) StreamActions(qw422016 *qt422016.Writer) {
	//line template/object/show.qtpl:63
	qw422016.N().S(` <li><a href="`)
	//line template/object/show.qtpl:64
	qw422016.E().S(p.Object.UIEndpoint("download", p.Object.Name))
	//line template/object/show.qtpl:64
	qw422016.N().S(`" class="btn btn-primary">Download</a></li> `)
//line template/object/show.qtpl:65
}

//line template/object/show.qtpl:65
func (p *ShowPage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/object/show.qtpl:65
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/object/show.qtpl:65
	p.StreamActions(qw422016)
	//line template/object/show.qtpl:65
	qt422016.ReleaseWriter(qw422016)
//line template/object/show.qtpl:65
}

//line template/object/show.qtpl:65
func (p *ShowPage) Actions() string {
	//line template/object/show.qtpl:65
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/object/show.qtpl:65
	p.WriteActions(qb422016)
	//line template/object/show.qtpl:65
	qs422016 := string(qb422016.B)
	//line template/object/show.qtpl:65
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/object/show.qtpl:65
	return qs422016
//line template/object/show.qtpl:65
}

//line template/object/show.qtpl:67
func (p *ShowPage) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/object/show.qtpl:67
}

//line template/object/show.qtpl:67
func (p *ShowPage) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/object/show.qtpl:67
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/object/show.qtpl:67
	p.StreamNavigation(qw422016)
	//line template/object/show.qtpl:67
	qt422016.ReleaseWriter(qw422016)
//line template/object/show.qtpl:67
}

//line template/object/show.qtpl:67
func (p *ShowPage) Navigation() string {
	//line template/object/show.qtpl:67
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/object/show.qtpl:67
	p.WriteNavigation(qb422016)
	//line template/object/show.qtpl:67
	qs422016 := string(qb422016.B)
	//line template/object/show.qtpl:67
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/object/show.qtpl:67
	return qs422016
//line template/object/show.qtpl:67
}
