// This file is automatically generated by qtc from "render.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/build/render.qtpl:1
package build

//line template/build/render.qtpl:1
import "github.com/andrewpillar/thrall/model"

//line template/build/render.qtpl:4
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/build/render.qtpl:4
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/build/render.qtpl:4
func StreamRenderBuilds(qw422016 *qt422016.Writer, builds []*model.Build) {
	//line template/build/render.qtpl:4
	qw422016.N().S(` `)
	//line template/build/render.qtpl:5
	if len(builds) == 0 {
		//line template/build/render.qtpl:5
		qw422016.N().S(` <div class="message">No builds have been submitted yet.</div> `)
		//line template/build/render.qtpl:7
	} else {
		//line template/build/render.qtpl:7
		qw422016.N().S(` <table class="table"> <tr> <th>Status</th> <th>Build</th> <th>Namespace</th> <th></th> <th></th> </tr> `)
		//line template/build/render.qtpl:16
		for _, b := range builds {
			//line template/build/render.qtpl:16
			qw422016.N().S(` <tr> `)
			//line template/build/render.qtpl:18
			switch b.Status {
			//line template/build/render.qtpl:19
			case model.Queued:
				//line template/build/render.qtpl:19
				qw422016.N().S(` <td><span class="build-status build-queued">queued</span></td> `)
			//line template/build/render.qtpl:21
			case model.Running:
				//line template/build/render.qtpl:21
				qw422016.N().S(` <td><span class="build-status build-running">running</span></td> `)
			//line template/build/render.qtpl:23
			case model.Passed:
				//line template/build/render.qtpl:23
				qw422016.N().S(` <td><span class="build-status build-passed">passed</span></td> `)
			//line template/build/render.qtpl:25
			case model.Failed:
				//line template/build/render.qtpl:25
				qw422016.N().S(` <td><span class="build-status build-failed">failed</span></td> `)
				//line template/build/render.qtpl:27
			}
			//line template/build/render.qtpl:27
			qw422016.N().S(` <td><a href="`)
			//line template/build/render.qtpl:28
			qw422016.E().S(b.URI())
			//line template/build/render.qtpl:28
			qw422016.N().S(`">#`)
			//line template/build/render.qtpl:28
			qw422016.E().V(b.ID)
			//line template/build/render.qtpl:28
			qw422016.N().S(`</a></td> <td> `)
			//line template/build/render.qtpl:30
			if b.Namespace != nil {
				//line template/build/render.qtpl:30
				qw422016.N().S(` <a href="`)
				//line template/build/render.qtpl:31
				qw422016.E().S(b.Namespace.URI())
				//line template/build/render.qtpl:31
				qw422016.N().S(`">`)
				//line template/build/render.qtpl:31
				qw422016.E().S(b.Namespace.FullName)
				//line template/build/render.qtpl:31
				qw422016.N().S(`</a> `)
				//line template/build/render.qtpl:32
			} else {
				//line template/build/render.qtpl:32
				qw422016.N().S(` -- `)
				//line template/build/render.qtpl:34
			}
			//line template/build/render.qtpl:34
			qw422016.N().S(` </td> <td class="align-right"> `)
			//line template/build/render.qtpl:37
			for _, t := range b.Tags {
				//line template/build/render.qtpl:37
				qw422016.N().S(` <a class="tag" href="?tag=`)
				//line template/build/render.qtpl:38
				qw422016.E().S(t.Name)
				//line template/build/render.qtpl:38
				qw422016.N().S(`">`)
				//line template/build/render.qtpl:38
				qw422016.E().S(t.Name)
				//line template/build/render.qtpl:38
				qw422016.N().S(`</a> `)
				//line template/build/render.qtpl:39
			}
			//line template/build/render.qtpl:39
			qw422016.N().S(` </td> <td class="align-right"> `)
			//line template/build/render.qtpl:42
			if b.FinishedAt != nil {
				//line template/build/render.qtpl:42
				qw422016.N().S(` `)
				//line template/build/render.qtpl:43
			} else {
				//line template/build/render.qtpl:43
				qw422016.N().S(` -- `)
				//line template/build/render.qtpl:45
			}
			//line template/build/render.qtpl:45
			qw422016.N().S(` </td> </tr> `)
			//line template/build/render.qtpl:48
		}
		//line template/build/render.qtpl:48
		qw422016.N().S(` </table> `)
		//line template/build/render.qtpl:50
	}
	//line template/build/render.qtpl:50
	qw422016.N().S(` `)
//line template/build/render.qtpl:51
}

//line template/build/render.qtpl:51
func WriteRenderBuilds(qq422016 qtio422016.Writer, builds []*model.Build) {
	//line template/build/render.qtpl:51
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/render.qtpl:51
	StreamRenderBuilds(qw422016, builds)
	//line template/build/render.qtpl:51
	qt422016.ReleaseWriter(qw422016)
//line template/build/render.qtpl:51
}

//line template/build/render.qtpl:51
func RenderBuilds(builds []*model.Build) string {
	//line template/build/render.qtpl:51
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/render.qtpl:51
	WriteRenderBuilds(qb422016, builds)
	//line template/build/render.qtpl:51
	qs422016 := string(qb422016.B)
	//line template/build/render.qtpl:51
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/render.qtpl:51
	return qs422016
//line template/build/render.qtpl:51
}
