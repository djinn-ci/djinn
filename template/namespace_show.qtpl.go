// Code generated by qtc from "namespace_show.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/namespace_show.qtpl:2
package template

//line template/namespace_show.qtpl:2
import (
	"regexp"

	"djinn-ci.com/namespace"
)

//line template/namespace_show.qtpl:9
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/namespace_show.qtpl:9
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/namespace_show.qtpl:10
type NamespaceShow struct {
	*Page

	Namespace *namespace.Namespace
	Partial   Partial
}

//line template/namespace_show.qtpl:19
func (p *NamespaceShow) StreamTitle(qw422016 *qt422016.Writer) {
//line template/namespace_show.qtpl:19
	qw422016.N().S(` `)
//line template/namespace_show.qtpl:20
	qw422016.E().S(p.Namespace.User.Username)
//line template/namespace_show.qtpl:20
	qw422016.N().S(`/`)
//line template/namespace_show.qtpl:20
	qw422016.E().S(p.Namespace.Path)
//line template/namespace_show.qtpl:20
	qw422016.N().S(` - `)
//line template/namespace_show.qtpl:20
	p.Partial.StreamTitle(qw422016)
//line template/namespace_show.qtpl:20
	qw422016.N().S(` `)
//line template/namespace_show.qtpl:21
}

//line template/namespace_show.qtpl:21
func (p *NamespaceShow) WriteTitle(qq422016 qtio422016.Writer) {
//line template/namespace_show.qtpl:21
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/namespace_show.qtpl:21
	p.StreamTitle(qw422016)
//line template/namespace_show.qtpl:21
	qt422016.ReleaseWriter(qw422016)
//line template/namespace_show.qtpl:21
}

//line template/namespace_show.qtpl:21
func (p *NamespaceShow) Title() string {
//line template/namespace_show.qtpl:21
	qb422016 := qt422016.AcquireByteBuffer()
//line template/namespace_show.qtpl:21
	p.WriteTitle(qb422016)
//line template/namespace_show.qtpl:21
	qs422016 := string(qb422016.B)
//line template/namespace_show.qtpl:21
	qt422016.ReleaseByteBuffer(qb422016)
//line template/namespace_show.qtpl:21
	return qs422016
//line template/namespace_show.qtpl:21
}

//line template/namespace_show.qtpl:23
func (p *NamespaceShow) StreamHeader(qw422016 *qt422016.Writer) {
//line template/namespace_show.qtpl:23
	qw422016.N().S(` `)
//line template/namespace_show.qtpl:24
	if p.Namespace.Parent != nil {
//line template/namespace_show.qtpl:24
		qw422016.N().S(` <a class="back" href="`)
//line template/namespace_show.qtpl:25
		qw422016.E().S(p.Namespace.Parent.Endpoint())
//line template/namespace_show.qtpl:25
		qw422016.N().S(`">`)
//line template/namespace_show.qtpl:25
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
//line template/namespace_show.qtpl:25
		qw422016.N().S(`</a> `)
//line template/namespace_show.qtpl:26
	} else {
//line template/namespace_show.qtpl:26
		qw422016.N().S(` `)
//line template/namespace_show.qtpl:27
		if p.User != nil {
//line template/namespace_show.qtpl:27
			qw422016.N().S(` <a class="back" href="/namespaces">`)
//line template/namespace_show.qtpl:28
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
//line template/namespace_show.qtpl:28
			qw422016.N().S(`</a> `)
//line template/namespace_show.qtpl:29
		}
//line template/namespace_show.qtpl:29
		qw422016.N().S(` `)
//line template/namespace_show.qtpl:30
	}
//line template/namespace_show.qtpl:30
	qw422016.N().S(` `)
//line template/namespace_show.qtpl:31
	streamnamespacePath(qw422016, p.Namespace.User.Username, p.Namespace.Path)
//line template/namespace_show.qtpl:31
	qw422016.N().S(` `)
//line template/namespace_show.qtpl:32
	if tag := p.URL.Query().Get("tag"); tag != "" {
//line template/namespace_show.qtpl:32
		qw422016.N().S(` <span class="pill pill-light"> `)
//line template/namespace_show.qtpl:34
		qw422016.E().S(tag)
//line template/namespace_show.qtpl:34
		qw422016.N().S(` <a href="`)
//line template/namespace_show.qtpl:35
		qw422016.E().S(p.URL.Path)
//line template/namespace_show.qtpl:35
		qw422016.N().S(`">`)
//line template/namespace_show.qtpl:35
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M18.984 6.422l-5.578 5.578 5.578 5.578-1.406 1.406-5.578-5.578-5.578 5.578-1.406-1.406 5.578-5.578-5.578-5.578 1.406-1.406 5.578 5.578 5.578-5.578z"></path>
</svg>
`)
//line template/namespace_show.qtpl:35
		qw422016.N().S(`</a> </span> `)
//line template/namespace_show.qtpl:37
	}
//line template/namespace_show.qtpl:37
	qw422016.N().S(` <small>`)
//line template/namespace_show.qtpl:38
	qw422016.E().S(p.Namespace.Description)
//line template/namespace_show.qtpl:38
	qw422016.N().S(`</small> `)
//line template/namespace_show.qtpl:39
}

//line template/namespace_show.qtpl:39
func (p *NamespaceShow) WriteHeader(qq422016 qtio422016.Writer) {
//line template/namespace_show.qtpl:39
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/namespace_show.qtpl:39
	p.StreamHeader(qw422016)
//line template/namespace_show.qtpl:39
	qt422016.ReleaseWriter(qw422016)
//line template/namespace_show.qtpl:39
}

//line template/namespace_show.qtpl:39
func (p *NamespaceShow) Header() string {
//line template/namespace_show.qtpl:39
	qb422016 := qt422016.AcquireByteBuffer()
//line template/namespace_show.qtpl:39
	p.WriteHeader(qb422016)
//line template/namespace_show.qtpl:39
	qs422016 := string(qb422016.B)
//line template/namespace_show.qtpl:39
	qt422016.ReleaseByteBuffer(qb422016)
//line template/namespace_show.qtpl:39
	return qs422016
//line template/namespace_show.qtpl:39
}

//line template/namespace_show.qtpl:41
func (p *NamespaceShow) StreamActions(qw422016 *qt422016.Writer) {
//line template/namespace_show.qtpl:41
	qw422016.N().S(` `)
//line template/namespace_show.qtpl:42
	if p.User != nil && p.User.ID == p.Namespace.UserID {
//line template/namespace_show.qtpl:42
		qw422016.N().S(` <li><a href="`)
//line template/namespace_show.qtpl:43
		qw422016.E().S(p.Namespace.Endpoint("edit"))
//line template/namespace_show.qtpl:43
		qw422016.N().S(`" class="btn btn-primary">Edit</a></li> `)
//line template/namespace_show.qtpl:44
		if p.Namespace.Level+1 < namespace.MaxDepth {
//line template/namespace_show.qtpl:44
			qw422016.N().S(` <li><a href="/namespaces/create?parent=`)
//line template/namespace_show.qtpl:45
			qw422016.E().S(p.Namespace.Path)
//line template/namespace_show.qtpl:45
			qw422016.N().S(`" class="btn btn-primary">Create</a></li> `)
//line template/namespace_show.qtpl:46
		}
//line template/namespace_show.qtpl:46
		qw422016.N().S(` `)
//line template/namespace_show.qtpl:47
	}
//line template/namespace_show.qtpl:47
	qw422016.N().S(` `)
//line template/namespace_show.qtpl:48
}

//line template/namespace_show.qtpl:48
func (p *NamespaceShow) WriteActions(qq422016 qtio422016.Writer) {
//line template/namespace_show.qtpl:48
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/namespace_show.qtpl:48
	p.StreamActions(qw422016)
//line template/namespace_show.qtpl:48
	qt422016.ReleaseWriter(qw422016)
//line template/namespace_show.qtpl:48
}

//line template/namespace_show.qtpl:48
func (p *NamespaceShow) Actions() string {
//line template/namespace_show.qtpl:48
	qb422016 := qt422016.AcquireByteBuffer()
//line template/namespace_show.qtpl:48
	p.WriteActions(qb422016)
//line template/namespace_show.qtpl:48
	qs422016 := string(qb422016.B)
//line template/namespace_show.qtpl:48
	qt422016.ReleaseByteBuffer(qb422016)
//line template/namespace_show.qtpl:48
	return qs422016
//line template/namespace_show.qtpl:48
}

//line template/namespace_show.qtpl:51
func (p *NamespaceShow) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/namespace_show.qtpl:52
	for _, link := range []NavLink{
		{
			Title:   "Builds",
			Href:    p.Namespace.Endpoint(),
			Icon:    "static/svg/build.svg",
			Pattern: regexp.MustCompile("^" + p.Namespace.Endpoint() + "$"),
		},
		{
			Title:   "Namespaces",
			Href:    p.Namespace.Endpoint("namespaces"),
			Icon:    "static/svg/folder.svg",
			Pattern: regexp.MustCompile("^" + p.Namespace.Endpoint("namespaces") + "$"),
		},
		{
			Title:   "Images",
			Href:    p.Namespace.Endpoint("images"),
			Icon:    "static/svg/image.svg",
			Pattern: regexp.MustCompile(p.Namespace.Endpoint("images")),
		},
		{
			Title:   "Objects",
			Href:    p.Namespace.Endpoint("objects"),
			Icon:    "static/svg/upload.svg",
			Pattern: regexp.MustCompile(p.Namespace.Endpoint("objects")),
		},
		{
			Title:   "Variables",
			Href:    p.Namespace.Endpoint("variables"),
			Icon:    "static/svg/code.svg",
			Pattern: regexp.MustCompile(p.Namespace.Endpoint("variables")),
		},
		{
			Title:   "SSH Keys",
			Href:    p.Namespace.Endpoint("keys"),
			Icon:    "static/svg/key.svg",
			Pattern: regexp.MustCompile(p.Namespace.Endpoint("keys")),
		},
		{
			Title:     "Invites",
			Href:      p.Namespace.Endpoint("invites"),
			Icon:      "static/svg/mail.svg",
			Pattern:   regexp.MustCompile(p.Namespace.Endpoint("invites")),
			Condition: func() bool { return p.User.ID == p.Namespace.UserID },
		},
		{
			Title:     "Collaborators",
			Href:      p.Namespace.Endpoint("collaborators"),
			Icon:      "static/svg/mail.svg",
			Pattern:   regexp.MustCompile(p.Namespace.Endpoint("collaborators")),
			Condition: func() bool { return p.User.ID == p.Namespace.UserID },
		},
		{
			Title:     "Webhooks",
			Href:      p.Namespace.Endpoint("webhooks"),
			Icon:      "static/svg/all_out.svg",
			Pattern:   regexp.MustCompile(p.Namespace.Endpoint("webhooks")),
			Condition: func() bool { return p.User.Has("webhook:read") },
		},
	} {
//line template/namespace_show.qtpl:110
		qw422016.N().S(`<li>`)
//line template/namespace_show.qtpl:111
		link.StreamRender(qw422016, p.URL.Path)
//line template/namespace_show.qtpl:111
		qw422016.N().S(`</li>`)
//line template/namespace_show.qtpl:112
	}
//line template/namespace_show.qtpl:113
}

//line template/namespace_show.qtpl:113
func (p *NamespaceShow) WriteNavigation(qq422016 qtio422016.Writer) {
//line template/namespace_show.qtpl:113
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/namespace_show.qtpl:113
	p.StreamNavigation(qw422016)
//line template/namespace_show.qtpl:113
	qt422016.ReleaseWriter(qw422016)
//line template/namespace_show.qtpl:113
}

//line template/namespace_show.qtpl:113
func (p *NamespaceShow) Navigation() string {
//line template/namespace_show.qtpl:113
	qb422016 := qt422016.AcquireByteBuffer()
//line template/namespace_show.qtpl:113
	p.WriteNavigation(qb422016)
//line template/namespace_show.qtpl:113
	qs422016 := string(qb422016.B)
//line template/namespace_show.qtpl:113
	qt422016.ReleaseByteBuffer(qb422016)
//line template/namespace_show.qtpl:113
	return qs422016
//line template/namespace_show.qtpl:113
}

//line template/namespace_show.qtpl:115
func (p *NamespaceShow) StreamFooter(qw422016 *qt422016.Writer) {
//line template/namespace_show.qtpl:115
}

//line template/namespace_show.qtpl:115
func (p *NamespaceShow) WriteFooter(qq422016 qtio422016.Writer) {
//line template/namespace_show.qtpl:115
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/namespace_show.qtpl:115
	p.StreamFooter(qw422016)
//line template/namespace_show.qtpl:115
	qt422016.ReleaseWriter(qw422016)
//line template/namespace_show.qtpl:115
}

//line template/namespace_show.qtpl:115
func (p *NamespaceShow) Footer() string {
//line template/namespace_show.qtpl:115
	qb422016 := qt422016.AcquireByteBuffer()
//line template/namespace_show.qtpl:115
	p.WriteFooter(qb422016)
//line template/namespace_show.qtpl:115
	qs422016 := string(qb422016.B)
//line template/namespace_show.qtpl:115
	qt422016.ReleaseByteBuffer(qb422016)
//line template/namespace_show.qtpl:115
	return qs422016
//line template/namespace_show.qtpl:115
}

//line template/namespace_show.qtpl:117
func (p *NamespaceShow) StreamBody(qw422016 *qt422016.Writer) {
//line template/namespace_show.qtpl:118
	p.Partial.StreamBody(qw422016)
//line template/namespace_show.qtpl:119
}

//line template/namespace_show.qtpl:119
func (p *NamespaceShow) WriteBody(qq422016 qtio422016.Writer) {
//line template/namespace_show.qtpl:119
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/namespace_show.qtpl:119
	p.StreamBody(qw422016)
//line template/namespace_show.qtpl:119
	qt422016.ReleaseWriter(qw422016)
//line template/namespace_show.qtpl:119
}

//line template/namespace_show.qtpl:119
func (p *NamespaceShow) Body() string {
//line template/namespace_show.qtpl:119
	qb422016 := qt422016.AcquireByteBuffer()
//line template/namespace_show.qtpl:119
	p.WriteBody(qb422016)
//line template/namespace_show.qtpl:119
	qs422016 := string(qb422016.B)
//line template/namespace_show.qtpl:119
	qt422016.ReleaseByteBuffer(qb422016)
//line template/namespace_show.qtpl:119
	return qs422016
//line template/namespace_show.qtpl:119
}
