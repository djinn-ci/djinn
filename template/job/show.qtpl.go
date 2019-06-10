// This file is automatically generated by qtc from "show.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/job/show.qtpl:2
package job

//line template/job/show.qtpl:2
import (
	"fmt"

	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/build"
)

//line template/job/show.qtpl:11
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/job/show.qtpl:11
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/job/show.qtpl:12
type ShowPage struct {
	template.Page

	Job *model.Job

	ShowOutput bool
}

//line template/job/show.qtpl:22
func (p *ShowPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/job/show.qtpl:22
	qw422016.N().S(` `)
	//line template/job/show.qtpl:23
	qw422016.E().S(p.Job.Name)
	//line template/job/show.qtpl:23
	qw422016.N().S(` - Thrall `)
//line template/job/show.qtpl:24
}

//line template/job/show.qtpl:24
func (p *ShowPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/job/show.qtpl:24
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/job/show.qtpl:24
	p.StreamTitle(qw422016)
	//line template/job/show.qtpl:24
	qt422016.ReleaseWriter(qw422016)
//line template/job/show.qtpl:24
}

//line template/job/show.qtpl:24
func (p *ShowPage) Title() string {
	//line template/job/show.qtpl:24
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/job/show.qtpl:24
	p.WriteTitle(qb422016)
	//line template/job/show.qtpl:24
	qs422016 := string(qb422016.B)
	//line template/job/show.qtpl:24
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/job/show.qtpl:24
	return qs422016
//line template/job/show.qtpl:24
}

//line template/job/show.qtpl:26
func (p *ShowPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/job/show.qtpl:26
	qw422016.N().S(` `)
	//line template/job/show.qtpl:27
	if p.ShowOutput {
		//line template/job/show.qtpl:27
		qw422016.N().S(` `)
		//line template/job/show.qtpl:28
		p.streamrenderOutput(qw422016)
		//line template/job/show.qtpl:28
		qw422016.N().S(` `)
		//line template/job/show.qtpl:29
	} else {
		//line template/job/show.qtpl:29
		qw422016.N().S(` <div class="panel mb-10"> <table class="table"> <tr> <td>Status:</td> <td class="align-right">`)
		//line template/job/show.qtpl:34
		build.StreamRenderStatus(qw422016, p.Job.Status)
		//line template/job/show.qtpl:34
		qw422016.N().S(`</td> </tr> <tr> <td>Started at:</td> <td class="align-right"> `)
		//line template/job/show.qtpl:39
		if p.Job.StartedAt != nil && p.Job.StartedAt.Valid {
			//line template/job/show.qtpl:39
			qw422016.N().S(` `)
			//line template/job/show.qtpl:40
		} else {
			//line template/job/show.qtpl:40
			qw422016.N().S(` -- `)
			//line template/job/show.qtpl:42
		}
		//line template/job/show.qtpl:42
		qw422016.N().S(` </td> </tr> <tr> <td>Finished at:</td> <td class="align-right"> `)
		//line template/job/show.qtpl:48
		if p.Job.FinishedAt != nil && p.Job.FinishedAt.Valid {
			//line template/job/show.qtpl:48
			qw422016.N().S(` `)
			//line template/job/show.qtpl:49
		} else {
			//line template/job/show.qtpl:49
			qw422016.N().S(` -- `)
			//line template/job/show.qtpl:51
		}
		//line template/job/show.qtpl:51
		qw422016.N().S(` </td> </tr> </table> </div> <div class="col-75 pr-5 left"> <div class="panel"> <div class="panel-header"><h3>Artifacts</h3></div> `)
		//line template/job/show.qtpl:59
		if len(p.Job.Artifacts) > 0 {
			//line template/job/show.qtpl:59
			qw422016.N().S(` <table class="table"> <thead> <tr> <th>SOURCE</th> <th>NAME</th> <th>HASHES</th> </tr> </thead> <tbody> `)
			//line template/job/show.qtpl:69
			for _, a := range p.Job.Artifacts {
				//line template/job/show.qtpl:69
				qw422016.N().S(` <tr> <td><code>`)
				//line template/job/show.qtpl:71
				qw422016.E().S(a.Source)
				//line template/job/show.qtpl:71
				qw422016.N().S(`</code></td> <td>`)
				//line template/job/show.qtpl:72
				qw422016.E().S(a.Name)
				//line template/job/show.qtpl:72
				qw422016.N().S(`</td> <td> <div class="mb-10">MD5 <code class="right">`)
				//line template/job/show.qtpl:74
				qw422016.E().S(fmt.Sprintf("%x", a.MD5))
				//line template/job/show.qtpl:74
				qw422016.N().S(`</code></div> <div class="mb-10">SHA256 <code class="right">`)
				//line template/job/show.qtpl:75
				qw422016.E().S(fmt.Sprintf("%x", a.SHA256))
				//line template/job/show.qtpl:75
				qw422016.N().S(`</code></div> </td> </tr> `)
				//line template/job/show.qtpl:78
			}
			//line template/job/show.qtpl:78
			qw422016.N().S(` </tbody> </table> `)
			//line template/job/show.qtpl:81
		} else {
			//line template/job/show.qtpl:81
			qw422016.N().S(` <div class="panel-message muted">No artifacts have been collected for this job.</div> `)
			//line template/job/show.qtpl:83
		}
		//line template/job/show.qtpl:83
		qw422016.N().S(` </div> </div> <div class="col-25 pl-5 right"> <div class="panel"> <div class="panel-header"><h3>Dependencies</h3></div> `)
		//line template/job/show.qtpl:89
		if len(p.Job.Dependencies) > 0 {
			//line template/job/show.qtpl:89
			qw422016.N().S(` <table class="table"> `)
			//line template/job/show.qtpl:91
			for _, j := range p.Job.Dependencies {
				//line template/job/show.qtpl:91
				qw422016.N().S(` <tr><td><a href="`)
				//line template/job/show.qtpl:92
				qw422016.E().S(j.UIEndpoint())
				//line template/job/show.qtpl:92
				qw422016.N().S(`">`)
				//line template/job/show.qtpl:92
				qw422016.E().S(j.Name)
				//line template/job/show.qtpl:92
				qw422016.N().S(`</a></td></tr> `)
				//line template/job/show.qtpl:93
			}
			//line template/job/show.qtpl:93
			qw422016.N().S(` </table> `)
			//line template/job/show.qtpl:95
		} else {
			//line template/job/show.qtpl:95
			qw422016.N().S(` <div class="panel-message muted">No job dependencies.</div> `)
			//line template/job/show.qtpl:97
		}
		//line template/job/show.qtpl:97
		qw422016.N().S(` </div> </div> `)
		//line template/job/show.qtpl:100
	}
	//line template/job/show.qtpl:100
	qw422016.N().S(` `)
//line template/job/show.qtpl:101
}

//line template/job/show.qtpl:101
func (p *ShowPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/job/show.qtpl:101
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/job/show.qtpl:101
	p.StreamBody(qw422016)
	//line template/job/show.qtpl:101
	qt422016.ReleaseWriter(qw422016)
//line template/job/show.qtpl:101
}

//line template/job/show.qtpl:101
func (p *ShowPage) Body() string {
	//line template/job/show.qtpl:101
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/job/show.qtpl:101
	p.WriteBody(qb422016)
	//line template/job/show.qtpl:101
	qs422016 := string(qb422016.B)
	//line template/job/show.qtpl:101
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/job/show.qtpl:101
	return qs422016
//line template/job/show.qtpl:101
}

//line template/job/show.qtpl:103
func (p *ShowPage) streamrenderOutput(qw422016 *qt422016.Writer) {
	//line template/job/show.qtpl:103
	qw422016.N().S(` <div class="panel"> `)
	//line template/job/show.qtpl:105
	if p.Job.Output.Valid && p.Job.Output.String != "" {
		//line template/job/show.qtpl:105
		qw422016.N().S(` <div class="panel-header"> <ul class="panel-actions"> <li><a class="btn btn-primary" href="`)
		//line template/job/show.qtpl:108
		qw422016.E().S(p.Job.UIEndpoint("output", "raw"))
		//line template/job/show.qtpl:108
		qw422016.N().S(`">`)
		//line template/job/show.qtpl:108
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12.984 9h5.531l-5.531-5.484v5.484zM15.984 14.016v-2.016h-7.969v2.016h7.969zM15.984 18v-2.016h-7.969v2.016h7.969zM14.016 2.016l6 6v12c0 1.078-0.938 1.969-2.016 1.969h-12c-1.078 0-2.016-0.891-2.016-1.969l0.047-16.031c0-1.078 0.891-1.969 1.969-1.969h8.016z"></path>
</svg>
`)
		//line template/job/show.qtpl:108
		qw422016.N().S(`<span>Raw</span></a></li> </ul> </div> `)
		//line template/job/show.qtpl:111
		template.StreamRenderCode(qw422016, p.Job.Output.String)
		//line template/job/show.qtpl:111
		qw422016.N().S(` `)
		//line template/job/show.qtpl:112
	} else {
		//line template/job/show.qtpl:112
		qw422016.N().S(` <div class="panel-message muted">No job output has been produced.</div> `)
		//line template/job/show.qtpl:114
	}
	//line template/job/show.qtpl:114
	qw422016.N().S(` </div> `)
//line template/job/show.qtpl:116
}

//line template/job/show.qtpl:116
func (p *ShowPage) writerenderOutput(qq422016 qtio422016.Writer) {
	//line template/job/show.qtpl:116
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/job/show.qtpl:116
	p.streamrenderOutput(qw422016)
	//line template/job/show.qtpl:116
	qt422016.ReleaseWriter(qw422016)
//line template/job/show.qtpl:116
}

//line template/job/show.qtpl:116
func (p *ShowPage) renderOutput() string {
	//line template/job/show.qtpl:116
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/job/show.qtpl:116
	p.writerenderOutput(qb422016)
	//line template/job/show.qtpl:116
	qs422016 := string(qb422016.B)
	//line template/job/show.qtpl:116
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/job/show.qtpl:116
	return qs422016
//line template/job/show.qtpl:116
}

//line template/job/show.qtpl:118
func (p *ShowPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/job/show.qtpl:118
	qw422016.N().S(` <a href="`)
	//line template/job/show.qtpl:119
	qw422016.E().S(p.Job.Build.UIEndpoint())
	//line template/job/show.qtpl:119
	qw422016.N().S(`" class="back">`)
	//line template/job/show.qtpl:119
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
	//line template/job/show.qtpl:119
	qw422016.N().S(`</a> Build #`)
	//line template/job/show.qtpl:119
	qw422016.E().V(p.Job.BuildID)
	//line template/job/show.qtpl:119
	qw422016.N().S(` / `)
	//line template/job/show.qtpl:119
	qw422016.E().S(p.Job.Stage.Name)
	//line template/job/show.qtpl:119
	qw422016.N().S(` - `)
	//line template/job/show.qtpl:119
	qw422016.E().S(p.Job.Name)
	//line template/job/show.qtpl:119
	qw422016.N().S(` `)
	//line template/job/show.qtpl:119
	build.StreamRenderStatus(qw422016, p.Job.Build.Status)
	//line template/job/show.qtpl:119
	qw422016.N().S(` `)
//line template/job/show.qtpl:120
}

//line template/job/show.qtpl:120
func (p *ShowPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/job/show.qtpl:120
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/job/show.qtpl:120
	p.StreamHeader(qw422016)
	//line template/job/show.qtpl:120
	qt422016.ReleaseWriter(qw422016)
//line template/job/show.qtpl:120
}

//line template/job/show.qtpl:120
func (p *ShowPage) Header() string {
	//line template/job/show.qtpl:120
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/job/show.qtpl:120
	p.WriteHeader(qb422016)
	//line template/job/show.qtpl:120
	qs422016 := string(qb422016.B)
	//line template/job/show.qtpl:120
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/job/show.qtpl:120
	return qs422016
//line template/job/show.qtpl:120
}

//line template/job/show.qtpl:122
func (p *ShowPage) StreamActions(qw422016 *qt422016.Writer) {
//line template/job/show.qtpl:122
}

//line template/job/show.qtpl:122
func (p *ShowPage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/job/show.qtpl:122
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/job/show.qtpl:122
	p.StreamActions(qw422016)
	//line template/job/show.qtpl:122
	qt422016.ReleaseWriter(qw422016)
//line template/job/show.qtpl:122
}

//line template/job/show.qtpl:122
func (p *ShowPage) Actions() string {
	//line template/job/show.qtpl:122
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/job/show.qtpl:122
	p.WriteActions(qb422016)
	//line template/job/show.qtpl:122
	qs422016 := string(qb422016.B)
	//line template/job/show.qtpl:122
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/job/show.qtpl:122
	return qs422016
//line template/job/show.qtpl:122
}

//line template/job/show.qtpl:124
func (p *ShowPage) StreamNavigation(qw422016 *qt422016.Writer) {
	//line template/job/show.qtpl:124
	qw422016.N().S(` `)
	//line template/job/show.qtpl:125
	qw422016.N().S(`<li>`)
	//line template/job/show.qtpl:126
	template.StreamRenderLink(qw422016, p.Job.UIEndpoint(), p.URI)
	//line template/job/show.qtpl:126
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12 9c1.641 0 3 1.359 3 3s-1.359 3-3 3-3-1.359-3-3 1.359-3 3-3zM12 17.016c2.766 0 5.016-2.25 5.016-5.016s-2.25-5.016-5.016-5.016-5.016 2.25-5.016 5.016 2.25 5.016 5.016 5.016zM12 4.5c5.016 0 9.281 3.094 11.016 7.5-1.734 4.406-6 7.5-11.016 7.5s-9.281-3.094-11.016-7.5c1.734-4.406 6-7.5 11.016-7.5z"></path>
</svg>
`)
	//line template/job/show.qtpl:126
	qw422016.N().S(`<span>Overview</span></a></li><li>`)
	//line template/job/show.qtpl:127
	template.StreamRenderLink(qw422016, p.Job.UIEndpoint("output"), p.URI)
	//line template/job/show.qtpl:127
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12.984 9h5.531l-5.531-5.484v5.484zM15.984 14.016v-2.016h-7.969v2.016h7.969zM15.984 18v-2.016h-7.969v2.016h7.969zM14.016 2.016l6 6v12c0 1.078-0.938 1.969-2.016 1.969h-12c-1.078 0-2.016-0.891-2.016-1.969l0.047-16.031c0-1.078 0.891-1.969 1.969-1.969h8.016z"></path>
</svg>
`)
	//line template/job/show.qtpl:127
	qw422016.N().S(`<span>Output</span></a></li>`)
	//line template/job/show.qtpl:128
	qw422016.N().S(` `)
//line template/job/show.qtpl:129
}

//line template/job/show.qtpl:129
func (p *ShowPage) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/job/show.qtpl:129
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/job/show.qtpl:129
	p.StreamNavigation(qw422016)
	//line template/job/show.qtpl:129
	qt422016.ReleaseWriter(qw422016)
//line template/job/show.qtpl:129
}

//line template/job/show.qtpl:129
func (p *ShowPage) Navigation() string {
	//line template/job/show.qtpl:129
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/job/show.qtpl:129
	p.WriteNavigation(qb422016)
	//line template/job/show.qtpl:129
	qs422016 := string(qb422016.B)
	//line template/job/show.qtpl:129
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/job/show.qtpl:129
	return qs422016
//line template/job/show.qtpl:129
}
