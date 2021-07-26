// Code generated by qtc from "index.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line image/template/index.qtpl:2
package template

//line image/template/index.qtpl:2
import (
	htmltemplate "html/template"

	"djinn-ci.com/database"
	"djinn-ci.com/image"
	"djinn-ci.com/template"
)

//line image/template/index.qtpl:11
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line image/template/index.qtpl:11
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line image/template/index.qtpl:12
type Index struct {
	template.BasePage

	CSRF      htmltemplate.HTML
	Paginator database.Paginator
	Images    []*image.Image
	Search    string
}

//line image/template/index.qtpl:23
func (p *Index) StreamTitle(qw422016 *qt422016.Writer) {
//line image/template/index.qtpl:23
	qw422016.N().S(` Images - Djinn CI `)
//line image/template/index.qtpl:25
}

//line image/template/index.qtpl:25
func (p *Index) WriteTitle(qq422016 qtio422016.Writer) {
//line image/template/index.qtpl:25
	qw422016 := qt422016.AcquireWriter(qq422016)
//line image/template/index.qtpl:25
	p.StreamTitle(qw422016)
//line image/template/index.qtpl:25
	qt422016.ReleaseWriter(qw422016)
//line image/template/index.qtpl:25
}

//line image/template/index.qtpl:25
func (p *Index) Title() string {
//line image/template/index.qtpl:25
	qb422016 := qt422016.AcquireByteBuffer()
//line image/template/index.qtpl:25
	p.WriteTitle(qb422016)
//line image/template/index.qtpl:25
	qs422016 := string(qb422016.B)
//line image/template/index.qtpl:25
	qt422016.ReleaseByteBuffer(qb422016)
//line image/template/index.qtpl:25
	return qs422016
//line image/template/index.qtpl:25
}

//line image/template/index.qtpl:27
func (p *Index) StreamBody(qw422016 *qt422016.Writer) {
//line image/template/index.qtpl:27
	qw422016.N().S(` <div class="panel"> `)
//line image/template/index.qtpl:29
	if len(p.Images) == 0 && p.Search == "" {
//line image/template/index.qtpl:29
		qw422016.N().S(` <div class="panel-message muted">Upload custom images to use as build environments.</div> `)
//line image/template/index.qtpl:31
	} else {
//line image/template/index.qtpl:31
		qw422016.N().S(` <div class="panel-header">`)
//line image/template/index.qtpl:32
		template.StreamRenderSearch(qw422016, p.URL.Path, p.Search, "Find an image...")
//line image/template/index.qtpl:32
		qw422016.N().S(`</div> `)
//line image/template/index.qtpl:33
		if len(p.Images) == 0 && p.Search != "" {
//line image/template/index.qtpl:33
			qw422016.N().S(` <div class="panel-message muted">No results found.</div> `)
//line image/template/index.qtpl:35
		} else {
//line image/template/index.qtpl:35
			qw422016.N().S(` <table class="table"> <thead> <tr> <th></th> <th>NAME</th> <th>NAMESPACE</th> <th>SOURCE</th> <th></th> <th></th> <th></th> </tr> </thead> <tbody> `)
//line image/template/index.qtpl:49
			for _, i := range p.Images {
//line image/template/index.qtpl:49
				qw422016.N().S(` <tr> <td> `)
//line image/template/index.qtpl:52
				if i.Download != nil && i.Download.StartedAt.Valid && !i.Download.FinishedAt.Valid {
//line image/template/index.qtpl:52
					qw422016.N().S(` <span class="muted">`)
//line image/template/index.qtpl:53
					qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M5.016 18h13.969v2.016h-13.969v-2.016zM18.984 9l-6.984 6.984-6.984-6.984h3.984v-6h6v6h3.984z"></path>
</svg>
`)
//line image/template/index.qtpl:53
					qw422016.N().S(`</span> `)
//line image/template/index.qtpl:54
				}
//line image/template/index.qtpl:54
				qw422016.N().S(` </td> <td>`)
//line image/template/index.qtpl:56
				qw422016.E().S(i.Name)
//line image/template/index.qtpl:56
				qw422016.N().S(`</td> <td> `)
//line image/template/index.qtpl:58
				if i.Namespace != nil {
//line image/template/index.qtpl:58
					qw422016.N().S(` <a href="`)
//line image/template/index.qtpl:59
					qw422016.E().S(i.Namespace.Endpoint())
//line image/template/index.qtpl:59
					qw422016.N().S(`">`)
//line image/template/index.qtpl:59
					qw422016.E().S(i.Namespace.Path)
//line image/template/index.qtpl:59
					qw422016.N().S(`</a> `)
//line image/template/index.qtpl:60
				} else {
//line image/template/index.qtpl:60
					qw422016.N().S(` <span class="muted">--</span> `)
//line image/template/index.qtpl:62
				}
//line image/template/index.qtpl:62
				qw422016.N().S(` </td> <td> `)
//line image/template/index.qtpl:65
				if i.Download != nil {
//line image/template/index.qtpl:65
					qw422016.N().S(` `)
//line image/template/index.qtpl:66
					qw422016.E().S(i.Download.Source.String())
//line image/template/index.qtpl:66
					qw422016.N().S(` `)
//line image/template/index.qtpl:67
				} else {
//line image/template/index.qtpl:67
					qw422016.N().S(` <span class="muted">--</span> `)
//line image/template/index.qtpl:69
				}
//line image/template/index.qtpl:69
				qw422016.N().S(` </td> <td class="align-right"> `)
//line image/template/index.qtpl:72
				if p.User.ID != i.UserID {
//line image/template/index.qtpl:72
					qw422016.N().S(` <span class="muted">`)
//line image/template/index.qtpl:73
					qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M15.984 12.984c2.344 0 7.031 1.172 7.031 3.516v2.484h-6v-2.484c0-1.5-0.797-2.625-1.969-3.469 0.328-0.047 0.656-0.047 0.938-0.047zM8.016 12.984c2.344 0 6.984 1.172 6.984 3.516v2.484h-14.016v-2.484c0-2.344 4.688-3.516 7.031-3.516zM8.016 11.016c-1.641 0-3-1.359-3-3s1.359-3 3-3 2.953 1.359 2.953 3-1.313 3-2.953 3zM15.984 11.016c-1.641 0-3-1.359-3-3s1.359-3 3-3 3 1.359 3 3-1.359 3-3 3z"></path>
</svg>
`)
//line image/template/index.qtpl:73
					qw422016.N().S(`</span> `)
//line image/template/index.qtpl:74
				}
//line image/template/index.qtpl:74
				qw422016.N().S(` </td> <td class="align-right"> <a class="btn btn-primary" href="`)
//line image/template/index.qtpl:77
				qw422016.E().S(i.Endpoint("download", i.Name))
//line image/template/index.qtpl:77
				qw422016.N().S(`">Download</a> `)
//line image/template/index.qtpl:78
				if p.User.ID == i.UserID || i.Namespace != nil && i.Namespace.UserID == p.User.ID {
//line image/template/index.qtpl:78
					qw422016.N().S(` <form method="POST" action="`)
//line image/template/index.qtpl:79
					qw422016.E().S(i.Endpoint())
//line image/template/index.qtpl:79
					qw422016.N().S(`"> `)
//line image/template/index.qtpl:80
					qw422016.N().V(p.CSRF)
//line image/template/index.qtpl:80
					qw422016.N().S(` <input type="hidden" name="_method" value="DELETE"/> <button type="submit" class="btn btn-danger">Delete</button> </form> `)
//line image/template/index.qtpl:84
				}
//line image/template/index.qtpl:84
				qw422016.N().S(` </td> </tr> `)
//line image/template/index.qtpl:87
			}
//line image/template/index.qtpl:87
			qw422016.N().S(` </tbody> </table> `)
//line image/template/index.qtpl:90
		}
//line image/template/index.qtpl:90
		qw422016.N().S(` `)
//line image/template/index.qtpl:91
	}
//line image/template/index.qtpl:91
	qw422016.N().S(` </div> `)
//line image/template/index.qtpl:93
}

//line image/template/index.qtpl:93
func (p *Index) WriteBody(qq422016 qtio422016.Writer) {
//line image/template/index.qtpl:93
	qw422016 := qt422016.AcquireWriter(qq422016)
//line image/template/index.qtpl:93
	p.StreamBody(qw422016)
//line image/template/index.qtpl:93
	qt422016.ReleaseWriter(qw422016)
//line image/template/index.qtpl:93
}

//line image/template/index.qtpl:93
func (p *Index) Body() string {
//line image/template/index.qtpl:93
	qb422016 := qt422016.AcquireByteBuffer()
//line image/template/index.qtpl:93
	p.WriteBody(qb422016)
//line image/template/index.qtpl:93
	qs422016 := string(qb422016.B)
//line image/template/index.qtpl:93
	qt422016.ReleaseByteBuffer(qb422016)
//line image/template/index.qtpl:93
	return qs422016
//line image/template/index.qtpl:93
}

//line image/template/index.qtpl:95
func (p *Index) StreamSection(qw422016 *qt422016.Writer) {
//line image/template/index.qtpl:95
	qw422016.N().S(` `)
//line image/template/index.qtpl:96
	p.StreamBody(qw422016)
//line image/template/index.qtpl:96
	qw422016.N().S(` `)
//line image/template/index.qtpl:97
}

//line image/template/index.qtpl:97
func (p *Index) WriteSection(qq422016 qtio422016.Writer) {
//line image/template/index.qtpl:97
	qw422016 := qt422016.AcquireWriter(qq422016)
//line image/template/index.qtpl:97
	p.StreamSection(qw422016)
//line image/template/index.qtpl:97
	qt422016.ReleaseWriter(qw422016)
//line image/template/index.qtpl:97
}

//line image/template/index.qtpl:97
func (p *Index) Section() string {
//line image/template/index.qtpl:97
	qb422016 := qt422016.AcquireByteBuffer()
//line image/template/index.qtpl:97
	p.WriteSection(qb422016)
//line image/template/index.qtpl:97
	qs422016 := string(qb422016.B)
//line image/template/index.qtpl:97
	qt422016.ReleaseByteBuffer(qb422016)
//line image/template/index.qtpl:97
	return qs422016
//line image/template/index.qtpl:97
}

//line image/template/index.qtpl:99
func (p *Index) StreamHeader(qw422016 *qt422016.Writer) {
//line image/template/index.qtpl:99
	qw422016.N().S(` Images `)
//line image/template/index.qtpl:101
}

//line image/template/index.qtpl:101
func (p *Index) WriteHeader(qq422016 qtio422016.Writer) {
//line image/template/index.qtpl:101
	qw422016 := qt422016.AcquireWriter(qq422016)
//line image/template/index.qtpl:101
	p.StreamHeader(qw422016)
//line image/template/index.qtpl:101
	qt422016.ReleaseWriter(qw422016)
//line image/template/index.qtpl:101
}

//line image/template/index.qtpl:101
func (p *Index) Header() string {
//line image/template/index.qtpl:101
	qb422016 := qt422016.AcquireByteBuffer()
//line image/template/index.qtpl:101
	p.WriteHeader(qb422016)
//line image/template/index.qtpl:101
	qs422016 := string(qb422016.B)
//line image/template/index.qtpl:101
	qt422016.ReleaseByteBuffer(qb422016)
//line image/template/index.qtpl:101
	return qs422016
//line image/template/index.qtpl:101
}

//line image/template/index.qtpl:103
func (p *Index) StreamActions(qw422016 *qt422016.Writer) {
//line image/template/index.qtpl:103
	qw422016.N().S(` `)
//line image/template/index.qtpl:104
	if _, ok := p.User.Permissions["image:write"]; ok {
//line image/template/index.qtpl:104
		qw422016.N().S(` <li><a href="/images/create" class="btn btn-primary">Create</a></li> `)
//line image/template/index.qtpl:106
	}
//line image/template/index.qtpl:106
	qw422016.N().S(` `)
//line image/template/index.qtpl:107
}

//line image/template/index.qtpl:107
func (p *Index) WriteActions(qq422016 qtio422016.Writer) {
//line image/template/index.qtpl:107
	qw422016 := qt422016.AcquireWriter(qq422016)
//line image/template/index.qtpl:107
	p.StreamActions(qw422016)
//line image/template/index.qtpl:107
	qt422016.ReleaseWriter(qw422016)
//line image/template/index.qtpl:107
}

//line image/template/index.qtpl:107
func (p *Index) Actions() string {
//line image/template/index.qtpl:107
	qb422016 := qt422016.AcquireByteBuffer()
//line image/template/index.qtpl:107
	p.WriteActions(qb422016)
//line image/template/index.qtpl:107
	qs422016 := string(qb422016.B)
//line image/template/index.qtpl:107
	qt422016.ReleaseByteBuffer(qb422016)
//line image/template/index.qtpl:107
	return qs422016
//line image/template/index.qtpl:107
}

//line image/template/index.qtpl:109
func (p *Index) StreamNavigation(qw422016 *qt422016.Writer) {
//line image/template/index.qtpl:109
}

//line image/template/index.qtpl:109
func (p *Index) WriteNavigation(qq422016 qtio422016.Writer) {
//line image/template/index.qtpl:109
	qw422016 := qt422016.AcquireWriter(qq422016)
//line image/template/index.qtpl:109
	p.StreamNavigation(qw422016)
//line image/template/index.qtpl:109
	qt422016.ReleaseWriter(qw422016)
//line image/template/index.qtpl:109
}

//line image/template/index.qtpl:109
func (p *Index) Navigation() string {
//line image/template/index.qtpl:109
	qb422016 := qt422016.AcquireByteBuffer()
//line image/template/index.qtpl:109
	p.WriteNavigation(qb422016)
//line image/template/index.qtpl:109
	qs422016 := string(qb422016.B)
//line image/template/index.qtpl:109
	qt422016.ReleaseByteBuffer(qb422016)
//line image/template/index.qtpl:109
	return qs422016
//line image/template/index.qtpl:109
}
