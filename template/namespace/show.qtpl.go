// This file is automatically generated by qtc from "show.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/namespace/show.qtpl:2
package namespace

//line template/namespace/show.qtpl:2
import (
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

//line template/namespace/show.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/namespace/show.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/namespace/show.qtpl:9
type ShowPage struct {
	*template.Page

	User      *model.User
	Namespace *model.Namespace
}

//line template/namespace/show.qtpl:17
func (p *ShowPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:17
	qw422016.N().S(`
`)
	//line template/namespace/show.qtpl:18
	qw422016.E().S(p.Namespace.Name)
	//line template/namespace/show.qtpl:18
	qw422016.N().S(`
`)
//line template/namespace/show.qtpl:19
}

//line template/namespace/show.qtpl:19
func (p *ShowPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:19
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:19
	p.StreamTitle(qw422016)
	//line template/namespace/show.qtpl:19
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:19
}

//line template/namespace/show.qtpl:19
func (p *ShowPage) Title() string {
	//line template/namespace/show.qtpl:19
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:19
	p.WriteTitle(qb422016)
	//line template/namespace/show.qtpl:19
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:19
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:19
	return qs422016
//line template/namespace/show.qtpl:19
}

//line template/namespace/show.qtpl:21
func (p *ShowPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/namespace/show.qtpl:21
	qw422016.N().S(`
<div class="dashboard-header">
	<h1>`)
	//line template/namespace/show.qtpl:23
	qw422016.E().S(p.Namespace.Name)
	//line template/namespace/show.qtpl:23
	qw422016.N().S(`</h1>
	<ul class="actions">
		<li><a href="/u/`)
	//line template/namespace/show.qtpl:25
	qw422016.E().S(p.User.Username)
	//line template/namespace/show.qtpl:25
	qw422016.N().S(`/`)
	//line template/namespace/show.qtpl:25
	qw422016.E().S(p.Namespace.Name)
	//line template/namespace/show.qtpl:25
	qw422016.N().S(`/-/edit" class="button button-secondary">Edit</a></li>
		<li><a href="/namespaces/create?parent=`)
	//line template/namespace/show.qtpl:26
	qw422016.E().S(p.Namespace.Name)
	//line template/namespace/show.qtpl:26
	qw422016.N().S(`" class="button button-primary">Create</a></li>
	</ul>
</div>
`)
//line template/namespace/show.qtpl:29
}

//line template/namespace/show.qtpl:29
func (p *ShowPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/namespace/show.qtpl:29
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/show.qtpl:29
	p.StreamBody(qw422016)
	//line template/namespace/show.qtpl:29
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/show.qtpl:29
}

//line template/namespace/show.qtpl:29
func (p *ShowPage) Body() string {
	//line template/namespace/show.qtpl:29
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/show.qtpl:29
	p.WriteBody(qb422016)
	//line template/namespace/show.qtpl:29
	qs422016 := string(qb422016.B)
	//line template/namespace/show.qtpl:29
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/show.qtpl:29
	return qs422016
//line template/namespace/show.qtpl:29
}
