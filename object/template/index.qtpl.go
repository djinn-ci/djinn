// Code generated by qtc from "index.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line object/template/index.qtpl:2
package template

//line object/template/index.qtpl:2
import (
	htmltemplate "html/template"

	"djinn-ci.com/database"
	"djinn-ci.com/object"
	"djinn-ci.com/template"
)

//line object/template/index.qtpl:11
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line object/template/index.qtpl:11
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line object/template/index.qtpl:12
type Index struct {
	template.BasePage

	CSRF      htmltemplate.HTML
	Paginator database.Paginator
	Objects   []*object.Object
	Search    string
}

//line object/template/index.qtpl:23
func (p *Index) StreamTitle(qw422016 *qt422016.Writer) {
//line object/template/index.qtpl:23
	qw422016.N().S(` Objects - Djinn CI `)
//line object/template/index.qtpl:25
}

//line object/template/index.qtpl:25
func (p *Index) WriteTitle(qq422016 qtio422016.Writer) {
//line object/template/index.qtpl:25
	qw422016 := qt422016.AcquireWriter(qq422016)
//line object/template/index.qtpl:25
	p.StreamTitle(qw422016)
//line object/template/index.qtpl:25
	qt422016.ReleaseWriter(qw422016)
//line object/template/index.qtpl:25
}

//line object/template/index.qtpl:25
func (p *Index) Title() string {
//line object/template/index.qtpl:25
	qb422016 := qt422016.AcquireByteBuffer()
//line object/template/index.qtpl:25
	p.WriteTitle(qb422016)
//line object/template/index.qtpl:25
	qs422016 := string(qb422016.B)
//line object/template/index.qtpl:25
	qt422016.ReleaseByteBuffer(qb422016)
//line object/template/index.qtpl:25
	return qs422016
//line object/template/index.qtpl:25
}

//line object/template/index.qtpl:27
func (p *Index) StreamBody(qw422016 *qt422016.Writer) {
//line object/template/index.qtpl:27
	qw422016.N().S(` <div class="panel"> `)
//line object/template/index.qtpl:29
	if len(p.Objects) == 0 && p.Search == "" {
//line object/template/index.qtpl:29
		qw422016.N().S(` <div class="panel-message muted">Objects are files that can be used in build environments.</div> `)
//line object/template/index.qtpl:31
	} else {
//line object/template/index.qtpl:31
		qw422016.N().S(` <div class="panel-header">`)
//line object/template/index.qtpl:32
		template.StreamRenderSearch(qw422016, p.URL.Path, p.Search, "Find an object...")
//line object/template/index.qtpl:32
		qw422016.N().S(`</div> `)
//line object/template/index.qtpl:33
		if len(p.Objects) == 0 && p.Search != "" {
//line object/template/index.qtpl:33
			qw422016.N().S(` <div class="panel-message muted">No results found.</div> `)
//line object/template/index.qtpl:35
		} else {
//line object/template/index.qtpl:35
			qw422016.N().S(` <table class="table"> <thead> <tr> <th>NAME</th> <th>TYPE</th> <th>SIZE</th> <th>NAMESPACE</th> <th></th> <th></th> </tr> </thead> <tbody> `)
//line object/template/index.qtpl:48
			for _, o := range p.Objects {
//line object/template/index.qtpl:48
				qw422016.N().S(` <tr> <td><a href="`)
//line object/template/index.qtpl:50
				qw422016.E().S(o.Endpoint())
//line object/template/index.qtpl:50
				qw422016.N().S(`">`)
//line object/template/index.qtpl:50
				qw422016.E().S(o.Name)
//line object/template/index.qtpl:50
				qw422016.N().S(`</a></td> <td><span class="code">`)
//line object/template/index.qtpl:51
				qw422016.E().S(o.Type)
//line object/template/index.qtpl:51
				qw422016.N().S(`</span></td> <td>`)
//line object/template/index.qtpl:52
				qw422016.E().S(template.RenderSize(o.Size))
//line object/template/index.qtpl:52
				qw422016.N().S(`</td> <td> `)
//line object/template/index.qtpl:54
				if o.Namespace != nil {
//line object/template/index.qtpl:54
					qw422016.N().S(` <a href="`)
//line object/template/index.qtpl:55
					qw422016.E().S(o.Namespace.Endpoint())
//line object/template/index.qtpl:55
					qw422016.N().S(`">`)
//line object/template/index.qtpl:55
					qw422016.E().S(o.Namespace.Path)
//line object/template/index.qtpl:55
					qw422016.N().S(`</a> `)
//line object/template/index.qtpl:56
				} else {
//line object/template/index.qtpl:56
					qw422016.N().S(` <span class="muted">--</span> `)
//line object/template/index.qtpl:58
				}
//line object/template/index.qtpl:58
				qw422016.N().S(` </td> <td class="align-right"> `)
//line object/template/index.qtpl:61
				if p.User.ID != o.UserID {
//line object/template/index.qtpl:61
					qw422016.N().S(` <span class="muted">`)
//line object/template/index.qtpl:62
					qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M15.984 12.984c2.344 0 7.031 1.172 7.031 3.516v2.484h-6v-2.484c0-1.5-0.797-2.625-1.969-3.469 0.328-0.047 0.656-0.047 0.938-0.047zM8.016 12.984c2.344 0 6.984 1.172 6.984 3.516v2.484h-14.016v-2.484c0-2.344 4.688-3.516 7.031-3.516zM8.016 11.016c-1.641 0-3-1.359-3-3s1.359-3 3-3 2.953 1.359 2.953 3-1.313 3-2.953 3zM15.984 11.016c-1.641 0-3-1.359-3-3s1.359-3 3-3 3 1.359 3 3-1.359 3-3 3z"></path>
</svg>
`)
//line object/template/index.qtpl:62
					qw422016.N().S(`</span> `)
//line object/template/index.qtpl:63
				}
//line object/template/index.qtpl:63
				qw422016.N().S(` </td> <td class="align-right"> `)
//line object/template/index.qtpl:66
				if p.User.ID == o.UserID || o.Namespace != nil && o.Namespace.UserID == p.User.ID {
//line object/template/index.qtpl:66
					qw422016.N().S(` <form method="POST" action="`)
//line object/template/index.qtpl:67
					qw422016.E().S(o.Endpoint())
//line object/template/index.qtpl:67
					qw422016.N().S(`"> `)
//line object/template/index.qtpl:68
					qw422016.N().V(p.CSRF)
//line object/template/index.qtpl:68
					qw422016.N().S(` <input type="hidden" name="_method" value="DELETE"/> <button type="submit" class="btn btn-danger">Delete</button> </form> `)
//line object/template/index.qtpl:72
				}
//line object/template/index.qtpl:72
				qw422016.N().S(` </td> </tr> `)
//line object/template/index.qtpl:75
			}
//line object/template/index.qtpl:75
			qw422016.N().S(` </tbody> </table> `)
//line object/template/index.qtpl:78
		}
//line object/template/index.qtpl:78
		qw422016.N().S(` `)
//line object/template/index.qtpl:79
	}
//line object/template/index.qtpl:79
	qw422016.N().S(` </div> `)
//line object/template/index.qtpl:81
	template.StreamRenderPaginator(qw422016, p.URL, p.Paginator)
//line object/template/index.qtpl:81
	qw422016.N().S(` `)
//line object/template/index.qtpl:82
}

//line object/template/index.qtpl:82
func (p *Index) WriteBody(qq422016 qtio422016.Writer) {
//line object/template/index.qtpl:82
	qw422016 := qt422016.AcquireWriter(qq422016)
//line object/template/index.qtpl:82
	p.StreamBody(qw422016)
//line object/template/index.qtpl:82
	qt422016.ReleaseWriter(qw422016)
//line object/template/index.qtpl:82
}

//line object/template/index.qtpl:82
func (p *Index) Body() string {
//line object/template/index.qtpl:82
	qb422016 := qt422016.AcquireByteBuffer()
//line object/template/index.qtpl:82
	p.WriteBody(qb422016)
//line object/template/index.qtpl:82
	qs422016 := string(qb422016.B)
//line object/template/index.qtpl:82
	qt422016.ReleaseByteBuffer(qb422016)
//line object/template/index.qtpl:82
	return qs422016
//line object/template/index.qtpl:82
}

//line object/template/index.qtpl:84
func (p *Index) StreamSection(qw422016 *qt422016.Writer) {
//line object/template/index.qtpl:84
	qw422016.N().S(` `)
//line object/template/index.qtpl:85
	p.StreamBody(qw422016)
//line object/template/index.qtpl:85
	qw422016.N().S(` `)
//line object/template/index.qtpl:86
}

//line object/template/index.qtpl:86
func (p *Index) WriteSection(qq422016 qtio422016.Writer) {
//line object/template/index.qtpl:86
	qw422016 := qt422016.AcquireWriter(qq422016)
//line object/template/index.qtpl:86
	p.StreamSection(qw422016)
//line object/template/index.qtpl:86
	qt422016.ReleaseWriter(qw422016)
//line object/template/index.qtpl:86
}

//line object/template/index.qtpl:86
func (p *Index) Section() string {
//line object/template/index.qtpl:86
	qb422016 := qt422016.AcquireByteBuffer()
//line object/template/index.qtpl:86
	p.WriteSection(qb422016)
//line object/template/index.qtpl:86
	qs422016 := string(qb422016.B)
//line object/template/index.qtpl:86
	qt422016.ReleaseByteBuffer(qb422016)
//line object/template/index.qtpl:86
	return qs422016
//line object/template/index.qtpl:86
}

//line object/template/index.qtpl:88
func (p *Index) StreamHeader(qw422016 *qt422016.Writer) {
//line object/template/index.qtpl:88
	qw422016.N().S(` Objects `)
//line object/template/index.qtpl:90
}

//line object/template/index.qtpl:90
func (p *Index) WriteHeader(qq422016 qtio422016.Writer) {
//line object/template/index.qtpl:90
	qw422016 := qt422016.AcquireWriter(qq422016)
//line object/template/index.qtpl:90
	p.StreamHeader(qw422016)
//line object/template/index.qtpl:90
	qt422016.ReleaseWriter(qw422016)
//line object/template/index.qtpl:90
}

//line object/template/index.qtpl:90
func (p *Index) Header() string {
//line object/template/index.qtpl:90
	qb422016 := qt422016.AcquireByteBuffer()
//line object/template/index.qtpl:90
	p.WriteHeader(qb422016)
//line object/template/index.qtpl:90
	qs422016 := string(qb422016.B)
//line object/template/index.qtpl:90
	qt422016.ReleaseByteBuffer(qb422016)
//line object/template/index.qtpl:90
	return qs422016
//line object/template/index.qtpl:90
}

//line object/template/index.qtpl:92
func (p *Index) StreamActions(qw422016 *qt422016.Writer) {
//line object/template/index.qtpl:92
	qw422016.N().S(` `)
//line object/template/index.qtpl:93
	if _, ok := p.User.Permissions["object:write"]; ok {
//line object/template/index.qtpl:93
		qw422016.N().S(` <li><a href="/objects/create" class="btn btn-primary">Create</a></li> `)
//line object/template/index.qtpl:95
	}
//line object/template/index.qtpl:95
	qw422016.N().S(` `)
//line object/template/index.qtpl:96
}

//line object/template/index.qtpl:96
func (p *Index) WriteActions(qq422016 qtio422016.Writer) {
//line object/template/index.qtpl:96
	qw422016 := qt422016.AcquireWriter(qq422016)
//line object/template/index.qtpl:96
	p.StreamActions(qw422016)
//line object/template/index.qtpl:96
	qt422016.ReleaseWriter(qw422016)
//line object/template/index.qtpl:96
}

//line object/template/index.qtpl:96
func (p *Index) Actions() string {
//line object/template/index.qtpl:96
	qb422016 := qt422016.AcquireByteBuffer()
//line object/template/index.qtpl:96
	p.WriteActions(qb422016)
//line object/template/index.qtpl:96
	qs422016 := string(qb422016.B)
//line object/template/index.qtpl:96
	qt422016.ReleaseByteBuffer(qb422016)
//line object/template/index.qtpl:96
	return qs422016
//line object/template/index.qtpl:96
}

//line object/template/index.qtpl:98
func (p *Index) StreamNavigation(qw422016 *qt422016.Writer) {
//line object/template/index.qtpl:98
}

//line object/template/index.qtpl:98
func (p *Index) WriteNavigation(qq422016 qtio422016.Writer) {
//line object/template/index.qtpl:98
	qw422016 := qt422016.AcquireWriter(qq422016)
//line object/template/index.qtpl:98
	p.StreamNavigation(qw422016)
//line object/template/index.qtpl:98
	qt422016.ReleaseWriter(qw422016)
//line object/template/index.qtpl:98
}

//line object/template/index.qtpl:98
func (p *Index) Navigation() string {
//line object/template/index.qtpl:98
	qb422016 := qt422016.AcquireByteBuffer()
//line object/template/index.qtpl:98
	p.WriteNavigation(qb422016)
//line object/template/index.qtpl:98
	qs422016 := string(qb422016.B)
//line object/template/index.qtpl:98
	qt422016.ReleaseByteBuffer(qb422016)
//line object/template/index.qtpl:98
	return qs422016
//line object/template/index.qtpl:98
}
