// This file is automatically generated by qtc from "show.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/build/show.qtpl:2
package build

//line template/build/show.qtpl:2
import (
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

//line template/build/show.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/build/show.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/build/show.qtpl:9
type ShowPage struct {
	template.Page

	Build *model.Build

	ShowManifest  bool
	ShowObjects   bool
	ShowArtifacts bool
	ShowVariables bool
	ShowOutput    bool
}

//line template/build/show.qtpl:23
func (p *ShowPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:23
	qw422016.N().S(` Build #`)
	//line template/build/show.qtpl:24
	qw422016.E().V(p.Build.ID)
	//line template/build/show.qtpl:24
	qw422016.N().S(` - Thrall `)
//line template/build/show.qtpl:25
}

//line template/build/show.qtpl:25
func (p *ShowPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:25
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:25
	p.StreamTitle(qw422016)
	//line template/build/show.qtpl:25
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:25
}

//line template/build/show.qtpl:25
func (p *ShowPage) Title() string {
	//line template/build/show.qtpl:25
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:25
	p.WriteTitle(qb422016)
	//line template/build/show.qtpl:25
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:25
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:25
	return qs422016
//line template/build/show.qtpl:25
}

//line template/build/show.qtpl:27
func (p *ShowPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:27
	qw422016.N().S(` `)
	//line template/build/show.qtpl:28
	if p.ShowManifest {
		//line template/build/show.qtpl:28
		qw422016.N().S(` `)
		//line template/build/show.qtpl:29
		p.streamrenderManifest(qw422016)
		//line template/build/show.qtpl:29
		qw422016.N().S(` `)
		//line template/build/show.qtpl:30
	} else if p.ShowObjects {
		//line template/build/show.qtpl:30
		qw422016.N().S(` `)
		//line template/build/show.qtpl:31
		p.streamrenderObjects(qw422016)
		//line template/build/show.qtpl:31
		qw422016.N().S(` `)
		//line template/build/show.qtpl:32
	} else if p.ShowArtifacts {
		//line template/build/show.qtpl:32
		qw422016.N().S(` `)
		//line template/build/show.qtpl:33
		p.streamrenderArtifacts(qw422016)
		//line template/build/show.qtpl:33
		qw422016.N().S(` `)
		//line template/build/show.qtpl:34
	} else if p.ShowVariables {
		//line template/build/show.qtpl:34
		qw422016.N().S(` `)
		//line template/build/show.qtpl:35
		p.streamrenderVariables(qw422016)
		//line template/build/show.qtpl:35
		qw422016.N().S(` `)
		//line template/build/show.qtpl:36
	} else if p.ShowOutput {
		//line template/build/show.qtpl:36
		qw422016.N().S(` `)
		//line template/build/show.qtpl:37
		p.streamrenderOutput(qw422016)
		//line template/build/show.qtpl:37
		qw422016.N().S(` `)
		//line template/build/show.qtpl:38
	} else {
		//line template/build/show.qtpl:38
		qw422016.N().S(` <div class="panel"> <table class="table"> <tr> <td>Status:</td> <td class="align-right">`)
		//line template/build/show.qtpl:43
		StreamRenderStatus(qw422016, p.Build.Status)
		//line template/build/show.qtpl:43
		qw422016.N().S(`</td> </tr> <tr> <td>Submitted by:</td> <td class="align-right">`)
		//line template/build/show.qtpl:47
		qw422016.E().S(p.Build.User.Username)
		//line template/build/show.qtpl:47
		qw422016.N().S(` &lt;`)
		//line template/build/show.qtpl:47
		qw422016.E().S(p.Build.User.Email)
		//line template/build/show.qtpl:47
		qw422016.N().S(`&gt;</td> </tr> <tr> <td>Started at:</td> <td class="align-right"> `)
		//line template/build/show.qtpl:52
		if p.Build.StartedAt != nil && p.Build.StartedAt.Valid {
			//line template/build/show.qtpl:52
			qw422016.N().S(` `)
			//line template/build/show.qtpl:53
		} else {
			//line template/build/show.qtpl:53
			qw422016.N().S(` -- `)
			//line template/build/show.qtpl:55
		}
		//line template/build/show.qtpl:55
		qw422016.N().S(` </td> </tr> <tr> <td>Finished at:</td> <td class="align-right"> `)
		//line template/build/show.qtpl:61
		if p.Build.FinishedAt != nil && p.Build.FinishedAt.Valid {
			//line template/build/show.qtpl:61
			qw422016.N().S(` `)
			//line template/build/show.qtpl:62
		} else {
			//line template/build/show.qtpl:62
			qw422016.N().S(` -- `)
			//line template/build/show.qtpl:64
		}
		//line template/build/show.qtpl:64
		qw422016.N().S(` </td> </tr> </table> </div> `)
		//line template/build/show.qtpl:69
		for _, s := range p.Build.Stages {
			//line template/build/show.qtpl:69
			qw422016.N().S(` <div class="panel"> <div class="panel-header"> <h3>`)
			//line template/build/show.qtpl:72
			qw422016.E().S(s.Name)
			//line template/build/show.qtpl:72
			qw422016.N().S(`</h3> </div> <table class="table"> <thead> <tr> <th class="cell-pill">STATUS</th> <th>JOB</th> <th class="cell-date">STARTED</th> <th class="cell-date">FINISHED</th> </tr> </thead> <tbody> `)
			//line template/build/show.qtpl:84
			for _, j := range s.Jobs {
				//line template/build/show.qtpl:84
				qw422016.N().S(` <tr> <td class="cell-pill">`)
				//line template/build/show.qtpl:86
				StreamRenderStatus(qw422016, j.Status)
				//line template/build/show.qtpl:86
				qw422016.N().S(`</td> <td><a href="/builds/`)
				//line template/build/show.qtpl:87
				qw422016.E().V(p.Build.ID)
				//line template/build/show.qtpl:87
				qw422016.N().S(`/jobs/`)
				//line template/build/show.qtpl:87
				qw422016.E().V(j.ID)
				//line template/build/show.qtpl:87
				qw422016.N().S(`">`)
				//line template/build/show.qtpl:87
				qw422016.E().S(j.Name)
				//line template/build/show.qtpl:87
				qw422016.N().S(`</a></td> <td class="cell-date"><span class="muted">--</span></td> <td class="cell-date"><span class="muted">--</span></td> </tr> `)
				//line template/build/show.qtpl:91
			}
			//line template/build/show.qtpl:91
			qw422016.N().S(` </tbody> </table> </div> `)
			//line template/build/show.qtpl:95
		}
		//line template/build/show.qtpl:95
		qw422016.N().S(` `)
		//line template/build/show.qtpl:96
	}
	//line template/build/show.qtpl:96
	qw422016.N().S(` `)
//line template/build/show.qtpl:97
}

//line template/build/show.qtpl:97
func (p *ShowPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:97
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:97
	p.StreamBody(qw422016)
	//line template/build/show.qtpl:97
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:97
}

//line template/build/show.qtpl:97
func (p *ShowPage) Body() string {
	//line template/build/show.qtpl:97
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:97
	p.WriteBody(qb422016)
	//line template/build/show.qtpl:97
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:97
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:97
	return qs422016
//line template/build/show.qtpl:97
}

//line template/build/show.qtpl:99
func (p *ShowPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:99
	qw422016.N().S(` <a href="/" class="back">`)
	//line template/build/show.qtpl:100
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
	//line template/build/show.qtpl:100
	qw422016.N().S(`</a> Build #`)
	//line template/build/show.qtpl:100
	qw422016.E().V(p.Build.ID)
	//line template/build/show.qtpl:100
	qw422016.N().S(` `)
//line template/build/show.qtpl:101
}

//line template/build/show.qtpl:101
func (p *ShowPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:101
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:101
	p.StreamHeader(qw422016)
	//line template/build/show.qtpl:101
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:101
}

//line template/build/show.qtpl:101
func (p *ShowPage) Header() string {
	//line template/build/show.qtpl:101
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:101
	p.WriteHeader(qb422016)
	//line template/build/show.qtpl:101
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:101
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:101
	return qs422016
//line template/build/show.qtpl:101
}

//line template/build/show.qtpl:103
func (p *ShowPage) StreamActions(qw422016 *qt422016.Writer) {
//line template/build/show.qtpl:103
}

//line template/build/show.qtpl:103
func (p *ShowPage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:103
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:103
	p.StreamActions(qw422016)
	//line template/build/show.qtpl:103
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:103
}

//line template/build/show.qtpl:103
func (p *ShowPage) Actions() string {
	//line template/build/show.qtpl:103
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:103
	p.WriteActions(qb422016)
	//line template/build/show.qtpl:103
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:103
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:103
	return qs422016
//line template/build/show.qtpl:103
}

//line template/build/show.qtpl:105
func (p *ShowPage) StreamNavigation(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:105
	qw422016.N().S(` `)
	//line template/build/show.qtpl:106
	qw422016.N().S(`<li>`)
	//line template/build/show.qtpl:107
	template.StreamRenderLink(qw422016, p.Build.UIEndpoint(), p.URI)
	//line template/build/show.qtpl:107
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12 9c1.641 0 3 1.359 3 3s-1.359 3-3 3-3-1.359-3-3 1.359-3 3-3zM12 17.016c2.766 0 5.016-2.25 5.016-5.016s-2.25-5.016-5.016-5.016-5.016 2.25-5.016 5.016 2.25 5.016 5.016 5.016zM12 4.5c5.016 0 9.281 3.094 11.016 7.5-1.734 4.406-6 7.5-11.016 7.5s-9.281-3.094-11.016-7.5c1.734-4.406 6-7.5 11.016-7.5z"></path>
</svg>
`)
	//line template/build/show.qtpl:107
	qw422016.N().S(`<span>Overview</span></a></li><li>`)
	//line template/build/show.qtpl:108
	template.StreamRenderLink(qw422016, p.Build.UIEndpoint()+"/manifest", p.URI)
	//line template/build/show.qtpl:108
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M6.984 12.984v-1.969h14.016v1.969h-14.016zM6.984 18.984v-1.969h14.016v1.969h-14.016zM6.984 5.016h14.016v1.969h-14.016v-1.969zM2.016 11.016v-1.031h3v0.938l-1.828 2.063h1.828v1.031h-3v-0.938l1.781-2.063h-1.781zM3 8.016v-3h-0.984v-1.031h1.969v4.031h-0.984zM2.016 17.016v-1.031h3v4.031h-3v-1.031h1.969v-0.469h-0.984v-1.031h0.984v-0.469h-1.969z"></path>
</svg>
`)
	//line template/build/show.qtpl:108
	qw422016.N().S(`<span>Manifest</span></a></li><li>`)
	//line template/build/show.qtpl:109
	template.StreamRenderLink(qw422016, p.Build.UIEndpoint()+"/objects", p.URI)
	//line template/build/show.qtpl:109
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M5.016 18h13.969v2.016h-13.969v-2.016zM9 15.984v-6h-3.984l6.984-6.984 6.984 6.984h-3.984v6h-6z"></path>
</svg>
`)
	//line template/build/show.qtpl:109
	qw422016.N().S(`<span>Objects</span></a></li><li>`)
	//line template/build/show.qtpl:110
	template.StreamRenderLink(qw422016, p.Build.UIEndpoint()+"/artifacts", p.URI)
	//line template/build/show.qtpl:110
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M5.016 18h13.969v2.016h-13.969v-2.016zM18.984 9l-6.984 6.984-6.984-6.984h3.984v-6h6v6h3.984z"></path>
</svg>
`)
	//line template/build/show.qtpl:110
	qw422016.N().S(`<span>Artifacts</span></a></li><li>`)
	//line template/build/show.qtpl:111
	template.StreamRenderLink(qw422016, p.Build.UIEndpoint()+"/variables", p.URI)
	//line template/build/show.qtpl:111
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M14.578 16.594l4.641-4.594-4.641-4.594 1.406-1.406 6 6-6 6zM9.422 16.594l-1.406 1.406-6-6 6-6 1.406 1.406-4.641 4.594z"></path>
</svg>
`)
	//line template/build/show.qtpl:111
	qw422016.N().S(`<span>Variables</span></a></li><li>`)
	//line template/build/show.qtpl:112
	template.StreamRenderLink(qw422016, p.Build.UIEndpoint()+"/output", p.URI)
	//line template/build/show.qtpl:112
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12.984 9h5.531l-5.531-5.484v5.484zM15.984 14.016v-2.016h-7.969v2.016h7.969zM15.984 18v-2.016h-7.969v2.016h7.969zM14.016 2.016l6 6v12c0 1.078-0.938 1.969-2.016 1.969h-12c-1.078 0-2.016-0.891-2.016-1.969l0.047-16.031c0-1.078 0.891-1.969 1.969-1.969h8.016z"></path>
</svg>
`)
	//line template/build/show.qtpl:112
	qw422016.N().S(`<span>Output</span></a></li>`)
	//line template/build/show.qtpl:113
	qw422016.N().S(` `)
//line template/build/show.qtpl:114
}

//line template/build/show.qtpl:114
func (p *ShowPage) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:114
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:114
	p.StreamNavigation(qw422016)
	//line template/build/show.qtpl:114
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:114
}

//line template/build/show.qtpl:114
func (p *ShowPage) Navigation() string {
	//line template/build/show.qtpl:114
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:114
	p.WriteNavigation(qb422016)
	//line template/build/show.qtpl:114
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:114
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:114
	return qs422016
//line template/build/show.qtpl:114
}

//line template/build/show.qtpl:116
func (p *ShowPage) streamrenderManifest(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:116
	qw422016.N().S(` <div class="panel"> <div class="panel-header"> <ul class="panel-actions"> <li><a class="btn btn-primary" href="/builds/`)
	//line template/build/show.qtpl:120
	qw422016.E().V(p.Build.ID)
	//line template/build/show.qtpl:120
	qw422016.N().S(`/manifest/raw">`)
	//line template/build/show.qtpl:120
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12.984 9h5.531l-5.531-5.484v5.484zM15.984 14.016v-2.016h-7.969v2.016h7.969zM15.984 18v-2.016h-7.969v2.016h7.969zM14.016 2.016l6 6v12c0 1.078-0.938 1.969-2.016 1.969h-12c-1.078 0-2.016-0.891-2.016-1.969l0.047-16.031c0-1.078 0.891-1.969 1.969-1.969h8.016z"></path>
</svg>
`)
	//line template/build/show.qtpl:120
	qw422016.N().S(`<span>Raw</span></a></li> </ul> </div> <pre class="code">`)
	//line template/build/show.qtpl:123
	template.StreamRenderCode(qw422016, p.Build.Manifest)
	//line template/build/show.qtpl:123
	qw422016.N().S(`</pre> </div> `)
//line template/build/show.qtpl:125
}

//line template/build/show.qtpl:125
func (p *ShowPage) writerenderManifest(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:125
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:125
	p.streamrenderManifest(qw422016)
	//line template/build/show.qtpl:125
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:125
}

//line template/build/show.qtpl:125
func (p *ShowPage) renderManifest() string {
	//line template/build/show.qtpl:125
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:125
	p.writerenderManifest(qb422016)
	//line template/build/show.qtpl:125
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:125
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:125
	return qs422016
//line template/build/show.qtpl:125
}

//line template/build/show.qtpl:127
func (p *ShowPage) streamrenderObjects(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:127
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/show.qtpl:129
	if len(p.Build.Objects) > 0 {
		//line template/build/show.qtpl:129
		qw422016.N().S(` <table class="table"> <thead> <tr> <th>SOURCE</th> <th>NAME</th> <th>PLACED</th> </tr> </thead> <tbody> `)
		//line template/build/show.qtpl:139
		for _, o := range p.Build.Objects {
			//line template/build/show.qtpl:139
			qw422016.N().S(` <tr> <td>`)
			//line template/build/show.qtpl:141
			qw422016.E().S(o.Source)
			//line template/build/show.qtpl:141
			qw422016.N().S(`</td> <td><code>`)
			//line template/build/show.qtpl:142
			qw422016.E().S(o.Name)
			//line template/build/show.qtpl:142
			qw422016.N().S(`</code></td> <td>`)
			//line template/build/show.qtpl:143
			if o.Placed {
				//line template/build/show.qtpl:143
				qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M9 16.172l10.594-10.594 1.406 1.406-12 12-5.578-5.578 1.406-1.406z"></path>
</svg>
`)
				//line template/build/show.qtpl:143
			} else {
				//line template/build/show.qtpl:143
				qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M18.984 6.422l-5.578 5.578 5.578 5.578-1.406 1.406-5.578-5.578-5.578 5.578-1.406-1.406 5.578-5.578-5.578-5.578 1.406-1.406 5.578 5.578 5.578-5.578z"></path>
</svg>
`)
				//line template/build/show.qtpl:143
			}
			//line template/build/show.qtpl:143
			qw422016.N().S(`</td> </tr> `)
			//line template/build/show.qtpl:145
		}
		//line template/build/show.qtpl:145
		qw422016.N().S(` </tbody> </table> `)
		//line template/build/show.qtpl:148
	} else {
		//line template/build/show.qtpl:148
		qw422016.N().S(` <div class="panel-message muted">No objects have been placed for this build.</div> `)
		//line template/build/show.qtpl:150
	}
	//line template/build/show.qtpl:150
	qw422016.N().S(` </div> `)
//line template/build/show.qtpl:152
}

//line template/build/show.qtpl:152
func (p *ShowPage) writerenderObjects(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:152
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:152
	p.streamrenderObjects(qw422016)
	//line template/build/show.qtpl:152
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:152
}

//line template/build/show.qtpl:152
func (p *ShowPage) renderObjects() string {
	//line template/build/show.qtpl:152
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:152
	p.writerenderObjects(qb422016)
	//line template/build/show.qtpl:152
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:152
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:152
	return qs422016
//line template/build/show.qtpl:152
}

//line template/build/show.qtpl:154
func (p *ShowPage) streamrenderArtifacts(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:154
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/show.qtpl:156
	if len(p.Build.Artifacts) > 0 {
		//line template/build/show.qtpl:156
		qw422016.N().S(` `)
		//line template/build/show.qtpl:157
	} else {
		//line template/build/show.qtpl:157
		qw422016.N().S(` <div class="panel-message muted">No artifacts have been collected from this build.</div> `)
		//line template/build/show.qtpl:159
	}
	//line template/build/show.qtpl:159
	qw422016.N().S(` </div> `)
//line template/build/show.qtpl:161
}

//line template/build/show.qtpl:161
func (p *ShowPage) writerenderArtifacts(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:161
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:161
	p.streamrenderArtifacts(qw422016)
	//line template/build/show.qtpl:161
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:161
}

//line template/build/show.qtpl:161
func (p *ShowPage) renderArtifacts() string {
	//line template/build/show.qtpl:161
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:161
	p.writerenderArtifacts(qb422016)
	//line template/build/show.qtpl:161
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:161
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:161
	return qs422016
//line template/build/show.qtpl:161
}

//line template/build/show.qtpl:163
func (p *ShowPage) streamrenderVariables(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:163
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/show.qtpl:165
	if len(p.Build.Variables) > 0 {
		//line template/build/show.qtpl:165
		qw422016.N().S(` `)
		//line template/build/show.qtpl:167
	} else {
		//line template/build/show.qtpl:167
		qw422016.N().S(` <div class="panel-message muted">No variables have been set for this build.</div> `)
		//line template/build/show.qtpl:169
	}
	//line template/build/show.qtpl:169
	qw422016.N().S(` </div> `)
//line template/build/show.qtpl:171
}

//line template/build/show.qtpl:171
func (p *ShowPage) writerenderVariables(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:171
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:171
	p.streamrenderVariables(qw422016)
	//line template/build/show.qtpl:171
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:171
}

//line template/build/show.qtpl:171
func (p *ShowPage) renderVariables() string {
	//line template/build/show.qtpl:171
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:171
	p.writerenderVariables(qb422016)
	//line template/build/show.qtpl:171
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:171
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:171
	return qs422016
//line template/build/show.qtpl:171
}

//line template/build/show.qtpl:173
func (p *ShowPage) streamrenderOutput(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:173
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/show.qtpl:175
	if p.Build.Output.Valid {
		//line template/build/show.qtpl:175
		qw422016.N().S(` <div class="panel-header"> <ul class="panel-actions"> <li><a class="btn btn-primary" href="/builds/`)
		//line template/build/show.qtpl:178
		qw422016.E().V(p.Build.ID)
		//line template/build/show.qtpl:178
		qw422016.N().S(`/output/raw">`)
		//line template/build/show.qtpl:178
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12.984 9h5.531l-5.531-5.484v5.484zM15.984 14.016v-2.016h-7.969v2.016h7.969zM15.984 18v-2.016h-7.969v2.016h7.969zM14.016 2.016l6 6v12c0 1.078-0.938 1.969-2.016 1.969h-12c-1.078 0-2.016-0.891-2.016-1.969l0.047-16.031c0-1.078 0.891-1.969 1.969-1.969h8.016z"></path>
</svg>
`)
		//line template/build/show.qtpl:178
		qw422016.N().S(`<span>Raw</span></a></li> </ul> </div> <pre class="code">`)
		//line template/build/show.qtpl:181
		template.StreamRenderCode(qw422016, p.Build.Output.String)
		//line template/build/show.qtpl:181
		qw422016.N().S(`</pre> `)
		//line template/build/show.qtpl:182
	} else {
		//line template/build/show.qtpl:182
		qw422016.N().S(` <div class="panel-message muted">No build output has been produced.</div> `)
		//line template/build/show.qtpl:184
	}
	//line template/build/show.qtpl:184
	qw422016.N().S(` </div> `)
//line template/build/show.qtpl:186
}

//line template/build/show.qtpl:186
func (p *ShowPage) writerenderOutput(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:186
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:186
	p.streamrenderOutput(qw422016)
	//line template/build/show.qtpl:186
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:186
}

//line template/build/show.qtpl:186
func (p *ShowPage) renderOutput() string {
	//line template/build/show.qtpl:186
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:186
	p.writerenderOutput(qb422016)
	//line template/build/show.qtpl:186
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:186
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:186
	return qs422016
//line template/build/show.qtpl:186
}
