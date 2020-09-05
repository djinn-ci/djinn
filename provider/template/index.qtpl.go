// Code generated by qtc from "index.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line provider/template/index.qtpl:2
package template

//line provider/template/index.qtpl:2
import (
	"github.com/andrewpillar/thrall/database"
	"github.com/andrewpillar/thrall/provider"
	"github.com/andrewpillar/thrall/template"
)

//line provider/template/index.qtpl:9
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line provider/template/index.qtpl:9
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line provider/template/index.qtpl:10
type RepoIndex struct {
	template.BasePage

	CSRF      string
	Paginator database.Paginator
	Repos     []*provider.Repo
	Provider  *provider.Provider
	Providers []*provider.Provider
}

var providerNames = map[string]string{
	"github": "GitHub",
	"gitlab": "GitLab",
}

//line provider/template/index.qtpl:27
func (p *RepoIndex) StreamTitle(qw422016 *qt422016.Writer) {
//line provider/template/index.qtpl:27
	qw422016.N().S(` Repositories - Thrall `)
//line provider/template/index.qtpl:29
}

//line provider/template/index.qtpl:29
func (p *RepoIndex) WriteTitle(qq422016 qtio422016.Writer) {
//line provider/template/index.qtpl:29
	qw422016 := qt422016.AcquireWriter(qq422016)
//line provider/template/index.qtpl:29
	p.StreamTitle(qw422016)
//line provider/template/index.qtpl:29
	qt422016.ReleaseWriter(qw422016)
//line provider/template/index.qtpl:29
}

//line provider/template/index.qtpl:29
func (p *RepoIndex) Title() string {
//line provider/template/index.qtpl:29
	qb422016 := qt422016.AcquireByteBuffer()
//line provider/template/index.qtpl:29
	p.WriteTitle(qb422016)
//line provider/template/index.qtpl:29
	qs422016 := string(qb422016.B)
//line provider/template/index.qtpl:29
	qt422016.ReleaseByteBuffer(qb422016)
//line provider/template/index.qtpl:29
	return qs422016
//line provider/template/index.qtpl:29
}

//line provider/template/index.qtpl:31
func (p *RepoIndex) StreamHeader(qw422016 *qt422016.Writer) {
//line provider/template/index.qtpl:31
	qw422016.N().S(` Repositories `)
//line provider/template/index.qtpl:33
}

//line provider/template/index.qtpl:33
func (p *RepoIndex) WriteHeader(qq422016 qtio422016.Writer) {
//line provider/template/index.qtpl:33
	qw422016 := qt422016.AcquireWriter(qq422016)
//line provider/template/index.qtpl:33
	p.StreamHeader(qw422016)
//line provider/template/index.qtpl:33
	qt422016.ReleaseWriter(qw422016)
//line provider/template/index.qtpl:33
}

//line provider/template/index.qtpl:33
func (p *RepoIndex) Header() string {
//line provider/template/index.qtpl:33
	qb422016 := qt422016.AcquireByteBuffer()
//line provider/template/index.qtpl:33
	p.WriteHeader(qb422016)
//line provider/template/index.qtpl:33
	qs422016 := string(qb422016.B)
//line provider/template/index.qtpl:33
	qt422016.ReleaseByteBuffer(qb422016)
//line provider/template/index.qtpl:33
	return qs422016
//line provider/template/index.qtpl:33
}

//line provider/template/index.qtpl:35
func (p *RepoIndex) StreamBody(qw422016 *qt422016.Writer) {
//line provider/template/index.qtpl:35
	qw422016.N().S(` <div class="panel"> <div class="panel-header"> `)
//line provider/template/index.qtpl:38
	qw422016.N().S(`<ul class="panel-nav">`)
//line provider/template/index.qtpl:40
	for _, prv := range p.Providers {
//line provider/template/index.qtpl:40
		qw422016.N().S(`<li><a href="`)
//line provider/template/index.qtpl:42
		qw422016.E().S(p.URL.Path)
//line provider/template/index.qtpl:42
		qw422016.N().S(`?provider=`)
//line provider/template/index.qtpl:42
		qw422016.E().S(prv.Name)
//line provider/template/index.qtpl:42
		qw422016.N().S(`"`)
//line provider/template/index.qtpl:42
		if p.Provider.Name == prv.Name {
//line provider/template/index.qtpl:42
			qw422016.N().S(`class="active"`)
//line provider/template/index.qtpl:42
		}
//line provider/template/index.qtpl:42
		qw422016.N().S(`>`)
//line provider/template/index.qtpl:42
		qw422016.E().S(providerNames[prv.Name])
//line provider/template/index.qtpl:42
		qw422016.N().S(`</a></li>`)
//line provider/template/index.qtpl:44
	}
//line provider/template/index.qtpl:44
	qw422016.N().S(`</ul>`)
//line provider/template/index.qtpl:46
	qw422016.N().S(` </div> `)
//line provider/template/index.qtpl:48
	if !p.Provider.Connected && len(p.Providers) == 0 {
//line provider/template/index.qtpl:48
		qw422016.N().S(` <div class="panel-message muted"> Connect to a Git provider to trigger builds on pushes, and pull requests. Get started by connecting from your account <a href="/settings">settings</a>. </div> `)
//line provider/template/index.qtpl:52
	} else if !p.Provider.Connected {
//line provider/template/index.qtpl:52
		qw422016.N().S(` <div class="panel-message muted"> Connect to `)
//line provider/template/index.qtpl:54
		qw422016.E().S(providerNames[p.Provider.Name])
//line provider/template/index.qtpl:54
		qw422016.N().S(` to trigger builds on pushes, and pull requests. Get started by connecting from your account <a href="/settings">settings</a>. </div> `)
//line provider/template/index.qtpl:56
	} else if len(p.Repos) == 0 {
//line provider/template/index.qtpl:56
		qw422016.N().S(` <div class="panel-message muted">No `)
//line provider/template/index.qtpl:57
		qw422016.E().S(providerNames[p.Provider.Name])
//line provider/template/index.qtpl:57
		qw422016.N().S(` repositories.</div> `)
//line provider/template/index.qtpl:58
	} else {
//line provider/template/index.qtpl:58
		qw422016.N().S(` <table class="table"> <thead> <tr> <th>NAME</th> <th></th> <th></th> </tr> </thead> <tbody> `)
//line provider/template/index.qtpl:68
		for _, r := range p.Repos {
//line provider/template/index.qtpl:68
			qw422016.N().S(` <tr> <td> <span class="muted"> `)
//line provider/template/index.qtpl:72
			switch r.Provider.Name {
//line provider/template/index.qtpl:73
			case "github":
//line provider/template/index.qtpl:73
				qw422016.N().S(` `)
//line provider/template/index.qtpl:74
				qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12 0.297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385 0.6 0.113 0.82-0.258 0.82-0.577 0-0.285-0.010-1.040-0.015-2.040-3.338 0.724-4.042-1.61-4.042-1.61-0.546-1.385-1.335-1.755-1.335-1.755-1.087-0.744 0.084-0.729 0.084-0.729 1.205 0.084 1.838 1.236 1.838 1.236 1.070 1.835 2.809 1.305 3.495 0.998 0.108-0.776 0.417-1.305 0.76-1.605-2.665-0.3-5.466-1.332-5.466-5.93 0-1.31 0.465-2.38 1.235-3.22-0.135-0.303-0.54-1.523 0.105-3.176 0 0 1.005-0.322 3.3 1.23 0.96-0.267 1.98-0.399 3-0.405 1.020 0.006 2.040 0.138 3 0.405 2.28-1.552 3.285-1.23 3.285-1.23 0.645 1.653 0.24 2.873 0.12 3.176 0.765 0.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92 0.42 0.36 0.81 1.096 0.81 2.22 0 1.606-0.015 2.896-0.015 3.286 0 0.315 0.21 0.69 0.825 0.57 4.801-1.574 8.236-6.074 8.236-11.369 0-6.627-5.373-12-12-12z"></path>
</svg>
`)
//line provider/template/index.qtpl:74
				qw422016.N().S(` `)
//line provider/template/index.qtpl:75
			case "gitlab":
//line provider/template/index.qtpl:75
				qw422016.N().S(` `)
//line provider/template/index.qtpl:76
				qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M23.955 13.587l-1.342-4.135-2.664-8.189c-0.135-0.423-0.73-0.423-0.867 0l-2.664 8.187h-8.836l-2.663-8.187c-0.136-0.423-0.734-0.423-0.869-0.003l-2.664 8.189-1.342 4.138c-0.121 0.375 0.014 0.789 0.331 1.023l11.625 8.444 11.625-8.443c0.318-0.235 0.453-0.647 0.33-1.024z"></path>
</svg>
`)
//line provider/template/index.qtpl:76
				qw422016.N().S(` `)
//line provider/template/index.qtpl:77
			}
//line provider/template/index.qtpl:77
			qw422016.N().S(` </span> <a href="`)
//line provider/template/index.qtpl:79
			qw422016.E().S(r.Href)
//line provider/template/index.qtpl:79
			qw422016.N().S(`" target="_blank">`)
//line provider/template/index.qtpl:79
			qw422016.E().S(r.Name)
//line provider/template/index.qtpl:79
			qw422016.N().S(`</a> </td> `)
//line provider/template/index.qtpl:81
			if !r.Provider.Connected {
//line provider/template/index.qtpl:81
				qw422016.N().S(` <td class="warning">`)
//line provider/template/index.qtpl:82
				qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12.984 14.016v-4.031h-1.969v4.031h1.969zM12.984 18v-2.016h-1.969v2.016h1.969zM0.984 21l11.016-18.984 11.016 18.984h-22.031z"></path>
</svg>
`)
//line provider/template/index.qtpl:82
				qw422016.N().S(` Disconnected from `)
//line provider/template/index.qtpl:82
				qw422016.E().S(providerNames[r.Provider.Name])
//line provider/template/index.qtpl:82
				qw422016.N().S(`</td> `)
//line provider/template/index.qtpl:83
			} else {
//line provider/template/index.qtpl:83
				qw422016.N().S(` <td></td> `)
//line provider/template/index.qtpl:85
			}
//line provider/template/index.qtpl:85
			qw422016.N().S(` <td class="align-right"> `)
//line provider/template/index.qtpl:87
			if !r.Enabled {
//line provider/template/index.qtpl:87
				qw422016.N().S(` <form method="POST" action="/repos/enable"> <input type="hidden" name="repo_id" value="`)
//line provider/template/index.qtpl:89
				qw422016.E().V(r.RepoID)
//line provider/template/index.qtpl:89
				qw422016.N().S(`"> <input type="hidden" name="provider" value="`)
//line provider/template/index.qtpl:90
				qw422016.E().S(r.Provider.Name)
//line provider/template/index.qtpl:90
				qw422016.N().S(`"> <input type="hidden" name="name" value="`)
//line provider/template/index.qtpl:91
				qw422016.E().S(r.Name)
//line provider/template/index.qtpl:91
				qw422016.N().S(`"> <input type="hidden" name="href" value="`)
//line provider/template/index.qtpl:92
				qw422016.E().S(r.Href)
//line provider/template/index.qtpl:92
				qw422016.N().S(`"> `)
//line provider/template/index.qtpl:93
			} else {
//line provider/template/index.qtpl:93
				qw422016.N().S(` <form method="POST" action="/repos/disable/`)
//line provider/template/index.qtpl:94
				qw422016.E().V(r.ID)
//line provider/template/index.qtpl:94
				qw422016.N().S(`"> <input type="hidden" name="_method" value="DELETE"/> `)
//line provider/template/index.qtpl:96
			}
//line provider/template/index.qtpl:96
			qw422016.N().S(` `)
//line provider/template/index.qtpl:97
			qw422016.N().S(p.CSRF)
//line provider/template/index.qtpl:97
			qw422016.N().S(` `)
//line provider/template/index.qtpl:98
			if !r.Enabled {
//line provider/template/index.qtpl:98
				qw422016.N().S(` <button type="submit" class="btn btn-primary" `)
//line provider/template/index.qtpl:99
				if !r.Provider.Connected {
//line provider/template/index.qtpl:99
					qw422016.N().S(`disabled="true"`)
//line provider/template/index.qtpl:99
				}
//line provider/template/index.qtpl:99
				qw422016.N().S(`>Enable</button> `)
//line provider/template/index.qtpl:100
			} else {
//line provider/template/index.qtpl:100
				qw422016.N().S(` <button type="submit" class="btn btn-danger" `)
//line provider/template/index.qtpl:101
				if !r.Provider.Connected {
//line provider/template/index.qtpl:101
					qw422016.N().S(`disabled="true"`)
//line provider/template/index.qtpl:101
				}
//line provider/template/index.qtpl:101
				qw422016.N().S(`>Disable</button> `)
//line provider/template/index.qtpl:102
			}
//line provider/template/index.qtpl:102
			qw422016.N().S(` </form> </td> </tr> `)
//line provider/template/index.qtpl:106
		}
//line provider/template/index.qtpl:106
		qw422016.N().S(` </tbody> </table> `)
//line provider/template/index.qtpl:109
	}
//line provider/template/index.qtpl:109
	qw422016.N().S(` </div> `)
//line provider/template/index.qtpl:111
	template.StreamRenderPaginator(qw422016, p.URL, p.Paginator)
//line provider/template/index.qtpl:111
	qw422016.N().S(` `)
//line provider/template/index.qtpl:112
}

//line provider/template/index.qtpl:112
func (p *RepoIndex) WriteBody(qq422016 qtio422016.Writer) {
//line provider/template/index.qtpl:112
	qw422016 := qt422016.AcquireWriter(qq422016)
//line provider/template/index.qtpl:112
	p.StreamBody(qw422016)
//line provider/template/index.qtpl:112
	qt422016.ReleaseWriter(qw422016)
//line provider/template/index.qtpl:112
}

//line provider/template/index.qtpl:112
func (p *RepoIndex) Body() string {
//line provider/template/index.qtpl:112
	qb422016 := qt422016.AcquireByteBuffer()
//line provider/template/index.qtpl:112
	p.WriteBody(qb422016)
//line provider/template/index.qtpl:112
	qs422016 := string(qb422016.B)
//line provider/template/index.qtpl:112
	qt422016.ReleaseByteBuffer(qb422016)
//line provider/template/index.qtpl:112
	return qs422016
//line provider/template/index.qtpl:112
}

//line provider/template/index.qtpl:114
func (p *RepoIndex) StreamActions(qw422016 *qt422016.Writer) {
//line provider/template/index.qtpl:114
	qw422016.N().S(` `)
//line provider/template/index.qtpl:115
	if len(p.Providers) > 0 {
//line provider/template/index.qtpl:115
		qw422016.N().S(` <form method="POST" action="/repos/reload?provider=`)
//line provider/template/index.qtpl:116
		qw422016.E().S(p.Provider.Name)
//line provider/template/index.qtpl:116
		qw422016.N().S(`&page=`)
//line provider/template/index.qtpl:116
		qw422016.E().V(p.Paginator.Page)
//line provider/template/index.qtpl:116
		qw422016.N().S(`"> `)
//line provider/template/index.qtpl:117
		qw422016.N().S(p.CSRF)
//line provider/template/index.qtpl:117
		qw422016.N().S(` <input type="hidden" name="_method" value="PATCH"> <button type="submit" class="btn btn-primary">Reload</button> </form> `)
//line provider/template/index.qtpl:121
	}
//line provider/template/index.qtpl:121
	qw422016.N().S(` `)
//line provider/template/index.qtpl:122
}

//line provider/template/index.qtpl:122
func (p *RepoIndex) WriteActions(qq422016 qtio422016.Writer) {
//line provider/template/index.qtpl:122
	qw422016 := qt422016.AcquireWriter(qq422016)
//line provider/template/index.qtpl:122
	p.StreamActions(qw422016)
//line provider/template/index.qtpl:122
	qt422016.ReleaseWriter(qw422016)
//line provider/template/index.qtpl:122
}

//line provider/template/index.qtpl:122
func (p *RepoIndex) Actions() string {
//line provider/template/index.qtpl:122
	qb422016 := qt422016.AcquireByteBuffer()
//line provider/template/index.qtpl:122
	p.WriteActions(qb422016)
//line provider/template/index.qtpl:122
	qs422016 := string(qb422016.B)
//line provider/template/index.qtpl:122
	qt422016.ReleaseByteBuffer(qb422016)
//line provider/template/index.qtpl:122
	return qs422016
//line provider/template/index.qtpl:122
}

//line provider/template/index.qtpl:124
func (p *RepoIndex) StreamNavigation(qw422016 *qt422016.Writer) {
//line provider/template/index.qtpl:124
}

//line provider/template/index.qtpl:124
func (p *RepoIndex) WriteNavigation(qq422016 qtio422016.Writer) {
//line provider/template/index.qtpl:124
	qw422016 := qt422016.AcquireWriter(qq422016)
//line provider/template/index.qtpl:124
	p.StreamNavigation(qw422016)
//line provider/template/index.qtpl:124
	qt422016.ReleaseWriter(qw422016)
//line provider/template/index.qtpl:124
}

//line provider/template/index.qtpl:124
func (p *RepoIndex) Navigation() string {
//line provider/template/index.qtpl:124
	qb422016 := qt422016.AcquireByteBuffer()
//line provider/template/index.qtpl:124
	p.WriteNavigation(qb422016)
//line provider/template/index.qtpl:124
	qs422016 := string(qb422016.B)
//line provider/template/index.qtpl:124
	qt422016.ReleaseByteBuffer(qb422016)
//line provider/template/index.qtpl:124
	return qs422016
//line provider/template/index.qtpl:124
}
