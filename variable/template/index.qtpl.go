// Code generated by qtc from "index.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line variable/template/index.qtpl:2
package template

//line variable/template/index.qtpl:2
import (
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
	"github.com/andrewpillar/thrall/variable"
)

//line variable/template/index.qtpl:9
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line variable/template/index.qtpl:9
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line variable/template/index.qtpl:10
type Index struct {
	template.BasePage

	CSRF      string
	Paginator model.Paginator
	Variables []*variable.Variable
	Search    string
}

//line variable/template/index.qtpl:21
func (p *Index) StreamTitle(qw422016 *qt422016.Writer) {
//line variable/template/index.qtpl:21
	qw422016.N().S(` Variables - Thrall `)
//line variable/template/index.qtpl:23
}

//line variable/template/index.qtpl:23
func (p *Index) WriteTitle(qq422016 qtio422016.Writer) {
//line variable/template/index.qtpl:23
	qw422016 := qt422016.AcquireWriter(qq422016)
//line variable/template/index.qtpl:23
	p.StreamTitle(qw422016)
//line variable/template/index.qtpl:23
	qt422016.ReleaseWriter(qw422016)
//line variable/template/index.qtpl:23
}

//line variable/template/index.qtpl:23
func (p *Index) Title() string {
//line variable/template/index.qtpl:23
	qb422016 := qt422016.AcquireByteBuffer()
//line variable/template/index.qtpl:23
	p.WriteTitle(qb422016)
//line variable/template/index.qtpl:23
	qs422016 := string(qb422016.B)
//line variable/template/index.qtpl:23
	qt422016.ReleaseByteBuffer(qb422016)
//line variable/template/index.qtpl:23
	return qs422016
//line variable/template/index.qtpl:23
}

//line variable/template/index.qtpl:25
func (p *Index) StreamHeader(qw422016 *qt422016.Writer) {
//line variable/template/index.qtpl:25
	qw422016.N().S(` Variables `)
//line variable/template/index.qtpl:27
}

//line variable/template/index.qtpl:27
func (p *Index) WriteHeader(qq422016 qtio422016.Writer) {
//line variable/template/index.qtpl:27
	qw422016 := qt422016.AcquireWriter(qq422016)
//line variable/template/index.qtpl:27
	p.StreamHeader(qw422016)
//line variable/template/index.qtpl:27
	qt422016.ReleaseWriter(qw422016)
//line variable/template/index.qtpl:27
}

//line variable/template/index.qtpl:27
func (p *Index) Header() string {
//line variable/template/index.qtpl:27
	qb422016 := qt422016.AcquireByteBuffer()
//line variable/template/index.qtpl:27
	p.WriteHeader(qb422016)
//line variable/template/index.qtpl:27
	qs422016 := string(qb422016.B)
//line variable/template/index.qtpl:27
	qt422016.ReleaseByteBuffer(qb422016)
//line variable/template/index.qtpl:27
	return qs422016
//line variable/template/index.qtpl:27
}

//line variable/template/index.qtpl:29
func (p *Index) StreamBody(qw422016 *qt422016.Writer) {
//line variable/template/index.qtpl:29
	qw422016.N().S(` <div class="panel"> `)
//line variable/template/index.qtpl:31
	if len(p.Variables) == 0 && p.Search == "" {
//line variable/template/index.qtpl:31
		qw422016.N().S(` <div class="panel-message muted">Set variables that can be used throughout build environments.</div> `)
//line variable/template/index.qtpl:33
	} else {
//line variable/template/index.qtpl:33
		qw422016.N().S(` <div class="panel-header">`)
//line variable/template/index.qtpl:34
		template.StreamRenderSearch(qw422016, p.URL.Path, p.Search, "Find a variable...")
//line variable/template/index.qtpl:34
		qw422016.N().S(`</div> `)
//line variable/template/index.qtpl:35
		if len(p.Variables) == 0 && p.Search != "" {
//line variable/template/index.qtpl:35
			qw422016.N().S(` <div class="panel-message muted">No results found.</div> `)
//line variable/template/index.qtpl:37
		} else {
//line variable/template/index.qtpl:37
			qw422016.N().S(` <table class="table"> <thead> <tr> <th>KEY</th> <th>VALUE</th> <th>NAMESPACE</th> <th></th> <th></th> </tr> </thead> <tbody> `)
//line variable/template/index.qtpl:49
			for _, v := range p.Variables {
//line variable/template/index.qtpl:49
				qw422016.N().S(` <tr> <td><span class="code">`)
//line variable/template/index.qtpl:51
				qw422016.E().S(v.Key)
//line variable/template/index.qtpl:51
				qw422016.N().S(`</span></td> <td><span class="code">`)
//line variable/template/index.qtpl:52
				qw422016.E().S(v.Value)
//line variable/template/index.qtpl:52
				qw422016.N().S(`</span></td> <td> `)
//line variable/template/index.qtpl:54
				if v.Namespace != nil {
//line variable/template/index.qtpl:54
					qw422016.N().S(` <a href="`)
//line variable/template/index.qtpl:55
					qw422016.E().S(v.Namespace.Endpoint())
//line variable/template/index.qtpl:55
					qw422016.N().S(`">`)
//line variable/template/index.qtpl:55
					qw422016.E().S(v.Namespace.Path)
//line variable/template/index.qtpl:55
					qw422016.N().S(`</a> `)
//line variable/template/index.qtpl:56
				} else {
//line variable/template/index.qtpl:56
					qw422016.N().S(` <span class="muted">--</span> `)
//line variable/template/index.qtpl:58
				}
//line variable/template/index.qtpl:58
				qw422016.N().S(` </td> <td class="align-right"> `)
//line variable/template/index.qtpl:61
				if p.User.ID != v.UserID {
//line variable/template/index.qtpl:61
					qw422016.N().S(` <span class="muted">`)
//line variable/template/index.qtpl:62
					qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M15.984 12.984c2.344 0 7.031 1.172 7.031 3.516v2.484h-6v-2.484c0-1.5-0.797-2.625-1.969-3.469 0.328-0.047 0.656-0.047 0.938-0.047zM8.016 12.984c2.344 0 6.984 1.172 6.984 3.516v2.484h-14.016v-2.484c0-2.344 4.688-3.516 7.031-3.516zM8.016 11.016c-1.641 0-3-1.359-3-3s1.359-3 3-3 2.953 1.359 2.953 3-1.313 3-2.953 3zM15.984 11.016c-1.641 0-3-1.359-3-3s1.359-3 3-3 3 1.359 3 3-1.359 3-3 3z"></path>
</svg>
`)
//line variable/template/index.qtpl:62
					qw422016.N().S(`</span> `)
//line variable/template/index.qtpl:63
				}
//line variable/template/index.qtpl:63
				qw422016.N().S(` </td> <td class="align-right"> `)
//line variable/template/index.qtpl:66
				if p.User.ID == v.UserID || v.Namespace != nil && v.Namespace.UserID == p.User.ID {
//line variable/template/index.qtpl:66
					qw422016.N().S(` <form method="POST" action="`)
//line variable/template/index.qtpl:67
					qw422016.E().S(v.Endpoint())
//line variable/template/index.qtpl:67
					qw422016.N().S(`"> `)
//line variable/template/index.qtpl:68
					qw422016.N().S(p.CSRF)
//line variable/template/index.qtpl:68
					qw422016.N().S(` <input type="hidden" name="_method" value="DELETE"/> <button type="submit" class="btn btn-danger">Delete</button> </form> `)
//line variable/template/index.qtpl:72
				}
//line variable/template/index.qtpl:72
				qw422016.N().S(` </td> </tr> `)
//line variable/template/index.qtpl:75
			}
//line variable/template/index.qtpl:75
			qw422016.N().S(` </tbody> </table> `)
//line variable/template/index.qtpl:78
		}
//line variable/template/index.qtpl:78
		qw422016.N().S(` `)
//line variable/template/index.qtpl:79
	}
//line variable/template/index.qtpl:79
	qw422016.N().S(` </div> `)
//line variable/template/index.qtpl:81
}

//line variable/template/index.qtpl:81
func (p *Index) WriteBody(qq422016 qtio422016.Writer) {
//line variable/template/index.qtpl:81
	qw422016 := qt422016.AcquireWriter(qq422016)
//line variable/template/index.qtpl:81
	p.StreamBody(qw422016)
//line variable/template/index.qtpl:81
	qt422016.ReleaseWriter(qw422016)
//line variable/template/index.qtpl:81
}

//line variable/template/index.qtpl:81
func (p *Index) Body() string {
//line variable/template/index.qtpl:81
	qb422016 := qt422016.AcquireByteBuffer()
//line variable/template/index.qtpl:81
	p.WriteBody(qb422016)
//line variable/template/index.qtpl:81
	qs422016 := string(qb422016.B)
//line variable/template/index.qtpl:81
	qt422016.ReleaseByteBuffer(qb422016)
//line variable/template/index.qtpl:81
	return qs422016
//line variable/template/index.qtpl:81
}

//line variable/template/index.qtpl:83
func (p *Index) StreamSection(qw422016 *qt422016.Writer) {
//line variable/template/index.qtpl:83
	qw422016.N().S(` `)
//line variable/template/index.qtpl:84
	p.StreamBody(qw422016)
//line variable/template/index.qtpl:84
	qw422016.N().S(` `)
//line variable/template/index.qtpl:85
}

//line variable/template/index.qtpl:85
func (p *Index) WriteSection(qq422016 qtio422016.Writer) {
//line variable/template/index.qtpl:85
	qw422016 := qt422016.AcquireWriter(qq422016)
//line variable/template/index.qtpl:85
	p.StreamSection(qw422016)
//line variable/template/index.qtpl:85
	qt422016.ReleaseWriter(qw422016)
//line variable/template/index.qtpl:85
}

//line variable/template/index.qtpl:85
func (p *Index) Section() string {
//line variable/template/index.qtpl:85
	qb422016 := qt422016.AcquireByteBuffer()
//line variable/template/index.qtpl:85
	p.WriteSection(qb422016)
//line variable/template/index.qtpl:85
	qs422016 := string(qb422016.B)
//line variable/template/index.qtpl:85
	qt422016.ReleaseByteBuffer(qb422016)
//line variable/template/index.qtpl:85
	return qs422016
//line variable/template/index.qtpl:85
}

//line variable/template/index.qtpl:87
func (p *Index) StreamActions(qw422016 *qt422016.Writer) {
//line variable/template/index.qtpl:87
	qw422016.N().S(` <li><a href="/variables/create" class="btn btn-primary">Create</a></li> `)
//line variable/template/index.qtpl:89
}

//line variable/template/index.qtpl:89
func (p *Index) WriteActions(qq422016 qtio422016.Writer) {
//line variable/template/index.qtpl:89
	qw422016 := qt422016.AcquireWriter(qq422016)
//line variable/template/index.qtpl:89
	p.StreamActions(qw422016)
//line variable/template/index.qtpl:89
	qt422016.ReleaseWriter(qw422016)
//line variable/template/index.qtpl:89
}

//line variable/template/index.qtpl:89
func (p *Index) Actions() string {
//line variable/template/index.qtpl:89
	qb422016 := qt422016.AcquireByteBuffer()
//line variable/template/index.qtpl:89
	p.WriteActions(qb422016)
//line variable/template/index.qtpl:89
	qs422016 := string(qb422016.B)
//line variable/template/index.qtpl:89
	qt422016.ReleaseByteBuffer(qb422016)
//line variable/template/index.qtpl:89
	return qs422016
//line variable/template/index.qtpl:89
}

//line variable/template/index.qtpl:91
func (p *Index) StreamNavigation(qw422016 *qt422016.Writer) {
//line variable/template/index.qtpl:91
}

//line variable/template/index.qtpl:91
func (p *Index) WriteNavigation(qq422016 qtio422016.Writer) {
//line variable/template/index.qtpl:91
	qw422016 := qt422016.AcquireWriter(qq422016)
//line variable/template/index.qtpl:91
	p.StreamNavigation(qw422016)
//line variable/template/index.qtpl:91
	qt422016.ReleaseWriter(qw422016)
//line variable/template/index.qtpl:91
}

//line variable/template/index.qtpl:91
func (p *Index) Navigation() string {
//line variable/template/index.qtpl:91
	qb422016 := qt422016.AcquireByteBuffer()
//line variable/template/index.qtpl:91
	p.WriteNavigation(qb422016)
//line variable/template/index.qtpl:91
	qs422016 := string(qb422016.B)
//line variable/template/index.qtpl:91
	qt422016.ReleaseByteBuffer(qb422016)
//line variable/template/index.qtpl:91
	return qs422016
//line variable/template/index.qtpl:91
}
