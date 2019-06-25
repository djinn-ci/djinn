// This file is automatically generated by qtc from "index.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/build/index.qtpl:2
package build

//line template/build/index.qtpl:2
import (
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/artifact"
)

//line template/build/index.qtpl:9
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/build/index.qtpl:9
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/build/index.qtpl:10
type IndexPage struct {
	template.Page

	Builds []*model.Build
	Status string
}

type ArtifactIndexPage struct {
	ShowPage

	Search    string
	Artifacts []*model.Artifact
}

type ObjectIndexPage struct {
	ShowPage

	Objects []*model.BuildObject
}

type VariableIndexPage struct {
	ShowPage

	Variables []*model.BuildVariable
}

//line template/build/index.qtpl:38
func (p *IndexPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:38
	qw422016.N().S(` Builds - Thrall `)
//line template/build/index.qtpl:40
}

//line template/build/index.qtpl:40
func (p *IndexPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:40
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:40
	p.StreamTitle(qw422016)
	//line template/build/index.qtpl:40
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:40
}

//line template/build/index.qtpl:40
func (p *IndexPage) Title() string {
	//line template/build/index.qtpl:40
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:40
	p.WriteTitle(qb422016)
	//line template/build/index.qtpl:40
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:40
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:40
	return qs422016
//line template/build/index.qtpl:40
}

//line template/build/index.qtpl:42
func (p *ObjectIndexPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:42
	qw422016.N().S(` Build #`)
	//line template/build/index.qtpl:43
	qw422016.E().V(p.Build.ID)
	//line template/build/index.qtpl:43
	qw422016.N().S(` Objects - Thrall `)
//line template/build/index.qtpl:44
}

//line template/build/index.qtpl:44
func (p *ObjectIndexPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:44
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:44
	p.StreamTitle(qw422016)
	//line template/build/index.qtpl:44
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:44
}

//line template/build/index.qtpl:44
func (p *ObjectIndexPage) Title() string {
	//line template/build/index.qtpl:44
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:44
	p.WriteTitle(qb422016)
	//line template/build/index.qtpl:44
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:44
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:44
	return qs422016
//line template/build/index.qtpl:44
}

//line template/build/index.qtpl:46
func (p *VariableIndexPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:46
	qw422016.N().S(` Build #`)
	//line template/build/index.qtpl:47
	qw422016.E().V(p.Build.ID)
	//line template/build/index.qtpl:47
	qw422016.N().S(` Variables - Thrall `)
//line template/build/index.qtpl:48
}

//line template/build/index.qtpl:48
func (p *VariableIndexPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:48
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:48
	p.StreamTitle(qw422016)
	//line template/build/index.qtpl:48
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:48
}

//line template/build/index.qtpl:48
func (p *VariableIndexPage) Title() string {
	//line template/build/index.qtpl:48
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:48
	p.WriteTitle(qb422016)
	//line template/build/index.qtpl:48
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:48
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:48
	return qs422016
//line template/build/index.qtpl:48
}

//line template/build/index.qtpl:50
func (p *IndexPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:50
	qw422016.N().S(` <div class="panel"> <div class="panel-header"> `)
	//line template/build/index.qtpl:53
	StreamRenderStatusNav(qw422016, p.URI, p.Status)
	//line template/build/index.qtpl:53
	qw422016.N().S(` `)
	//line template/build/index.qtpl:54
	template.StreamRenderSearch(qw422016, p.URI, "", "Find a build...")
	//line template/build/index.qtpl:54
	qw422016.N().S(` </div> `)
	//line template/build/index.qtpl:56
	if len(p.Builds) == 0 {
		//line template/build/index.qtpl:56
		qw422016.N().S(` <div class="panel-message muted"> `)
		//line template/build/index.qtpl:58
		if p.Status == "" {
			//line template/build/index.qtpl:58
			qw422016.N().S(`No builds have been submitted yet.`)
			//line template/build/index.qtpl:58
		} else {
			//line template/build/index.qtpl:58
			qw422016.N().S(`No `)
			//line template/build/index.qtpl:58
			qw422016.E().S(p.Status)
			//line template/build/index.qtpl:58
			qw422016.N().S(` builds.`)
			//line template/build/index.qtpl:58
		}
		//line template/build/index.qtpl:58
		qw422016.N().S(` </div> `)
		//line template/build/index.qtpl:60
	} else {
		//line template/build/index.qtpl:60
		qw422016.N().S(` `)
		//line template/build/index.qtpl:61
		StreamRenderTable(qw422016, p.Builds)
		//line template/build/index.qtpl:61
		qw422016.N().S(` `)
		//line template/build/index.qtpl:62
	}
	//line template/build/index.qtpl:62
	qw422016.N().S(` </div> `)
//line template/build/index.qtpl:64
}

//line template/build/index.qtpl:64
func (p *IndexPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:64
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:64
	p.StreamBody(qw422016)
	//line template/build/index.qtpl:64
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:64
}

//line template/build/index.qtpl:64
func (p *IndexPage) Body() string {
	//line template/build/index.qtpl:64
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:64
	p.WriteBody(qb422016)
	//line template/build/index.qtpl:64
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:64
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:64
	return qs422016
//line template/build/index.qtpl:64
}

//line template/build/index.qtpl:66
func (p *ArtifactIndexPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:66
	qw422016.N().S(` <div class="panel"> <div class="panel-header">`)
	//line template/build/index.qtpl:68
	template.StreamRenderSearch(qw422016, p.URI, p.Search, "Find an artifact...")
	//line template/build/index.qtpl:68
	qw422016.N().S(`</div> `)
	//line template/build/index.qtpl:69
	if len(p.Artifacts) > 0 {
		//line template/build/index.qtpl:69
		qw422016.N().S(` `)
		//line template/build/index.qtpl:70
		artifact.StreamRenderTable(qw422016, p.Artifacts)
		//line template/build/index.qtpl:70
		qw422016.N().S(` `)
		//line template/build/index.qtpl:71
	} else {
		//line template/build/index.qtpl:71
		qw422016.N().S(` <div class="panel-message muted">No artifacts have been collected from this build.</div> `)
		//line template/build/index.qtpl:73
	}
	//line template/build/index.qtpl:73
	qw422016.N().S(` </div> `)
//line template/build/index.qtpl:75
}

//line template/build/index.qtpl:75
func (p *ArtifactIndexPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:75
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:75
	p.StreamBody(qw422016)
	//line template/build/index.qtpl:75
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:75
}

//line template/build/index.qtpl:75
func (p *ArtifactIndexPage) Body() string {
	//line template/build/index.qtpl:75
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:75
	p.WriteBody(qb422016)
	//line template/build/index.qtpl:75
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:75
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:75
	return qs422016
//line template/build/index.qtpl:75
}

//line template/build/index.qtpl:77
func (p *ObjectIndexPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:77
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/index.qtpl:79
	if len(p.Objects) == 0 {
		//line template/build/index.qtpl:79
		qw422016.N().S(` <div class="panel-message muted">No objects have been placed for this build.</div> `)
		//line template/build/index.qtpl:81
	} else {
		//line template/build/index.qtpl:81
		qw422016.N().S(` `)
		//line template/build/index.qtpl:82
		StreamRenderObjectTable(qw422016, p.Objects)
		//line template/build/index.qtpl:82
		qw422016.N().S(` `)
		//line template/build/index.qtpl:83
	}
	//line template/build/index.qtpl:83
	qw422016.N().S(` </div> `)
//line template/build/index.qtpl:85
}

//line template/build/index.qtpl:85
func (p *ObjectIndexPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:85
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:85
	p.StreamBody(qw422016)
	//line template/build/index.qtpl:85
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:85
}

//line template/build/index.qtpl:85
func (p *ObjectIndexPage) Body() string {
	//line template/build/index.qtpl:85
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:85
	p.WriteBody(qb422016)
	//line template/build/index.qtpl:85
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:85
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:85
	return qs422016
//line template/build/index.qtpl:85
}

//line template/build/index.qtpl:87
func (p *VariableIndexPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:87
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/index.qtpl:89
	if len(p.Variables) == 0 {
		//line template/build/index.qtpl:89
		qw422016.N().S(` <div class="panel-message muted">No variables have been set for this build.</div> `)
		//line template/build/index.qtpl:91
	} else {
		//line template/build/index.qtpl:91
		qw422016.N().S(` `)
		//line template/build/index.qtpl:92
		StreamRenderVariableTable(qw422016, p.Variables)
		//line template/build/index.qtpl:92
		qw422016.N().S(` `)
		//line template/build/index.qtpl:93
	}
	//line template/build/index.qtpl:93
	qw422016.N().S(` </div> `)
//line template/build/index.qtpl:95
}

//line template/build/index.qtpl:95
func (p *VariableIndexPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:95
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:95
	p.StreamBody(qw422016)
	//line template/build/index.qtpl:95
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:95
}

//line template/build/index.qtpl:95
func (p *VariableIndexPage) Body() string {
	//line template/build/index.qtpl:95
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:95
	p.WriteBody(qb422016)
	//line template/build/index.qtpl:95
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:95
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:95
	return qs422016
//line template/build/index.qtpl:95
}

//line template/build/index.qtpl:97
func (p *IndexPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:97
	qw422016.N().S(` Builds `)
//line template/build/index.qtpl:99
}

//line template/build/index.qtpl:99
func (p *IndexPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:99
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:99
	p.StreamHeader(qw422016)
	//line template/build/index.qtpl:99
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:99
}

//line template/build/index.qtpl:99
func (p *IndexPage) Header() string {
	//line template/build/index.qtpl:99
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:99
	p.WriteHeader(qb422016)
	//line template/build/index.qtpl:99
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:99
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:99
	return qs422016
//line template/build/index.qtpl:99
}

//line template/build/index.qtpl:101
func (p *IndexPage) StreamActions(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:101
	qw422016.N().S(` <li><a href="/builds/create" class="btn btn-primary">Submit</a></li> `)
//line template/build/index.qtpl:103
}

//line template/build/index.qtpl:103
func (p *IndexPage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:103
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:103
	p.StreamActions(qw422016)
	//line template/build/index.qtpl:103
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:103
}

//line template/build/index.qtpl:103
func (p *IndexPage) Actions() string {
	//line template/build/index.qtpl:103
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:103
	p.WriteActions(qb422016)
	//line template/build/index.qtpl:103
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:103
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:103
	return qs422016
//line template/build/index.qtpl:103
}

//line template/build/index.qtpl:105
func (p *IndexPage) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/build/index.qtpl:105
}

//line template/build/index.qtpl:105
func (p *IndexPage) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:105
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:105
	p.StreamNavigation(qw422016)
	//line template/build/index.qtpl:105
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:105
}

//line template/build/index.qtpl:105
func (p *IndexPage) Navigation() string {
	//line template/build/index.qtpl:105
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:105
	p.WriteNavigation(qb422016)
	//line template/build/index.qtpl:105
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:105
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:105
	return qs422016
//line template/build/index.qtpl:105
}
