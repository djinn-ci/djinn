// This file is automatically generated by qtc from "edit.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/namespace/edit.qtpl:2
package namespace

//line template/namespace/edit.qtpl:2
import (
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

//line template/namespace/edit.qtpl:9
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/namespace/edit.qtpl:9
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/namespace/edit.qtpl:10
type EditPage struct {
	*template.Page

	Errors    form.Errors
	Form      form.Form
	Namespace *model.Namespace
}

//line template/namespace/edit.qtpl:20
func (p *EditPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/namespace/edit.qtpl:20
	qw422016.N().S(` Edit Namespace - Thrall `)
//line template/namespace/edit.qtpl:22
}

//line template/namespace/edit.qtpl:22
func (p *EditPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/namespace/edit.qtpl:22
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/edit.qtpl:22
	p.StreamTitle(qw422016)
	//line template/namespace/edit.qtpl:22
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/edit.qtpl:22
}

//line template/namespace/edit.qtpl:22
func (p *EditPage) Title() string {
	//line template/namespace/edit.qtpl:22
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/edit.qtpl:22
	p.WriteTitle(qb422016)
	//line template/namespace/edit.qtpl:22
	qs422016 := string(qb422016.B)
	//line template/namespace/edit.qtpl:22
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/edit.qtpl:22
	return qs422016
//line template/namespace/edit.qtpl:22
}

//line template/namespace/edit.qtpl:24
func (p *EditPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/namespace/edit.qtpl:24
	qw422016.N().S(` <div class="panel"> <form class="slim" method="POST" action="`)
	//line template/namespace/edit.qtpl:26
	qw422016.E().S(p.Namespace.URI())
	//line template/namespace/edit.qtpl:26
	qw422016.N().S(`"> <input type="hidden" name="_method" value="PATCH"/> `)
	//line template/namespace/edit.qtpl:28
	if p.Errors.First("namespace") != "" {
		//line template/namespace/edit.qtpl:28
		qw422016.N().S(` <div class="form-error">Failed to create namespace: `)
		//line template/namespace/edit.qtpl:29
		qw422016.E().S(p.Errors.First("namespace"))
		//line template/namespace/edit.qtpl:29
		qw422016.N().S(`</div> `)
		//line template/namespace/edit.qtpl:30
	}
	//line template/namespace/edit.qtpl:30
	qw422016.N().S(` <div class="form-field"> <label class="label">Name</label> `)
	//line template/namespace/edit.qtpl:33
	if p.Form.Get("name") != "" {
		//line template/namespace/edit.qtpl:33
		qw422016.N().S(` <input class="text" type="text" name="name" value="`)
		//line template/namespace/edit.qtpl:34
		qw422016.E().S(p.Form.Get("name"))
		//line template/namespace/edit.qtpl:34
		qw422016.N().S(`" autocomplete="off"/> `)
		//line template/namespace/edit.qtpl:35
	} else {
		//line template/namespace/edit.qtpl:35
		qw422016.N().S(` <input class="text" type="text" name="name" value="`)
		//line template/namespace/edit.qtpl:36
		qw422016.E().S(p.Namespace.Name)
		//line template/namespace/edit.qtpl:36
		qw422016.N().S(`" autocomplete="off"/> `)
		//line template/namespace/edit.qtpl:37
	}
	//line template/namespace/edit.qtpl:37
	qw422016.N().S(` <div class="error">`)
	//line template/namespace/edit.qtpl:38
	qw422016.E().S(p.Errors.First("description"))
	//line template/namespace/edit.qtpl:38
	qw422016.N().S(`</div> </div> <div class="form-field"> <label class="label">Description <small>(optional)</small></label> `)
	//line template/namespace/edit.qtpl:42
	if p.Form.Get("description") != "" {
		//line template/namespace/edit.qtpl:42
		qw422016.N().S(` <input class="text" type="text" name="description" value="`)
		//line template/namespace/edit.qtpl:43
		qw422016.E().S(p.Form.Get("description"))
		//line template/namespace/edit.qtpl:43
		qw422016.N().S(`"/> `)
		//line template/namespace/edit.qtpl:44
	} else {
		//line template/namespace/edit.qtpl:44
		qw422016.N().S(` <input class="text" type="text" name="description" value="`)
		//line template/namespace/edit.qtpl:45
		qw422016.E().S(p.Namespace.Description)
		//line template/namespace/edit.qtpl:45
		qw422016.N().S(`"/> `)
		//line template/namespace/edit.qtpl:46
	}
	//line template/namespace/edit.qtpl:46
	qw422016.N().S(` <div class="error">`)
	//line template/namespace/edit.qtpl:47
	qw422016.E().S(p.Errors.First("description"))
	//line template/namespace/edit.qtpl:47
	qw422016.N().S(`</div> </div> <div class="form-field"> <label class="option"> <input class="selector" type="radio" name="visibility" value="private" `)
	//line template/namespace/edit.qtpl:51
	if p.Namespace.Visibility == model.Private {
		//line template/namespace/edit.qtpl:51
		qw422016.N().S(`checked="true"`)
		//line template/namespace/edit.qtpl:51
	}
	//line template/namespace/edit.qtpl:51
	qw422016.N().S(`/> <label class="label">Private</label> `)
	//line template/namespace/edit.qtpl:53
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M15.094 8.016v-2.016c0-1.688-1.406-3.094-3.094-3.094s-3.094 1.406-3.094 3.094v2.016h6.188zM12 17.016c1.078 0 2.016-0.938 2.016-2.016s-0.938-2.016-2.016-2.016-2.016 0.938-2.016 2.016 0.938 2.016 2.016 2.016zM18 8.016c1.078 0 2.016 0.891 2.016 1.969v10.031c0 1.078-0.938 1.969-2.016 1.969h-12c-1.078 0-2.016-0.891-2.016-1.969v-10.031c0-1.078 0.938-1.969 2.016-1.969h0.984v-2.016c0-2.766 2.25-5.016 5.016-5.016s5.016 2.25 5.016 5.016v2.016h0.984z"></path>
</svg>
`)
	//line template/namespace/edit.qtpl:53
	qw422016.N().S(` <div class="description">You choose who can view builds in the namespace.</div> </label> <label class="option"> <input class="selector" type="radio" name="visibility" value="internal" `)
	//line template/namespace/edit.qtpl:57
	if p.Namespace.Visibility == model.Internal {
		//line template/namespace/edit.qtpl:57
		qw422016.N().S(`checked="true"`)
		//line template/namespace/edit.qtpl:57
	}
	//line template/namespace/edit.qtpl:57
	qw422016.N().S(`/> <label class="label">Internal</label> `)
	//line template/namespace/edit.qtpl:59
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12 0.984l9 4.031v6c0 5.531-3.844 10.734-9 12-5.156-1.266-9-6.469-9-12v-6zM12 12v8.953c3.703-1.172 6.469-4.828 6.984-8.953h-6.984zM12 12v-8.813l-6.984 3.094v5.719h6.984z"></path>
</svg>
`)
	//line template/namespace/edit.qtpl:59
	qw422016.N().S(` <div class="description">Anyone with an account will be able to view builds in the namespace.</div> </label> <label class="option"> <input class="selector" type="radio" name="visibility" value="public" `)
	//line template/namespace/edit.qtpl:63
	if p.Namespace.Visibility == model.Public {
		//line template/namespace/edit.qtpl:63
		qw422016.N().S(`checked="true"`)
		//line template/namespace/edit.qtpl:63
	}
	//line template/namespace/edit.qtpl:63
	qw422016.N().S(`/> <label class="label">Public</label> `)
	//line template/namespace/edit.qtpl:65
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M17.906 17.391c1.313-1.406 2.109-3.328 2.109-5.391 0-3.328-2.063-6.234-5.016-7.406v0.422c0 1.078-0.938 1.969-2.016 1.969h-1.969v2.016c0 0.563-0.469 0.984-1.031 0.984h-1.969v2.016h6c0.563 0 0.984 0.422 0.984 0.984v3h0.984c0.891 0 1.641 0.609 1.922 1.406zM11.016 19.922v-1.922c-1.078 0-2.016-0.938-2.016-2.016v-0.984l-4.781-4.781c-0.141 0.563-0.234 1.172-0.234 1.781 0 4.078 3.094 7.453 7.031 7.922zM12 2.016c5.531 0 9.984 4.453 9.984 9.984s-4.453 9.984-9.984 9.984-9.984-4.453-9.984-9.984 4.453-9.984 9.984-9.984z"></path>
</svg>
`)
	//line template/namespace/edit.qtpl:65
	qw422016.N().S(` <div class="description">Anyone will be able to view builds in the namespace.</div> </label> </div> <div class="form-field"> <button type="submit" class="button button-primary">Save</button> </div> </form> </div> <div class="panel"> <form class="slim" method="POST" action="`)
	//line template/namespace/edit.qtpl:75
	qw422016.E().S(p.Namespace.URI())
	//line template/namespace/edit.qtpl:75
	qw422016.N().S(`"> <input type="hidden" name="_method" value="DELETE"/> <div class="form-field"> <button type="submit" class="right button button-danger">Delete</button> <div> <strong>Delete Namespace</strong><br/> Builds within the namespace will not be deleted. </div> </div> </form> </div> `)
//line template/namespace/edit.qtpl:86
}

//line template/namespace/edit.qtpl:86
func (p *EditPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/namespace/edit.qtpl:86
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/edit.qtpl:86
	p.StreamBody(qw422016)
	//line template/namespace/edit.qtpl:86
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/edit.qtpl:86
}

//line template/namespace/edit.qtpl:86
func (p *EditPage) Body() string {
	//line template/namespace/edit.qtpl:86
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/edit.qtpl:86
	p.WriteBody(qb422016)
	//line template/namespace/edit.qtpl:86
	qs422016 := string(qb422016.B)
	//line template/namespace/edit.qtpl:86
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/edit.qtpl:86
	return qs422016
//line template/namespace/edit.qtpl:86
}

//line template/namespace/edit.qtpl:88
func (p *EditPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/namespace/edit.qtpl:88
	qw422016.N().S(` <a class="back" href="/u/`)
	//line template/namespace/edit.qtpl:89
	qw422016.E().S(p.Namespace.User.Username)
	//line template/namespace/edit.qtpl:89
	qw422016.N().S(`/`)
	//line template/namespace/edit.qtpl:89
	qw422016.E().S(p.Namespace.FullName)
	//line template/namespace/edit.qtpl:89
	qw422016.N().S(`"> `)
	//line template/namespace/edit.qtpl:90
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
	//line template/namespace/edit.qtpl:90
	qw422016.N().S(` </a> `)
	//line template/namespace/edit.qtpl:92
	streamrenderFullName(qw422016, p.Namespace.User.Username, p.Namespace.FullName)
	//line template/namespace/edit.qtpl:92
	qw422016.N().S(` - Edit `)
//line template/namespace/edit.qtpl:93
}

//line template/namespace/edit.qtpl:93
func (p *EditPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/namespace/edit.qtpl:93
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/edit.qtpl:93
	p.StreamHeader(qw422016)
	//line template/namespace/edit.qtpl:93
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/edit.qtpl:93
}

//line template/namespace/edit.qtpl:93
func (p *EditPage) Header() string {
	//line template/namespace/edit.qtpl:93
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/edit.qtpl:93
	p.WriteHeader(qb422016)
	//line template/namespace/edit.qtpl:93
	qs422016 := string(qb422016.B)
	//line template/namespace/edit.qtpl:93
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/edit.qtpl:93
	return qs422016
//line template/namespace/edit.qtpl:93
}

//line template/namespace/edit.qtpl:95
func (p *EditPage) StreamActions(qw422016 *qt422016.Writer) {
//line template/namespace/edit.qtpl:95
}

//line template/namespace/edit.qtpl:95
func (p *EditPage) WriteActions(qq422016 qtio422016.Writer) {
	//line template/namespace/edit.qtpl:95
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/edit.qtpl:95
	p.StreamActions(qw422016)
	//line template/namespace/edit.qtpl:95
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/edit.qtpl:95
}

//line template/namespace/edit.qtpl:95
func (p *EditPage) Actions() string {
	//line template/namespace/edit.qtpl:95
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/edit.qtpl:95
	p.WriteActions(qb422016)
	//line template/namespace/edit.qtpl:95
	qs422016 := string(qb422016.B)
	//line template/namespace/edit.qtpl:95
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/edit.qtpl:95
	return qs422016
//line template/namespace/edit.qtpl:95
}

//line template/namespace/edit.qtpl:96
func (p *EditPage) StreamTabs(qw422016 *qt422016.Writer) {
//line template/namespace/edit.qtpl:96
}

//line template/namespace/edit.qtpl:96
func (p *EditPage) WriteTabs(qq422016 qtio422016.Writer) {
	//line template/namespace/edit.qtpl:96
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/edit.qtpl:96
	p.StreamTabs(qw422016)
	//line template/namespace/edit.qtpl:96
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/edit.qtpl:96
}

//line template/namespace/edit.qtpl:96
func (p *EditPage) Tabs() string {
	//line template/namespace/edit.qtpl:96
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/edit.qtpl:96
	p.WriteTabs(qb422016)
	//line template/namespace/edit.qtpl:96
	qs422016 := string(qb422016.B)
	//line template/namespace/edit.qtpl:96
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/edit.qtpl:96
	return qs422016
//line template/namespace/edit.qtpl:96
}
