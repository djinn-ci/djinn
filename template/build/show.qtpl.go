// This file is automatically generated by qtc from "show.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/build/show.qtpl:2
package build

//line template/build/show.qtpl:2
import (
	"fmt"

	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

//line template/build/show.qtpl:10
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/build/show.qtpl:10
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/build/show.qtpl:11
type ShowPage struct {
	template.Page

	Build *model.Build

	ShowManifest  bool
	ShowObjects   bool
	ShowArtifacts bool
	ShowVariables bool
	ShowOutput    bool
}

//line template/build/show.qtpl:25
func (p *ShowPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:25
	qw422016.N().S(` Build #`)
	//line template/build/show.qtpl:26
	qw422016.E().V(p.Build.ID)
	//line template/build/show.qtpl:26
	qw422016.N().S(` - Thrall `)
//line template/build/show.qtpl:27
}

//line template/build/show.qtpl:27
func (p *ShowPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:27
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:27
	p.StreamTitle(qw422016)
	//line template/build/show.qtpl:27
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:27
}

//line template/build/show.qtpl:27
func (p *ShowPage) Title() string {
	//line template/build/show.qtpl:27
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:27
	p.WriteTitle(qb422016)
	//line template/build/show.qtpl:27
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:27
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:27
	return qs422016
//line template/build/show.qtpl:27
}

//line template/build/show.qtpl:29
func (p *ShowPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:29
	qw422016.N().S(` `)
	//line template/build/show.qtpl:30
	if p.ShowManifest {
		//line template/build/show.qtpl:30
		qw422016.N().S(` `)
		//line template/build/show.qtpl:31
		p.streamrenderManifest(qw422016)
		//line template/build/show.qtpl:31
		qw422016.N().S(` `)
		//line template/build/show.qtpl:32
	} else if p.ShowObjects {
		//line template/build/show.qtpl:32
		qw422016.N().S(` `)
		//line template/build/show.qtpl:33
		p.streamrenderObjects(qw422016)
		//line template/build/show.qtpl:33
		qw422016.N().S(` `)
		//line template/build/show.qtpl:34
	} else if p.ShowArtifacts {
		//line template/build/show.qtpl:34
		qw422016.N().S(` `)
		//line template/build/show.qtpl:35
		p.streamrenderArtifacts(qw422016)
		//line template/build/show.qtpl:35
		qw422016.N().S(` `)
		//line template/build/show.qtpl:36
	} else if p.ShowVariables {
		//line template/build/show.qtpl:36
		qw422016.N().S(` `)
		//line template/build/show.qtpl:37
		p.streamrenderVariables(qw422016)
		//line template/build/show.qtpl:37
		qw422016.N().S(` `)
		//line template/build/show.qtpl:38
	} else if p.ShowOutput {
		//line template/build/show.qtpl:38
		qw422016.N().S(` `)
		//line template/build/show.qtpl:39
		p.streamrenderOutput(qw422016)
		//line template/build/show.qtpl:39
		qw422016.N().S(` `)
		//line template/build/show.qtpl:40
	} else {
		//line template/build/show.qtpl:40
		qw422016.N().S(` <div class="mb-10 overflow"> <div class="col-75 pr-5 left"> <div class="panel"> <div class="panel-header"> `)
		//line template/build/show.qtpl:45
		if p.Build.Namespace.IsZero() {
			//line template/build/show.qtpl:45
			qw422016.N().S(` <h3>Submitted by `)
			//line template/build/show.qtpl:46
			qw422016.E().S(p.Build.User.Username)
			//line template/build/show.qtpl:46
			qw422016.N().S(` &lt;`)
			//line template/build/show.qtpl:46
			qw422016.E().S(p.Build.User.Email)
			//line template/build/show.qtpl:46
			qw422016.N().S(`&gt;</h3> `)
			//line template/build/show.qtpl:47
		} else {
			//line template/build/show.qtpl:47
			qw422016.N().S(` <h3>Submitted by `)
			//line template/build/show.qtpl:48
			qw422016.E().S(p.Build.User.Username)
			//line template/build/show.qtpl:48
			qw422016.N().S(` &lt;`)
			//line template/build/show.qtpl:48
			qw422016.E().S(p.Build.User.Email)
			//line template/build/show.qtpl:48
			qw422016.N().S(`&gt; to <a href="`)
			//line template/build/show.qtpl:48
			qw422016.E().S(p.Build.Namespace.UIEndpoint())
			//line template/build/show.qtpl:48
			qw422016.N().S(`">`)
			//line template/build/show.qtpl:48
			qw422016.E().S(p.Build.Namespace.Path)
			//line template/build/show.qtpl:48
			qw422016.N().S(`</a></h3> `)
			//line template/build/show.qtpl:49
		}
		//line template/build/show.qtpl:49
		qw422016.N().S(` </div> <div class="panel-body"> <pre class="code">some commit message goes here</pre> </div> <div class="panel-footer"> <span class="code">Commit 1a2b3c4d...</span> </div> </div> </div> <div class="col-25 pl-5 right"> <div class="panel"> <table class="table"> <tr> <td>Started at:</td> <td class="align-right"> `)
		//line template/build/show.qtpl:65
		if p.Build.StartedAt != nil && p.Build.StartedAt.Valid {
			//line template/build/show.qtpl:65
			qw422016.N().S(` `)
			//line template/build/show.qtpl:66
		} else {
			//line template/build/show.qtpl:66
			qw422016.N().S(` -- `)
			//line template/build/show.qtpl:68
		}
		//line template/build/show.qtpl:68
		qw422016.N().S(` </td> </tr> <tr> <td>Finished at:</td> <td class="align-right"> `)
		//line template/build/show.qtpl:74
		if p.Build.FinishedAt != nil && p.Build.FinishedAt.Valid {
			//line template/build/show.qtpl:74
			qw422016.N().S(` `)
			//line template/build/show.qtpl:75
		} else {
			//line template/build/show.qtpl:75
			qw422016.N().S(` -- `)
			//line template/build/show.qtpl:77
		}
		//line template/build/show.qtpl:77
		qw422016.N().S(` </td> </tr> </table> </div> </div> </div> `)
		//line template/build/show.qtpl:84
		for _, s := range p.Build.Stages {
			//line template/build/show.qtpl:84
			qw422016.N().S(` `)
			//line template/build/show.qtpl:85
			if len(s.Jobs) > 0 {
				//line template/build/show.qtpl:85
				qw422016.N().S(` <div class="panel"> <div class="panel-header"> <h3>`)
				//line template/build/show.qtpl:88
				qw422016.E().S(s.Name)
				//line template/build/show.qtpl:88
				qw422016.N().S(`</h3> </div> <table class="table"> <thead> <tr> <th class="cell-pill">STATUS</th> <th>JOB</th> <th class="cell-date">STARTED</th> <th class="cell-date">FINISHED</th> </tr> </thead> <tbody> `)
				//line template/build/show.qtpl:100
				for _, j := range s.Jobs {
					//line template/build/show.qtpl:100
					qw422016.N().S(` <tr> <td class="cell-pill">`)
					//line template/build/show.qtpl:102
					StreamRenderStatus(qw422016, j.Status)
					//line template/build/show.qtpl:102
					qw422016.N().S(`</td> <td><a href="`)
					//line template/build/show.qtpl:103
					qw422016.E().S(j.UIEndpoint())
					//line template/build/show.qtpl:103
					qw422016.N().S(`">`)
					//line template/build/show.qtpl:103
					qw422016.E().S(j.Name)
					//line template/build/show.qtpl:103
					qw422016.N().S(`</a></td> <td class="cell-date"><span class="muted">--</span></td> <td class="cell-date"><span class="muted">--</span></td> </tr> `)
					//line template/build/show.qtpl:107
				}
				//line template/build/show.qtpl:107
				qw422016.N().S(` </tbody> </table> </div> `)
				//line template/build/show.qtpl:111
			}
			//line template/build/show.qtpl:111
			qw422016.N().S(` `)
			//line template/build/show.qtpl:112
		}
		//line template/build/show.qtpl:112
		qw422016.N().S(` `)
		//line template/build/show.qtpl:113
	}
	//line template/build/show.qtpl:113
	qw422016.N().S(` `)
//line template/build/show.qtpl:114
}

//line template/build/show.qtpl:114
func (p *ShowPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:114
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:114
	p.StreamBody(qw422016)
	//line template/build/show.qtpl:114
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:114
}

//line template/build/show.qtpl:114
func (p *ShowPage) Body() string {
	//line template/build/show.qtpl:114
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:114
	p.WriteBody(qb422016)
	//line template/build/show.qtpl:114
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:114
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:114
	return qs422016
//line template/build/show.qtpl:114
}

//line template/build/show.qtpl:116
func (p *ShowPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:116
	qw422016.N().S(` <a href="/" class="back">`)
	//line template/build/show.qtpl:117
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
	//line template/build/show.qtpl:117
	qw422016.N().S(`</a> Build #`)
	//line template/build/show.qtpl:117
	qw422016.E().V(p.Build.ID)
	//line template/build/show.qtpl:117
	qw422016.N().S(` `)
	//line template/build/show.qtpl:117
	StreamRenderStatus(qw422016, p.Build.Status)
	//line template/build/show.qtpl:117
	qw422016.N().S(` `)
//line template/build/show.qtpl:118
}

//line template/build/show.qtpl:118
func (p *ShowPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:118
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:118
	p.StreamHeader(qw422016)
	//line template/build/show.qtpl:118
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:118
}

//line template/build/show.qtpl:118
func (p *ShowPage) Header() string {
	//line template/build/show.qtpl:118
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:118
	p.WriteHeader(qb422016)
	//line template/build/show.qtpl:118
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:118
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:118
	return qs422016
//line template/build/show.qtpl:118
}

//line template/build/show.qtpl:120
func (p *ShowPage) StreamActions(qw422016 *qt422016.Writer) {
//line template/build/show.qtpl:120
}

//line template/build/show.qtpl:120
func (p *ShowPage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:120
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:120
	p.StreamActions(qw422016)
	//line template/build/show.qtpl:120
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:120
}

//line template/build/show.qtpl:120
func (p *ShowPage) Actions() string {
	//line template/build/show.qtpl:120
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:120
	p.WriteActions(qb422016)
	//line template/build/show.qtpl:120
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:120
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:120
	return qs422016
//line template/build/show.qtpl:120
}

//line template/build/show.qtpl:122
func (p *ShowPage) StreamNavigation(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:122
	qw422016.N().S(` `)
	//line template/build/show.qtpl:123
	qw422016.N().S(`<li>`)
	//line template/build/show.qtpl:124
	template.StreamRenderLink(qw422016, p.Build.UIEndpoint(), p.URI)
	//line template/build/show.qtpl:124
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12 9c1.641 0 3 1.359 3 3s-1.359 3-3 3-3-1.359-3-3 1.359-3 3-3zM12 17.016c2.766 0 5.016-2.25 5.016-5.016s-2.25-5.016-5.016-5.016-5.016 2.25-5.016 5.016 2.25 5.016 5.016 5.016zM12 4.5c5.016 0 9.281 3.094 11.016 7.5-1.734 4.406-6 7.5-11.016 7.5s-9.281-3.094-11.016-7.5c1.734-4.406 6-7.5 11.016-7.5z"></path>
</svg>
`)
	//line template/build/show.qtpl:124
	qw422016.N().S(`<span>Overview</span></a></li><li>`)
	//line template/build/show.qtpl:125
	template.StreamRenderLink(qw422016, p.Build.UIEndpoint("manifest"), p.URI)
	//line template/build/show.qtpl:125
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M6.984 12.984v-1.969h14.016v1.969h-14.016zM6.984 18.984v-1.969h14.016v1.969h-14.016zM6.984 5.016h14.016v1.969h-14.016v-1.969zM2.016 11.016v-1.031h3v0.938l-1.828 2.063h1.828v1.031h-3v-0.938l1.781-2.063h-1.781zM3 8.016v-3h-0.984v-1.031h1.969v4.031h-0.984zM2.016 17.016v-1.031h3v4.031h-3v-1.031h1.969v-0.469h-0.984v-1.031h0.984v-0.469h-1.969z"></path>
</svg>
`)
	//line template/build/show.qtpl:125
	qw422016.N().S(`<span>Manifest</span></a></li><li>`)
	//line template/build/show.qtpl:126
	template.StreamRenderLink(qw422016, p.Build.UIEndpoint("objects"), p.URI)
	//line template/build/show.qtpl:126
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M5.016 18h13.969v2.016h-13.969v-2.016zM9 15.984v-6h-3.984l6.984-6.984 6.984 6.984h-3.984v6h-6z"></path>
</svg>
`)
	//line template/build/show.qtpl:126
	qw422016.N().S(`<span>Objects</span></a></li><li>`)
	//line template/build/show.qtpl:127
	template.StreamRenderLink(qw422016, p.Build.UIEndpoint("artifacts"), p.URI)
	//line template/build/show.qtpl:127
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M5.016 18h13.969v2.016h-13.969v-2.016zM18.984 9l-6.984 6.984-6.984-6.984h3.984v-6h6v6h3.984z"></path>
</svg>
`)
	//line template/build/show.qtpl:127
	qw422016.N().S(`<span>Artifacts</span></a></li><li>`)
	//line template/build/show.qtpl:128
	template.StreamRenderLink(qw422016, p.Build.UIEndpoint("variables"), p.URI)
	//line template/build/show.qtpl:128
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M14.578 16.594l4.641-4.594-4.641-4.594 1.406-1.406 6 6-6 6zM9.422 16.594l-1.406 1.406-6-6 6-6 1.406 1.406-4.641 4.594z"></path>
</svg>
`)
	//line template/build/show.qtpl:128
	qw422016.N().S(`<span>Variables</span></a></li><li>`)
	//line template/build/show.qtpl:129
	template.StreamRenderLink(qw422016, p.Build.UIEndpoint("output"), p.URI)
	//line template/build/show.qtpl:129
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12.984 9h5.531l-5.531-5.484v5.484zM15.984 14.016v-2.016h-7.969v2.016h7.969zM15.984 18v-2.016h-7.969v2.016h7.969zM14.016 2.016l6 6v12c0 1.078-0.938 1.969-2.016 1.969h-12c-1.078 0-2.016-0.891-2.016-1.969l0.047-16.031c0-1.078 0.891-1.969 1.969-1.969h8.016z"></path>
</svg>
`)
	//line template/build/show.qtpl:129
	qw422016.N().S(`<span>Output</span></a></li>`)
	//line template/build/show.qtpl:130
	qw422016.N().S(` `)
//line template/build/show.qtpl:131
}

//line template/build/show.qtpl:131
func (p *ShowPage) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:131
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:131
	p.StreamNavigation(qw422016)
	//line template/build/show.qtpl:131
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:131
}

//line template/build/show.qtpl:131
func (p *ShowPage) Navigation() string {
	//line template/build/show.qtpl:131
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:131
	p.WriteNavigation(qb422016)
	//line template/build/show.qtpl:131
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:131
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:131
	return qs422016
//line template/build/show.qtpl:131
}

//line template/build/show.qtpl:133
func (p *ShowPage) streamrenderManifest(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:133
	qw422016.N().S(` <div class="panel"> <div class="panel-header"> <ul class="panel-actions"> <li><a class="btn btn-primary" href="`)
	//line template/build/show.qtpl:137
	qw422016.E().S(p.Build.UIEndpoint("manifest", "raw"))
	//line template/build/show.qtpl:137
	qw422016.N().S(`">`)
	//line template/build/show.qtpl:137
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12.984 9h5.531l-5.531-5.484v5.484zM15.984 14.016v-2.016h-7.969v2.016h7.969zM15.984 18v-2.016h-7.969v2.016h7.969zM14.016 2.016l6 6v12c0 1.078-0.938 1.969-2.016 1.969h-12c-1.078 0-2.016-0.891-2.016-1.969l0.047-16.031c0-1.078 0.891-1.969 1.969-1.969h8.016z"></path>
</svg>
`)
	//line template/build/show.qtpl:137
	qw422016.N().S(`<span>Raw</span></a></li> </ul> </div> `)
	//line template/build/show.qtpl:140
	template.StreamRenderCode(qw422016, p.Build.Manifest)
	//line template/build/show.qtpl:140
	qw422016.N().S(` </div> `)
//line template/build/show.qtpl:142
}

//line template/build/show.qtpl:142
func (p *ShowPage) writerenderManifest(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:142
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:142
	p.streamrenderManifest(qw422016)
	//line template/build/show.qtpl:142
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:142
}

//line template/build/show.qtpl:142
func (p *ShowPage) renderManifest() string {
	//line template/build/show.qtpl:142
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:142
	p.writerenderManifest(qb422016)
	//line template/build/show.qtpl:142
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:142
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:142
	return qs422016
//line template/build/show.qtpl:142
}

//line template/build/show.qtpl:144
func (p *ShowPage) streamrenderObjects(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:144
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/show.qtpl:146
	if len(p.Build.Objects) > 0 {
		//line template/build/show.qtpl:146
		qw422016.N().S(` <table class="table"> <thead> <tr> <th>SOURCE</th> <th>NAME</th> <th>HASHES</th> <th></th> </tr> </thead> <tbody> `)
		//line template/build/show.qtpl:157
		for _, o := range p.Build.Objects {
			//line template/build/show.qtpl:157
			qw422016.N().S(` <tr> <td> `)
			//line template/build/show.qtpl:160
			if o.Object != nil {
				//line template/build/show.qtpl:160
				qw422016.N().S(` <a href="/objects/`)
				//line template/build/show.qtpl:161
				qw422016.E().V(o.Object.ID)
				//line template/build/show.qtpl:161
				qw422016.N().S(`">`)
				//line template/build/show.qtpl:161
				qw422016.E().S(o.Source)
				//line template/build/show.qtpl:161
				qw422016.N().S(`</a> `)
				//line template/build/show.qtpl:162
			} else {
				//line template/build/show.qtpl:162
				qw422016.N().S(` <a title="Object not found"><strike>`)
				//line template/build/show.qtpl:163
				qw422016.E().S(o.Source)
				//line template/build/show.qtpl:163
				qw422016.N().S(`</strike></a> `)
				//line template/build/show.qtpl:164
			}
			//line template/build/show.qtpl:164
			qw422016.N().S(` </td> <td><code>`)
			//line template/build/show.qtpl:166
			qw422016.E().S(o.Name)
			//line template/build/show.qtpl:166
			qw422016.N().S(`</code></td> <td> <div class="mb-10">MD5 <span class="code right">`)
			//line template/build/show.qtpl:168
			if o.Object != nil {
				//line template/build/show.qtpl:168
				qw422016.E().S(fmt.Sprintf("%x", o.Object.MD5))
				//line template/build/show.qtpl:168
			} else {
				//line template/build/show.qtpl:168
				qw422016.N().S(`--`)
				//line template/build/show.qtpl:168
			}
			//line template/build/show.qtpl:168
			qw422016.N().S(`</span></div> <div>SHA256 <span class="code right">`)
			//line template/build/show.qtpl:169
			if o.Object != nil {
				//line template/build/show.qtpl:169
				qw422016.E().S(fmt.Sprintf("%x", o.Object.SHA256))
				//line template/build/show.qtpl:169
			} else {
				//line template/build/show.qtpl:169
				qw422016.N().S(`--`)
				//line template/build/show.qtpl:169
			}
			//line template/build/show.qtpl:169
			qw422016.N().S(`</span></div> </td> <td class="align-right"> `)
			//line template/build/show.qtpl:172
			if o.Placed {
				//line template/build/show.qtpl:172
				qw422016.N().S(` <span class="pill pill-green">`)
				//line template/build/show.qtpl:173
				qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M9 16.172l10.594-10.594 1.406 1.406-12 12-5.578-5.578 1.406-1.406z"></path>
</svg>
`)
				//line template/build/show.qtpl:173
				qw422016.N().S(` Placed</span> `)
				//line template/build/show.qtpl:174
			} else {
				//line template/build/show.qtpl:174
				qw422016.N().S(` <span class="pill pill-red">`)
				//line template/build/show.qtpl:175
				qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M18.984 6.422l-5.578 5.578 5.578 5.578-1.406 1.406-5.578-5.578-5.578 5.578-1.406-1.406 5.578-5.578-5.578-5.578 1.406-1.406 5.578 5.578 5.578-5.578z"></path>
</svg>
`)
				//line template/build/show.qtpl:175
				qw422016.N().S(` Not Placed</span> `)
				//line template/build/show.qtpl:176
			}
			//line template/build/show.qtpl:176
			qw422016.N().S(` </td> </tr> `)
			//line template/build/show.qtpl:179
		}
		//line template/build/show.qtpl:179
		qw422016.N().S(` </tbody> </table> `)
		//line template/build/show.qtpl:182
	} else {
		//line template/build/show.qtpl:182
		qw422016.N().S(` <div class="panel-message muted">No objects have been placed for this build.</div> `)
		//line template/build/show.qtpl:184
	}
	//line template/build/show.qtpl:184
	qw422016.N().S(` </div> `)
//line template/build/show.qtpl:186
}

//line template/build/show.qtpl:186
func (p *ShowPage) writerenderObjects(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:186
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:186
	p.streamrenderObjects(qw422016)
	//line template/build/show.qtpl:186
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:186
}

//line template/build/show.qtpl:186
func (p *ShowPage) renderObjects() string {
	//line template/build/show.qtpl:186
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:186
	p.writerenderObjects(qb422016)
	//line template/build/show.qtpl:186
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:186
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:186
	return qs422016
//line template/build/show.qtpl:186
}

//line template/build/show.qtpl:188
func (p *ShowPage) streamrenderArtifacts(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:188
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/show.qtpl:190
	if len(p.Build.Artifacts) > 0 {
		//line template/build/show.qtpl:190
		qw422016.N().S(` `)
		//line template/build/show.qtpl:191
	} else {
		//line template/build/show.qtpl:191
		qw422016.N().S(` <div class="panel-message muted">No artifacts have been collected from this build.</div> `)
		//line template/build/show.qtpl:193
	}
	//line template/build/show.qtpl:193
	qw422016.N().S(` </div> `)
//line template/build/show.qtpl:195
}

//line template/build/show.qtpl:195
func (p *ShowPage) writerenderArtifacts(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:195
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:195
	p.streamrenderArtifacts(qw422016)
	//line template/build/show.qtpl:195
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:195
}

//line template/build/show.qtpl:195
func (p *ShowPage) renderArtifacts() string {
	//line template/build/show.qtpl:195
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:195
	p.writerenderArtifacts(qb422016)
	//line template/build/show.qtpl:195
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:195
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:195
	return qs422016
//line template/build/show.qtpl:195
}

//line template/build/show.qtpl:197
func (p *ShowPage) streamrenderVariables(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:197
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/show.qtpl:199
	if len(p.Build.Variables) > 0 {
		//line template/build/show.qtpl:199
		qw422016.N().S(` `)
		//line template/build/show.qtpl:201
	} else {
		//line template/build/show.qtpl:201
		qw422016.N().S(` <div class="panel-message muted">No variables have been set for this build.</div> `)
		//line template/build/show.qtpl:203
	}
	//line template/build/show.qtpl:203
	qw422016.N().S(` </div> `)
//line template/build/show.qtpl:205
}

//line template/build/show.qtpl:205
func (p *ShowPage) writerenderVariables(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:205
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:205
	p.streamrenderVariables(qw422016)
	//line template/build/show.qtpl:205
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:205
}

//line template/build/show.qtpl:205
func (p *ShowPage) renderVariables() string {
	//line template/build/show.qtpl:205
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:205
	p.writerenderVariables(qb422016)
	//line template/build/show.qtpl:205
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:205
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:205
	return qs422016
//line template/build/show.qtpl:205
}

//line template/build/show.qtpl:207
func (p *ShowPage) streamrenderOutput(qw422016 *qt422016.Writer) {
	//line template/build/show.qtpl:207
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/show.qtpl:209
	if p.Build.Output.Valid {
		//line template/build/show.qtpl:209
		qw422016.N().S(` <div class="panel-header"> <ul class="panel-actions"> <li><a class="btn btn-primary" href="`)
		//line template/build/show.qtpl:212
		qw422016.E().S(p.Build.UIEndpoint("output", "raw"))
		//line template/build/show.qtpl:212
		qw422016.N().S(`">`)
		//line template/build/show.qtpl:212
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12.984 9h5.531l-5.531-5.484v5.484zM15.984 14.016v-2.016h-7.969v2.016h7.969zM15.984 18v-2.016h-7.969v2.016h7.969zM14.016 2.016l6 6v12c0 1.078-0.938 1.969-2.016 1.969h-12c-1.078 0-2.016-0.891-2.016-1.969l0.047-16.031c0-1.078 0.891-1.969 1.969-1.969h8.016z"></path>
</svg>
`)
		//line template/build/show.qtpl:212
		qw422016.N().S(`<span>Raw</span></a></li> </ul> </div> `)
		//line template/build/show.qtpl:215
		template.StreamRenderCode(qw422016, p.Build.Output.String)
		//line template/build/show.qtpl:215
		qw422016.N().S(` `)
		//line template/build/show.qtpl:216
	} else {
		//line template/build/show.qtpl:216
		qw422016.N().S(` <div class="panel-message muted">No build output has been produced.</div> `)
		//line template/build/show.qtpl:218
	}
	//line template/build/show.qtpl:218
	qw422016.N().S(` </div> `)
//line template/build/show.qtpl:220
}

//line template/build/show.qtpl:220
func (p *ShowPage) writerenderOutput(qq422016 qtio422016.Writer) {
	//line template/build/show.qtpl:220
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/show.qtpl:220
	p.streamrenderOutput(qw422016)
	//line template/build/show.qtpl:220
	qt422016.ReleaseWriter(qw422016)
//line template/build/show.qtpl:220
}

//line template/build/show.qtpl:220
func (p *ShowPage) renderOutput() string {
	//line template/build/show.qtpl:220
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/show.qtpl:220
	p.writerenderOutput(qb422016)
	//line template/build/show.qtpl:220
	qs422016 := string(qb422016.B)
	//line template/build/show.qtpl:220
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/show.qtpl:220
	return qs422016
//line template/build/show.qtpl:220
}
