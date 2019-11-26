// This file is automatically generated by qtc from "index.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/build/index.qtpl:2
package build

//line template/build/index.qtpl:2
import (
	"strings"

	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

//line template/build/index.qtpl:10
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/build/index.qtpl:10
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/build/index.qtpl:11
type IndexPage struct {
	template.BasePage

	Paginator model.Paginator
	Builds    []*model.Build
	Search    string
	Status    string
	Tag       string
}

//line template/build/index.qtpl:23
func (p *IndexPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:23
	qw422016.N().S(` Builds - Thrall `)
//line template/build/index.qtpl:25
}

//line template/build/index.qtpl:25
func (p *IndexPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:25
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:25
	p.StreamTitle(qw422016)
	//line template/build/index.qtpl:25
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:25
}

//line template/build/index.qtpl:25
func (p *IndexPage) Title() string {
	//line template/build/index.qtpl:25
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:25
	p.WriteTitle(qb422016)
	//line template/build/index.qtpl:25
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:25
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:25
	return qs422016
//line template/build/index.qtpl:25
}

//line template/build/index.qtpl:27
func (p *IndexPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:27
	qw422016.N().S(` <div class="panel"> `)
	//line template/build/index.qtpl:29
	if len(p.Builds) == 0 && p.Search == "" && p.Status == "" {
		//line template/build/index.qtpl:29
		qw422016.N().S(` <div class="panel-message muted">No builds have been submitted yet.</div> `)
		//line template/build/index.qtpl:31
	} else {
		//line template/build/index.qtpl:31
		qw422016.N().S(` <div class="panel-header"> `)
		//line template/build/index.qtpl:33
		StreamRenderStatusNav(qw422016, p.URL.Path, p.Status)
		//line template/build/index.qtpl:33
		qw422016.N().S(` `)
		//line template/build/index.qtpl:34
		template.StreamRenderSearch(qw422016, p.URL.Path, p.Search, "Find a build...")
		//line template/build/index.qtpl:34
		qw422016.N().S(` </div> `)
		//line template/build/index.qtpl:36
		if len(p.Builds) == 0 && p.Search != "" {
			//line template/build/index.qtpl:36
			qw422016.N().S(` <div class="panel-message muted">No results found.</div> `)
			//line template/build/index.qtpl:38
		} else if len(p.Builds) == 0 && p.Status != "" {
			//line template/build/index.qtpl:38
			qw422016.N().S(` <div class="panel-message muted">No `)
			//line template/build/index.qtpl:39
			qw422016.E().S(strings.Replace(p.Status, "_", " ", -1))
			//line template/build/index.qtpl:39
			qw422016.N().S(` builds.</div> `)
			//line template/build/index.qtpl:40
		} else {
			//line template/build/index.qtpl:40
			qw422016.N().S(` <table class="table"> <thead> <tr> <th>STATUS</th> <th>BUILD</th> <th>NAMESPACE</th> <th></th> <th></th> </tr> </thead> <tbody> `)
			//line template/build/index.qtpl:52
			for _, b := range p.Builds {
				//line template/build/index.qtpl:52
				qw422016.N().S(` <tr> <td>`)
				//line template/build/index.qtpl:54
				template.StreamRenderStatus(qw422016, b.Status)
				//line template/build/index.qtpl:54
				qw422016.N().S(`</td> <td><a href="`)
				//line template/build/index.qtpl:55
				qw422016.E().S(b.UIEndpoint())
				//line template/build/index.qtpl:55
				qw422016.N().S(`">#`)
				//line template/build/index.qtpl:55
				qw422016.E().V(b.ID)
				//line template/build/index.qtpl:55
				if b.Trigger.Comment != "" {
					//line template/build/index.qtpl:55
					qw422016.N().S(` - `)
					//line template/build/index.qtpl:55
					qw422016.E().S(b.Trigger.CommentTitle())
					//line template/build/index.qtpl:55
				}
				//line template/build/index.qtpl:55
				qw422016.N().S(`</a></td> <td> `)
				//line template/build/index.qtpl:57
				if b.Namespace != nil {
					//line template/build/index.qtpl:57
					qw422016.N().S(` <a href="`)
					//line template/build/index.qtpl:58
					qw422016.E().S(b.Namespace.UIEndpoint())
					//line template/build/index.qtpl:58
					qw422016.N().S(`">`)
					//line template/build/index.qtpl:58
					qw422016.E().S(b.Namespace.Path)
					//line template/build/index.qtpl:58
					qw422016.N().S(`</a> `)
					//line template/build/index.qtpl:59
				} else {
					//line template/build/index.qtpl:59
					qw422016.N().S(` <span class="muted">--</span> `)
					//line template/build/index.qtpl:61
				}
				//line template/build/index.qtpl:61
				qw422016.N().S(` </td> <td class="align-right"> `)
				//line template/build/index.qtpl:64
				for _, t := range b.Tags {
					//line template/build/index.qtpl:64
					qw422016.N().S(` <a class="pill pill-light" href="?tag=`)
					//line template/build/index.qtpl:65
					qw422016.E().S(t.Name)
					//line template/build/index.qtpl:65
					qw422016.N().S(`">`)
					//line template/build/index.qtpl:65
					qw422016.E().S(t.Name)
					//line template/build/index.qtpl:65
					qw422016.N().S(`</a> `)
					//line template/build/index.qtpl:66
				}
				//line template/build/index.qtpl:66
				qw422016.N().S(` </td> <td class="align-right"> `)
				//line template/build/index.qtpl:69
				if !b.FinishedAt.Valid || !b.StartedAt.Valid {
					//line template/build/index.qtpl:69
					qw422016.N().S(` <span class="muted">--</span> `)
					//line template/build/index.qtpl:71
				} else {
					//line template/build/index.qtpl:71
					qw422016.N().S(` `)
					//line template/build/index.qtpl:72
					qw422016.E().V(b.FinishedAt.Time.Sub(b.StartedAt.Time))
					//line template/build/index.qtpl:72
					qw422016.N().S(` `)
					//line template/build/index.qtpl:73
				}
				//line template/build/index.qtpl:73
				qw422016.N().S(` </td> </tr> `)
				//line template/build/index.qtpl:76
			}
			//line template/build/index.qtpl:76
			qw422016.N().S(` </tbody> </table> `)
			//line template/build/index.qtpl:79
		}
		//line template/build/index.qtpl:79
		qw422016.N().S(` `)
		//line template/build/index.qtpl:80
	}
	//line template/build/index.qtpl:80
	qw422016.N().S(` </div> `)
	//line template/build/index.qtpl:82
	template.StreamRenderPaginator(qw422016, p.URL.Path, p.Paginator)
	//line template/build/index.qtpl:82
	qw422016.N().S(` `)
//line template/build/index.qtpl:83
}

//line template/build/index.qtpl:83
func (p *IndexPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:83
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:83
	p.StreamBody(qw422016)
	//line template/build/index.qtpl:83
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:83
}

//line template/build/index.qtpl:83
func (p *IndexPage) Body() string {
	//line template/build/index.qtpl:83
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:83
	p.WriteBody(qb422016)
	//line template/build/index.qtpl:83
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:83
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:83
	return qs422016
//line template/build/index.qtpl:83
}

//line template/build/index.qtpl:85
func (p *IndexPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:85
	qw422016.N().S(` Builds `)
	//line template/build/index.qtpl:87
	if p.Tag != "" {
		//line template/build/index.qtpl:87
		qw422016.N().S(` <span class="pill pill-light">`)
		//line template/build/index.qtpl:88
		qw422016.E().S(p.Tag)
		//line template/build/index.qtpl:88
		qw422016.N().S(`<a href="`)
		//line template/build/index.qtpl:88
		qw422016.E().S(p.URL.Path)
		//line template/build/index.qtpl:88
		qw422016.N().S(`">`)
		//line template/build/index.qtpl:88
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M18.984 6.422l-5.578 5.578 5.578 5.578-1.406 1.406-5.578-5.578-5.578 5.578-1.406-1.406 5.578-5.578-5.578-5.578 1.406-1.406 5.578 5.578 5.578-5.578z"></path>
</svg>
`)
		//line template/build/index.qtpl:88
		qw422016.N().S(`</a></span> `)
		//line template/build/index.qtpl:89
	}
	//line template/build/index.qtpl:89
	qw422016.N().S(` `)
//line template/build/index.qtpl:90
}

//line template/build/index.qtpl:90
func (p *IndexPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:90
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:90
	p.StreamHeader(qw422016)
	//line template/build/index.qtpl:90
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:90
}

//line template/build/index.qtpl:90
func (p *IndexPage) Header() string {
	//line template/build/index.qtpl:90
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:90
	p.WriteHeader(qb422016)
	//line template/build/index.qtpl:90
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:90
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:90
	return qs422016
//line template/build/index.qtpl:90
}

//line template/build/index.qtpl:92
func (p *IndexPage) StreamActions(qw422016 *qt422016.Writer) {
	//line template/build/index.qtpl:92
	qw422016.N().S(` <li><a href="/builds/create" class="btn btn-primary">Submit</a></li> `)
//line template/build/index.qtpl:94
}

//line template/build/index.qtpl:94
func (p *IndexPage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:94
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:94
	p.StreamActions(qw422016)
	//line template/build/index.qtpl:94
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:94
}

//line template/build/index.qtpl:94
func (p *IndexPage) Actions() string {
	//line template/build/index.qtpl:94
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:94
	p.WriteActions(qb422016)
	//line template/build/index.qtpl:94
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:94
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:94
	return qs422016
//line template/build/index.qtpl:94
}

//line template/build/index.qtpl:96
func (p *IndexPage) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/build/index.qtpl:96
}

//line template/build/index.qtpl:96
func (p *IndexPage) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/build/index.qtpl:96
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/build/index.qtpl:96
	p.StreamNavigation(qw422016)
	//line template/build/index.qtpl:96
	qt422016.ReleaseWriter(qw422016)
//line template/build/index.qtpl:96
}

//line template/build/index.qtpl:96
func (p *IndexPage) Navigation() string {
	//line template/build/index.qtpl:96
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/build/index.qtpl:96
	p.WriteNavigation(qb422016)
	//line template/build/index.qtpl:96
	qs422016 := string(qb422016.B)
	//line template/build/index.qtpl:96
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/build/index.qtpl:96
	return qs422016
//line template/build/index.qtpl:96
}
