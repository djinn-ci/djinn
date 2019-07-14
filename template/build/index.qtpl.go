// This file is automatically generated by qtc from "index.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/build/index.qtpl:2
package build

//line template/build/index.qtpl:2
import (
	htmltemplate "html/template"

	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/template/artifact"
)

//line template/build/index.qtpl:11
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/build/index.qtpl:11
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/build/index.qtpl:12
type IndexPage struct {
	template.Page

	Builds []*model.Build
	Search string
	Status string
	Tag    string
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

type TagIndexPage struct {
	ShowPage

	CSRF htmltemplate.HTML
	Tags []*model.Tag
}

type VariableIndexPage struct {
	ShowPage

	Variables []*model.BuildVariable
}

//line template/build/index.qtpl:49
func (p *IndexPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:49
	qw422016.N().S(` Builds - Thrall `)
//line template/build/index.qtpl:51
}

//line template/build/index.qtpl:51
func (p *IndexPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:51
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:51
	p.StreamTitle(qw422016)
	//line template/build/index.qtpl:51
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:51
}

//line template/build/index.qtpl:51
func (p *IndexPage) Title() string {
	//line template/build/index.qtpl:51
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:51
	p.WriteTitle(qb422016)
	//line template/build/index.qtpl:51
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:51
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:51
	return qs422016
//line template/build/index.qtpl:51
}

//line template/build/index.qtpl:53
func (p *ObjectIndexPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:53
	qw422016.N().S(` Build #`)
	//line template/build/index.qtpl:54
	qw422016.E().V(p.Build.ID)
	//line template/build/index.qtpl:54
	qw422016.N().S(` Objects - Thrall `)
//line template/build/index.qtpl:55
}

//line template/build/index.qtpl:55
func (p *ObjectIndexPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:55
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:55
	p.StreamTitle(qw422016)
	//line template/build/index.qtpl:55
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:55
}

//line template/build/index.qtpl:55
func (p *ObjectIndexPage) Title() string {
	//line template/build/index.qtpl:55
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:55
	p.WriteTitle(qb422016)
	//line template/build/index.qtpl:55
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:55
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:55
	return qs422016
//line template/build/index.qtpl:55
}

//line template/build/index.qtpl:57
func (p *VariableIndexPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:57
	qw422016.N().S(` Build #`)
	//line template/build/index.qtpl:58
	qw422016.E().V(p.Build.ID)
	//line template/build/index.qtpl:58
	qw422016.N().S(` Variables - Thrall `)
//line template/build/index.qtpl:59
}

//line template/build/index.qtpl:59
func (p *VariableIndexPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:59
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:59
	p.StreamTitle(qw422016)
	//line template/build/index.qtpl:59
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:59
}

//line template/build/index.qtpl:59
func (p *VariableIndexPage) Title() string {
	//line template/build/index.qtpl:59
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:59
	p.WriteTitle(qb422016)
	//line template/build/index.qtpl:59
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:59
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:59
	return qs422016
//line template/build/index.qtpl:59
}

//line template/build/index.qtpl:61
func (p *IndexPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:61
	qw422016.N().S(` <div class="panel">`)
	//line template/build/index.qtpl:62
	StreamRenderIndex(qw422016, p.Builds, p.URI, p.Status, p.Search)
	//line template/build/index.qtpl:62
	qw422016.N().S(`</div> `)
//line template/build/index.qtpl:63
}

//line template/build/index.qtpl:63
func (p *IndexPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:63
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:63
	p.StreamBody(qw422016)
	//line template/build/index.qtpl:63
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:63
}

//line template/build/index.qtpl:63
func (p *IndexPage) Body() string {
	//line template/build/index.qtpl:63
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:63
	p.WriteBody(qb422016)
	//line template/build/index.qtpl:63
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:63
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:63
	return qs422016
//line template/build/index.qtpl:63
}

//line template/build/index.qtpl:65
func (p *ArtifactIndexPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:65
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/index.qtpl:67
	if len(p.Artifacts) > 0 {
		//line template/build/index.qtpl:67
		qw422016.N().S(` <div class="panel-header">`)
		//line template/build/index.qtpl:68
		template.StreamRenderSearch(qw422016, p.URI, p.Search, "Find an artifact...")
		//line template/build/index.qtpl:68
		qw422016.N().S(`</div> `)
		//line template/build/index.qtpl:69
		artifact.StreamRenderTable(qw422016, p.Artifacts)
		//line template/build/index.qtpl:69
		qw422016.N().S(` `)
		//line template/build/index.qtpl:70
	} else {
		//line template/build/index.qtpl:70
		qw422016.N().S(` <div class="panel-message muted">No artifacts have been collected from this build.</div> `)
		//line template/build/index.qtpl:72
	}
	//line template/build/index.qtpl:72
	qw422016.N().S(` </div> `)
//line template/build/index.qtpl:74
}

//line template/build/index.qtpl:74
func (p *ArtifactIndexPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:74
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:74
	p.StreamBody(qw422016)
	//line template/build/index.qtpl:74
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:74
}

//line template/build/index.qtpl:74
func (p *ArtifactIndexPage) Body() string {
	//line template/build/index.qtpl:74
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:74
	p.WriteBody(qb422016)
	//line template/build/index.qtpl:74
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:74
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:74
	return qs422016
//line template/build/index.qtpl:74
}

//line template/build/index.qtpl:76
func (p *ObjectIndexPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:76
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/index.qtpl:78
	if len(p.Objects) == 0 {
		//line template/build/index.qtpl:78
		qw422016.N().S(` <div class="panel-message muted">No objects have been placed for this build.</div> `)
		//line template/build/index.qtpl:80
	} else {
		//line template/build/index.qtpl:80
		qw422016.N().S(` `)
		//line template/build/index.qtpl:81
		StreamRenderObjectTable(qw422016, p.Objects)
		//line template/build/index.qtpl:81
		qw422016.N().S(` `)
		//line template/build/index.qtpl:82
	}
	//line template/build/index.qtpl:82
	qw422016.N().S(` </div> `)
//line template/build/index.qtpl:84
}

//line template/build/index.qtpl:84
func (p *ObjectIndexPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:84
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:84
	p.StreamBody(qw422016)
	//line template/build/index.qtpl:84
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:84
}

//line template/build/index.qtpl:84
func (p *ObjectIndexPage) Body() string {
	//line template/build/index.qtpl:84
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:84
	p.WriteBody(qb422016)
	//line template/build/index.qtpl:84
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:84
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:84
	return qs422016
//line template/build/index.qtpl:84
}

//line template/build/index.qtpl:86
func (p *TagIndexPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:86
	qw422016.N().S(` <div class="panel"> <div class="panel-header panel-body"> <form method="POST" action="`)
	//line template/build/index.qtpl:89
	qw422016.E().S(p.Build.UIEndpoint("tags"))
	//line template/build/index.qtpl:89
	qw422016.N().S(`"> `)
	//line template/build/index.qtpl:90
	qw422016.N().S(string(p.CSRF))
	//line template/build/index.qtpl:90
	qw422016.N().S(` <div class="form-field form-field-inline"> <input type="text" class="form-text" name="tags" placeholder="Tag this build..." autocomplete="off"/> <button type="submit" class="btn btn-primary">Tag</button> </div> </form> </div> `)
	//line template/build/index.qtpl:97
	if len(p.Tags) == 0 {
		//line template/build/index.qtpl:97
		qw422016.N().S(` <div class="panel-message muted">No tags have been set for this build.</div> `)
		//line template/build/index.qtpl:99
	} else {
		//line template/build/index.qtpl:99
		qw422016.N().S(` `)
		//line template/build/index.qtpl:100
		StreamRenderTagTable(qw422016, p.Tags, p.CSRF)
		//line template/build/index.qtpl:100
		qw422016.N().S(` `)
		//line template/build/index.qtpl:101
	}
	//line template/build/index.qtpl:101
	qw422016.N().S(` </div> `)
//line template/build/index.qtpl:103
}

//line template/build/index.qtpl:103
func (p *TagIndexPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:103
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:103
	p.StreamBody(qw422016)
	//line template/build/index.qtpl:103
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:103
}

//line template/build/index.qtpl:103
func (p *TagIndexPage) Body() string {
	//line template/build/index.qtpl:103
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:103
	p.WriteBody(qb422016)
	//line template/build/index.qtpl:103
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:103
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:103
	return qs422016
//line template/build/index.qtpl:103
}

//line template/build/index.qtpl:105
func (p *VariableIndexPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:105
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/index.qtpl:107
	if len(p.Variables) == 0 {
		//line template/build/index.qtpl:107
		qw422016.N().S(` <div class="panel-message muted">No variables have been set for this build.</div> `)
		//line template/build/index.qtpl:109
	} else {
		//line template/build/index.qtpl:109
		qw422016.N().S(` `)
		//line template/build/index.qtpl:110
		StreamRenderVariableTable(qw422016, p.Variables)
		//line template/build/index.qtpl:110
		qw422016.N().S(` `)
		//line template/build/index.qtpl:111
	}
	//line template/build/index.qtpl:111
	qw422016.N().S(` </div> `)
//line template/build/index.qtpl:113
}

//line template/build/index.qtpl:113
func (p *VariableIndexPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:113
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:113
	p.StreamBody(qw422016)
	//line template/build/index.qtpl:113
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:113
}

//line template/build/index.qtpl:113
func (p *VariableIndexPage) Body() string {
	//line template/build/index.qtpl:113
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:113
	p.WriteBody(qb422016)
	//line template/build/index.qtpl:113
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:113
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:113
	return qs422016
//line template/build/index.qtpl:113
}

//line template/build/index.qtpl:115
func (p *IndexPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:115
	qw422016.N().S(` Builds `)
	//line template/build/index.qtpl:117
	if p.Tag != "" {
		//line template/build/index.qtpl:117
		qw422016.N().S(` <span class="pill pill-light">`)
		//line template/build/index.qtpl:118
		qw422016.E().S(p.Tag)
		//line template/build/index.qtpl:118
		qw422016.N().S(`<a href="`)
		//line template/build/index.qtpl:118
		qw422016.E().S(p.URI)
		//line template/build/index.qtpl:118
		qw422016.N().S(`">`)
		//line template/build/index.qtpl:118
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M18.984 6.422l-5.578 5.578 5.578 5.578-1.406 1.406-5.578-5.578-5.578 5.578-1.406-1.406 5.578-5.578-5.578-5.578 1.406-1.406 5.578 5.578 5.578-5.578z"></path>
</svg>
`)
		//line template/build/index.qtpl:118
		qw422016.N().S(`</a></span> `)
		//line template/build/index.qtpl:119
	}
	//line template/build/index.qtpl:119
	qw422016.N().S(` `)
//line template/build/index.qtpl:120
}

//line template/build/index.qtpl:120
func (p *IndexPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:120
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:120
	p.StreamHeader(qw422016)
	//line template/build/index.qtpl:120
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:120
}

//line template/build/index.qtpl:120
func (p *IndexPage) Header() string {
	//line template/build/index.qtpl:120
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:120
	p.WriteHeader(qb422016)
	//line template/build/index.qtpl:120
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:120
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:120
	return qs422016
//line template/build/index.qtpl:120
}

//line template/build/index.qtpl:122
func (p *IndexPage) StreamActions(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:122
	qw422016.N().S(` <li><a href="/builds/create" class="btn btn-primary">Submit</a></li> `)
//line template/build/index.qtpl:124
}

//line template/build/index.qtpl:124
func (p *IndexPage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:124
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:124
	p.StreamActions(qw422016)
	//line template/build/index.qtpl:124
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:124
}

//line template/build/index.qtpl:124
func (p *IndexPage) Actions() string {
	//line template/build/index.qtpl:124
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:124
	p.WriteActions(qb422016)
	//line template/build/index.qtpl:124
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:124
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:124
	return qs422016
//line template/build/index.qtpl:124
}

//line template/build/index.qtpl:126
func (p *IndexPage) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/build/index.qtpl:126
}

//line template/build/index.qtpl:126
func (p *IndexPage) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:126
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:126
	p.StreamNavigation(qw422016)
	//line template/build/index.qtpl:126
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:126
}

//line template/build/index.qtpl:126
func (p *IndexPage) Navigation() string {
	//line template/build/index.qtpl:126
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:126
	p.WriteNavigation(qb422016)
	//line template/build/index.qtpl:126
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:126
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:126
	return qs422016
//line template/build/index.qtpl:126
}
