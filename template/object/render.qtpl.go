// This file is automatically generated by qtc from "render.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/object/render.qtpl:2
package object

//line template/object/render.qtpl:2
import (
	"fmt"

	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

//line template/object/render.qtpl:10
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/object/render.qtpl:10
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/object/render.qtpl:10
func StreamRenderTable(qw422016 *qt422016.Writer, oo []*model.Object, csrf string) {
	//line template/object/render.qtpl:10
	qw422016.N().S(`
	<table class="table">
		<thead>
			<tr>
				<th>NAME</th>
				<th>TYPE</th>
				<th>SIZE</th>
				<th>MD5</th>
				<th>SHA256</th>
				<th>NAMESPACE</th>
				<th></th>
			</tr>
		</thead>
		<tbody>
			`)
	//line template/object/render.qtpl:24
	for _, o := range oo {
		//line template/object/render.qtpl:24
		qw422016.N().S(`
				<tr>
					<td><a href="`)
		//line template/object/render.qtpl:26
		qw422016.E().S(o.UIEndpoint())
		//line template/object/render.qtpl:26
		qw422016.N().S(`">`)
		//line template/object/render.qtpl:26
		qw422016.E().S(o.Name)
		//line template/object/render.qtpl:26
		qw422016.N().S(`</a></td>
					<td><span class="code">`)
		//line template/object/render.qtpl:27
		qw422016.E().S(o.Type)
		//line template/object/render.qtpl:27
		qw422016.N().S(`</span></td>
					<td>`)
		//line template/object/render.qtpl:28
		qw422016.E().S(template.RenderSize(o.Size))
		//line template/object/render.qtpl:28
		qw422016.N().S(`</td>
					<td><span class="code">`)
		//line template/object/render.qtpl:29
		qw422016.E().S(fmt.Sprintf("%x", o.MD5))
		//line template/object/render.qtpl:29
		qw422016.N().S(`</span></td>
					<td><span class="code">`)
		//line template/object/render.qtpl:30
		qw422016.E().S(fmt.Sprintf("%x", o.SHA256))
		//line template/object/render.qtpl:30
		qw422016.N().S(`</span></td>
					<td>
						`)
		//line template/object/render.qtpl:32
		if o.Namespace != nil {
			//line template/object/render.qtpl:32
			qw422016.N().S(`
							<a href="`)
			//line template/object/render.qtpl:33
			qw422016.E().S(o.Namespace.UIEndpoint())
			//line template/object/render.qtpl:33
			qw422016.N().S(`">`)
			//line template/object/render.qtpl:33
			qw422016.E().S(o.Namespace.Path)
			//line template/object/render.qtpl:33
			qw422016.N().S(`</a>
						`)
			//line template/object/render.qtpl:34
		} else {
			//line template/object/render.qtpl:34
			qw422016.N().S(`
							<span class="muted">--</span>
						`)
			//line template/object/render.qtpl:36
		}
		//line template/object/render.qtpl:36
		qw422016.N().S(`
					</td>
					<td class="align-right">
						<form method="POST" action="`)
		//line template/object/render.qtpl:39
		qw422016.E().S(o.UIEndpoint())
		//line template/object/render.qtpl:39
		qw422016.N().S(`">
							`)
		//line template/object/render.qtpl:40
		qw422016.N().S(string(csrf))
		//line template/object/render.qtpl:40
		qw422016.N().S(`
							<input type="hidden" name="_method" value="DELETE"/>
							<button type="submit" class="btn btn-danger">Delete</button>
						</form>
					</td>
				</tr>
			`)
		//line template/object/render.qtpl:46
	}
	//line template/object/render.qtpl:46
	qw422016.N().S(`
		</tbody>
	</table>
`)
//line template/object/render.qtpl:49
}

//line template/object/render.qtpl:49
func WriteRenderTable(qq422016 qtio422016.Writer, oo []*model.Object, csrf string) {
	//line template/object/render.qtpl:49
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/object/render.qtpl:49
	StreamRenderTable(qw422016, oo, csrf)
	//line template/object/render.qtpl:49
	qt422016.ReleaseWriter(qw422016)
//line template/object/render.qtpl:49
}

//line template/object/render.qtpl:49
func RenderTable(oo []*model.Object, csrf string) string {
	//line template/object/render.qtpl:49
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/object/render.qtpl:49
	WriteRenderTable(qb422016, oo, csrf)
	//line template/object/render.qtpl:49
	qs422016 := string(qb422016.B)
	//line template/object/render.qtpl:49
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/object/render.qtpl:49
	return qs422016
//line template/object/render.qtpl:49
}

//line template/object/render.qtpl:51
func StreamRenderIndex(qw422016 *qt422016.Writer, oo []*model.Object, uri, search, csrf string) {
	//line template/object/render.qtpl:51
	qw422016.N().S(`
	`)
	//line template/object/render.qtpl:52
	if len(oo) == 0 && search == "" {
		//line template/object/render.qtpl:52
		qw422016.N().S(`
		<div class="panel-message muted">Objects are files that can be used in build environments.</div>
	`)
		//line template/object/render.qtpl:54
	} else {
		//line template/object/render.qtpl:54
		qw422016.N().S(`
		<div class="panel-header">`)
		//line template/object/render.qtpl:55
		template.StreamRenderSearch(qw422016, uri, search, "Find an object...")
		//line template/object/render.qtpl:55
		qw422016.N().S(`</div>
		`)
		//line template/object/render.qtpl:56
		if len(oo) == 0 && search != "" {
			//line template/object/render.qtpl:56
			qw422016.N().S(`
			<div class="panel-message muted">No results found.</div>
		`)
			//line template/object/render.qtpl:58
		} else {
			//line template/object/render.qtpl:58
			qw422016.N().S(`
			`)
			//line template/object/render.qtpl:59
			StreamRenderTable(qw422016, oo, csrf)
			//line template/object/render.qtpl:59
			qw422016.N().S(`
		`)
			//line template/object/render.qtpl:60
		}
		//line template/object/render.qtpl:60
		qw422016.N().S(`
	`)
		//line template/object/render.qtpl:61
	}
	//line template/object/render.qtpl:61
	qw422016.N().S(`
`)
//line template/object/render.qtpl:62
}

//line template/object/render.qtpl:62
func WriteRenderIndex(qq422016 qtio422016.Writer, oo []*model.Object, uri, search, csrf string) {
	//line template/object/render.qtpl:62
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/object/render.qtpl:62
	StreamRenderIndex(qw422016, oo, uri, search, csrf)
	//line template/object/render.qtpl:62
	qt422016.ReleaseWriter(qw422016)
//line template/object/render.qtpl:62
}

//line template/object/render.qtpl:62
func RenderIndex(oo []*model.Object, uri, search, csrf string) string {
	//line template/object/render.qtpl:62
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/object/render.qtpl:62
	WriteRenderIndex(qb422016, oo, uri, search, csrf)
	//line template/object/render.qtpl:62
	qs422016 := string(qb422016.B)
	//line template/object/render.qtpl:62
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/object/render.qtpl:62
	return qs422016
//line template/object/render.qtpl:62
}
