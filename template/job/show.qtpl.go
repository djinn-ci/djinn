// This file is automatically generated by qtc from "show.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/job/show.qtpl:2
package job

//line template/job/show.qtpl:2
import (
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/artifact"
	"github.com/andrewpillar/thrall/template/build"
)

//line template/job/show.qtpl:10
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/job/show.qtpl:10
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/job/show.qtpl:11
type ShowPage struct {
	template.BasePage

	Job *model.Job
}

//line template/job/show.qtpl:19
func (p *ShowPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/job/show.qtpl:19
	qw422016.N().S(` `)
	//line template/job/show.qtpl:20
	qw422016.E().S(p.Job.Name)
	//line template/job/show.qtpl:20
	qw422016.N().S(` - Thrall `)
//line template/job/show.qtpl:21
}

//line template/job/show.qtpl:21
func (p *ShowPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/job/show.qtpl:21
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/job/show.qtpl:21
	p.StreamTitle(qw422016)
	//line template/job/show.qtpl:21
	qt422016.ReleaseWriter(qw422016)
//line template/job/show.qtpl:21
}

//line template/job/show.qtpl:21
func (p *ShowPage) Title() string {
	//line template/job/show.qtpl:21
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/job/show.qtpl:21
	p.WriteTitle(qb422016)
	//line template/job/show.qtpl:21
	qs422016 := string(qb422016.B)
	//line template/job/show.qtpl:21
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/job/show.qtpl:21
	return qs422016
//line template/job/show.qtpl:21
}

//line template/job/show.qtpl:23
func (p *ShowPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/job/show.qtpl:23
	qw422016.N().S(` <div class="overflow"> <div class="col-25 pr-5 left"> <div class="panel"> <table class="table"> <tr> <td>Status:</td> <td class="align-right">`)
	//line template/job/show.qtpl:30
	template.StreamRenderStatus(qw422016, p.Job.Status)
	//line template/job/show.qtpl:30
	qw422016.N().S(`</td> </tr> <tr> <td>Started at:</td> <td class="align-right"> `)
	//line template/job/show.qtpl:35
	if p.Job.StartedAt.Valid {
		//line template/job/show.qtpl:35
		qw422016.N().S(` `)
		//line template/job/show.qtpl:36
		qw422016.E().S(p.Job.StartedAt.Time.Format("2006-01-02T15:04:05"))
		//line template/job/show.qtpl:36
		qw422016.N().S(` `)
		//line template/job/show.qtpl:37
	} else {
		//line template/job/show.qtpl:37
		qw422016.N().S(` <span class="muted">--</span> `)
		//line template/job/show.qtpl:39
	}
	//line template/job/show.qtpl:39
	qw422016.N().S(` </td> </tr> <tr> <td>Finished at:</td> <td class="align-right"> `)
	//line template/job/show.qtpl:45
	if p.Job.FinishedAt.Valid {
		//line template/job/show.qtpl:45
		qw422016.N().S(` `)
		//line template/job/show.qtpl:46
		qw422016.E().S(p.Job.FinishedAt.Time.Format("2006-01-02T15:04:05"))
		//line template/job/show.qtpl:46
		qw422016.N().S(` `)
		//line template/job/show.qtpl:47
	} else {
		//line template/job/show.qtpl:47
		qw422016.N().S(` <span class="muted">--</span> `)
		//line template/job/show.qtpl:49
	}
	//line template/job/show.qtpl:49
	qw422016.N().S(` </td> </tr> <tr> <td>Duration:</td> <td class="align-right"> `)
	//line template/job/show.qtpl:55
	if !p.Job.FinishedAt.Valid || !p.Job.StartedAt.Valid {
		//line template/job/show.qtpl:55
		qw422016.N().S(` <span class="muted">--</span> `)
		//line template/job/show.qtpl:57
	} else {
		//line template/job/show.qtpl:57
		qw422016.N().S(` `)
		//line template/job/show.qtpl:58
		qw422016.E().V(p.Job.FinishedAt.Time.Sub(p.Job.StartedAt.Time))
		//line template/job/show.qtpl:58
		qw422016.N().S(` `)
		//line template/job/show.qtpl:59
	}
	//line template/job/show.qtpl:59
	qw422016.N().S(` </td> </tr> </table> </div> </div> <div class="col-75 pl-5 right"> `)
	//line template/job/show.qtpl:66
	build.StreamRenderTrigger(qw422016, p.Job.Build)
	//line template/job/show.qtpl:66
	qw422016.N().S(` <div class="panel"> `)
	//line template/job/show.qtpl:68
	if p.Job.Output.Valid {
		//line template/job/show.qtpl:68
		qw422016.N().S(` <div class="panel-header"> <h3>Output</h3> <ul class="panel-actions"> <li><a class="btn btn-primary" href="`)
		//line template/job/show.qtpl:72
		qw422016.E().S(p.Job.UIEndpoint("output", "raw"))
		//line template/job/show.qtpl:72
		qw422016.N().S(`">`)
		//line template/job/show.qtpl:72
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12.984 9h5.531l-5.531-5.484v5.484zM15.984 14.016v-2.016h-7.969v2.016h7.969zM15.984 18v-2.016h-7.969v2.016h7.969zM14.016 2.016l6 6v12c0 1.078-0.938 1.969-2.016 1.969h-12c-1.078 0-2.016-0.891-2.016-1.969l0.047-16.031c0-1.078 0.891-1.969 1.969-1.969h8.016z"></path>
</svg>
`)
		//line template/job/show.qtpl:72
		qw422016.N().S(`<span>Raw</span></a></li> </ul> </div> `)
		//line template/job/show.qtpl:75
		template.StreamRenderCode(qw422016, p.Job.Output.String)
		//line template/job/show.qtpl:75
		qw422016.N().S(` `)
		//line template/job/show.qtpl:76
	} else {
		//line template/job/show.qtpl:76
		qw422016.N().S(` <div class="panel-message muted">No job output has been produced.</div> `)
		//line template/job/show.qtpl:78
	}
	//line template/job/show.qtpl:78
	qw422016.N().S(` </div> <div class="panel"> <div class="panel-header"><h3>Artifacts</h3></div> `)
	//line template/job/show.qtpl:82
	if len(p.Job.Artifacts) > 0 {
		//line template/job/show.qtpl:82
		qw422016.N().S(` `)
		//line template/job/show.qtpl:83
		artifact.StreamRenderTable(qw422016, p.Job.Artifacts)
		//line template/job/show.qtpl:83
		qw422016.N().S(` `)
		//line template/job/show.qtpl:84
	} else {
		//line template/job/show.qtpl:84
		qw422016.N().S(` <div class="panel-message muted">No artifacts have been collected from this job.</div> `)
		//line template/job/show.qtpl:86
	}
	//line template/job/show.qtpl:86
	qw422016.N().S(` </div> </div> </div> `)
//line template/job/show.qtpl:90
}

//line template/job/show.qtpl:90
func (p *ShowPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/job/show.qtpl:90
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/job/show.qtpl:90
	p.StreamBody(qw422016)
	//line template/job/show.qtpl:90
	qt422016.ReleaseWriter(qw422016)
//line template/job/show.qtpl:90
}

//line template/job/show.qtpl:90
func (p *ShowPage) Body() string {
	//line template/job/show.qtpl:90
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/job/show.qtpl:90
	p.WriteBody(qb422016)
	//line template/job/show.qtpl:90
	qs422016 := string(qb422016.B)
	//line template/job/show.qtpl:90
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/job/show.qtpl:90
	return qs422016
//line template/job/show.qtpl:90
}

//line template/job/show.qtpl:92
func (p *ShowPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/job/show.qtpl:92
	qw422016.N().S(` <a href="`)
	//line template/job/show.qtpl:93
	qw422016.E().S(p.Job.Build.UIEndpoint())
	//line template/job/show.qtpl:93
	qw422016.N().S(`" class="back">`)
	//line template/job/show.qtpl:93
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
	//line template/job/show.qtpl:93
	qw422016.N().S(`</a> `)
	//line template/job/show.qtpl:94
	if !p.Job.Build.Namespace.IsZero() {
		//line template/job/show.qtpl:94
		qw422016.N().S(` <a href="`)
		//line template/job/show.qtpl:95
		qw422016.E().S(p.Job.Build.Namespace.UIEndpoint())
		//line template/job/show.qtpl:95
		qw422016.N().S(`">`)
		//line template/job/show.qtpl:95
		qw422016.E().S(p.Job.Build.Namespace.Name)
		//line template/job/show.qtpl:95
		qw422016.N().S(`</a> / `)
		//line template/job/show.qtpl:96
	}
	//line template/job/show.qtpl:96
	qw422016.N().S(` Build #`)
	//line template/job/show.qtpl:97
	qw422016.E().V(p.Job.BuildID)
	//line template/job/show.qtpl:97
	qw422016.N().S(` / `)
	//line template/job/show.qtpl:97
	qw422016.E().S(p.Job.Stage.Name)
	//line template/job/show.qtpl:97
	qw422016.N().S(` - `)
	//line template/job/show.qtpl:97
	qw422016.E().S(p.Job.Name)
	//line template/job/show.qtpl:97
	qw422016.N().S(` `)
//line template/job/show.qtpl:98
}

//line template/job/show.qtpl:98
func (p *ShowPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/job/show.qtpl:98
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/job/show.qtpl:98
	p.StreamHeader(qw422016)
	//line template/job/show.qtpl:98
	qt422016.ReleaseWriter(qw422016)
//line template/job/show.qtpl:98
}

//line template/job/show.qtpl:98
func (p *ShowPage) Header() string {
	//line template/job/show.qtpl:98
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/job/show.qtpl:98
	p.WriteHeader(qb422016)
	//line template/job/show.qtpl:98
	qs422016 := string(qb422016.B)
	//line template/job/show.qtpl:98
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/job/show.qtpl:98
	return qs422016
//line template/job/show.qtpl:98
}

//line template/job/show.qtpl:100
func (p *ShowPage) StreamActions(qw422016 *qt422016.Writer) {
//line template/job/show.qtpl:100
}

//line template/job/show.qtpl:100
func (p *ShowPage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/job/show.qtpl:100
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/job/show.qtpl:100
	p.StreamActions(qw422016)
	//line template/job/show.qtpl:100
	qt422016.ReleaseWriter(qw422016)
//line template/job/show.qtpl:100
}

//line template/job/show.qtpl:100
func (p *ShowPage) Actions() string {
	//line template/job/show.qtpl:100
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/job/show.qtpl:100
	p.WriteActions(qb422016)
	//line template/job/show.qtpl:100
	qs422016 := string(qb422016.B)
	//line template/job/show.qtpl:100
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/job/show.qtpl:100
	return qs422016
//line template/job/show.qtpl:100
}

//line template/job/show.qtpl:101
func (p *ShowPage) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/job/show.qtpl:101
}

//line template/job/show.qtpl:101
func (p *ShowPage) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/job/show.qtpl:101
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/job/show.qtpl:101
	p.StreamNavigation(qw422016)
	//line template/job/show.qtpl:101
	qt422016.ReleaseWriter(qw422016)
//line template/job/show.qtpl:101
}

//line template/job/show.qtpl:101
func (p *ShowPage) Navigation() string {
	//line template/job/show.qtpl:101
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/job/show.qtpl:101
	p.WriteNavigation(qb422016)
	//line template/job/show.qtpl:101
	qs422016 := string(qb422016.B)
	//line template/job/show.qtpl:101
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/job/show.qtpl:101
	return qs422016
//line template/job/show.qtpl:101
}
