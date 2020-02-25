// Code generated by qtc from "form.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/token/form.qtpl:2
package token

//line template/token/form.qtpl:2
import (
	"strings"

	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/model/types"
	"github.com/andrewpillar/thrall/template"
)

//line template/token/form.qtpl:11
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/token/form.qtpl:11
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/token/form.qtpl:12
type Form struct {
	template.BasePage
	template.Form

	Token  *model.Token
	Scopes map[string]struct{}
}

func (p *Form) action() string {
	if p.Token == nil {
		return "/settings/tokens"
	}

	return p.Token.UIEndpoint()
}

func (p *Form) Field(field string) string {
	old := p.Form.Fields[field]

	if p.Token != nil {
		if old != "" {
			return old
		}

		if field == "name" {
			return p.Token.Name
		}
		return ""
	}

	return old
}

//line template/token/form.qtpl:47
func (p *Form) StreamTitle(qw422016 *qt422016.Writer) {
//line template/token/form.qtpl:47
	qw422016.N().S(` `)
//line template/token/form.qtpl:48
	if p.Token == nil {
//line template/token/form.qtpl:48
		qw422016.N().S(` Settings - New Token `)
//line template/token/form.qtpl:50
	} else {
//line template/token/form.qtpl:50
		qw422016.N().S(` Settings - Edit Token `)
//line template/token/form.qtpl:52
	}
//line template/token/form.qtpl:52
	qw422016.N().S(` `)
//line template/token/form.qtpl:53
}

//line template/token/form.qtpl:53
func (p *Form) WriteTitle(qq422016 qtio422016.Writer) {
//line template/token/form.qtpl:53
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/token/form.qtpl:53
	p.StreamTitle(qw422016)
//line template/token/form.qtpl:53
	qt422016.ReleaseWriter(qw422016)
//line template/token/form.qtpl:53
}

//line template/token/form.qtpl:53
func (p *Form) Title() string {
//line template/token/form.qtpl:53
	qb422016 := qt422016.AcquireByteBuffer()
//line template/token/form.qtpl:53
	p.WriteTitle(qb422016)
//line template/token/form.qtpl:53
	qs422016 := string(qb422016.B)
//line template/token/form.qtpl:53
	qt422016.ReleaseByteBuffer(qb422016)
//line template/token/form.qtpl:53
	return qs422016
//line template/token/form.qtpl:53
}

//line template/token/form.qtpl:55
func (p *Form) StreamHeader(qw422016 *qt422016.Writer) {
//line template/token/form.qtpl:55
	qw422016.N().S(` `)
//line template/token/form.qtpl:56
	if p.Token == nil {
//line template/token/form.qtpl:56
		qw422016.N().S(` <a href="/settings/tokens" class="back">`)
//line template/token/form.qtpl:57
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
//line template/token/form.qtpl:57
		qw422016.N().S(`</a> New Token `)
//line template/token/form.qtpl:58
	} else {
//line template/token/form.qtpl:58
		qw422016.N().S(` <a href="/settings/tokens" class="back">`)
//line template/token/form.qtpl:59
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
//line template/token/form.qtpl:59
		qw422016.N().S(`</a> Edit Token `)
//line template/token/form.qtpl:60
	}
//line template/token/form.qtpl:60
	qw422016.N().S(` `)
//line template/token/form.qtpl:61
}

//line template/token/form.qtpl:61
func (p *Form) WriteHeader(qq422016 qtio422016.Writer) {
//line template/token/form.qtpl:61
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/token/form.qtpl:61
	p.StreamHeader(qw422016)
//line template/token/form.qtpl:61
	qt422016.ReleaseWriter(qw422016)
//line template/token/form.qtpl:61
}

//line template/token/form.qtpl:61
func (p *Form) Header() string {
//line template/token/form.qtpl:61
	qb422016 := qt422016.AcquireByteBuffer()
//line template/token/form.qtpl:61
	p.WriteHeader(qb422016)
//line template/token/form.qtpl:61
	qs422016 := string(qb422016.B)
//line template/token/form.qtpl:61
	qt422016.ReleaseByteBuffer(qb422016)
//line template/token/form.qtpl:61
	return qs422016
//line template/token/form.qtpl:61
}

//line template/token/form.qtpl:63
func (p *Form) StreamBody(qw422016 *qt422016.Writer) {
//line template/token/form.qtpl:63
	qw422016.N().S(` <div class="panel"> `)
//line template/token/form.qtpl:65
	if p.Token != nil {
//line template/token/form.qtpl:65
		qw422016.N().S(` <form class="panel-body slim" method="POST" action="`)
//line template/token/form.qtpl:66
		qw422016.E().S(p.Token.UIEndpoint())
//line template/token/form.qtpl:66
		qw422016.N().S(`"> <input type="hidden" name="_method" value="PATCH"/> `)
//line template/token/form.qtpl:68
	} else {
//line template/token/form.qtpl:68
		qw422016.N().S(` <form class="panel-body slim" method="POST" action="/settings/tokens"> `)
//line template/token/form.qtpl:70
	}
//line template/token/form.qtpl:70
	qw422016.N().S(` `)
//line template/token/form.qtpl:71
	qw422016.N().S(p.CSRF)
//line template/token/form.qtpl:71
	qw422016.N().S(` <div class="form-field"> <label class="label" for="name">Name</label> <input type="text" class="form-text" id="name" name="name" value="`)
//line template/token/form.qtpl:74
	qw422016.E().S(p.Field("name"))
//line template/token/form.qtpl:74
	qw422016.N().S(`" autocomplete="off"/> `)
//line template/token/form.qtpl:75
	p.StreamError(qw422016, "name")
//line template/token/form.qtpl:75
	qw422016.N().S(` </div> `)
//line template/token/form.qtpl:77
	for _, res := range types.Resources {
//line template/token/form.qtpl:77
		qw422016.N().S(` <div class="form-field"> <label class="label">`)
//line template/token/form.qtpl:79
		qw422016.E().S(strings.Title(res.String()))
//line template/token/form.qtpl:79
		qw422016.N().S(`</label> `)
//line template/token/form.qtpl:80
		for _, perm := range types.Permissions {
//line template/token/form.qtpl:80
			qw422016.N().S(` `)
//line template/token/form.qtpl:81
			if _, ok := p.Scopes[res.String()+":"+perm.String()]; ok {
//line template/token/form.qtpl:81
				qw422016.N().S(` <label> <input type="checkbox" name="scope[]" value="`)
//line template/token/form.qtpl:83
				qw422016.E().S(res.String())
//line template/token/form.qtpl:83
				qw422016.N().S(`:`)
//line template/token/form.qtpl:83
				qw422016.E().S(perm.String())
//line template/token/form.qtpl:83
				qw422016.N().S(`" checked="true"/> `)
//line template/token/form.qtpl:83
				qw422016.E().S(strings.Title(perm.String()))
//line template/token/form.qtpl:83
				qw422016.N().S(` </label> `)
//line template/token/form.qtpl:85
			} else {
//line template/token/form.qtpl:85
				qw422016.N().S(` <label> <input type="checkbox" name="scope[]" value="`)
//line template/token/form.qtpl:87
				qw422016.E().S(res.String())
//line template/token/form.qtpl:87
				qw422016.N().S(`:`)
//line template/token/form.qtpl:87
				qw422016.E().S(perm.String())
//line template/token/form.qtpl:87
				qw422016.N().S(`"/> `)
//line template/token/form.qtpl:87
				qw422016.E().S(strings.Title(perm.String()))
//line template/token/form.qtpl:87
				qw422016.N().S(` </label> `)
//line template/token/form.qtpl:89
			}
//line template/token/form.qtpl:89
			qw422016.N().S(` `)
//line template/token/form.qtpl:90
		}
//line template/token/form.qtpl:90
		qw422016.N().S(` </div> `)
//line template/token/form.qtpl:92
	}
//line template/token/form.qtpl:92
	qw422016.N().S(` <div class="form-field"> `)
//line template/token/form.qtpl:94
	if p.Token == nil {
//line template/token/form.qtpl:94
		qw422016.N().S(` <button type="submit" class="btn btn-primary">Create</button> `)
//line template/token/form.qtpl:96
	} else {
//line template/token/form.qtpl:96
		qw422016.N().S(` <button type="submit" class="btn btn-primary">Save</button> `)
//line template/token/form.qtpl:98
	}
//line template/token/form.qtpl:98
	qw422016.N().S(` </div> </form> </div> `)
//line template/token/form.qtpl:102
}

//line template/token/form.qtpl:102
func (p *Form) WriteBody(qq422016 qtio422016.Writer) {
//line template/token/form.qtpl:102
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/token/form.qtpl:102
	p.StreamBody(qw422016)
//line template/token/form.qtpl:102
	qt422016.ReleaseWriter(qw422016)
//line template/token/form.qtpl:102
}

//line template/token/form.qtpl:102
func (p *Form) Body() string {
//line template/token/form.qtpl:102
	qb422016 := qt422016.AcquireByteBuffer()
//line template/token/form.qtpl:102
	p.WriteBody(qb422016)
//line template/token/form.qtpl:102
	qs422016 := string(qb422016.B)
//line template/token/form.qtpl:102
	qt422016.ReleaseByteBuffer(qb422016)
//line template/token/form.qtpl:102
	return qs422016
//line template/token/form.qtpl:102
}

//line template/token/form.qtpl:104
func (p *Form) StreamActions(qw422016 *qt422016.Writer) {
//line template/token/form.qtpl:104
}

//line template/token/form.qtpl:104
func (p *Form) WriteActions(qq422016 qtio422016.Writer) {
//line template/token/form.qtpl:104
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/token/form.qtpl:104
	p.StreamActions(qw422016)
//line template/token/form.qtpl:104
	qt422016.ReleaseWriter(qw422016)
//line template/token/form.qtpl:104
}

//line template/token/form.qtpl:104
func (p *Form) Actions() string {
//line template/token/form.qtpl:104
	qb422016 := qt422016.AcquireByteBuffer()
//line template/token/form.qtpl:104
	p.WriteActions(qb422016)
//line template/token/form.qtpl:104
	qs422016 := string(qb422016.B)
//line template/token/form.qtpl:104
	qt422016.ReleaseByteBuffer(qb422016)
//line template/token/form.qtpl:104
	return qs422016
//line template/token/form.qtpl:104
}

//line template/token/form.qtpl:105
func (p *Form) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/token/form.qtpl:105
}

//line template/token/form.qtpl:105
func (p *Form) WriteNavigation(qq422016 qtio422016.Writer) {
//line template/token/form.qtpl:105
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/token/form.qtpl:105
	p.StreamNavigation(qw422016)
//line template/token/form.qtpl:105
	qt422016.ReleaseWriter(qw422016)
//line template/token/form.qtpl:105
}

//line template/token/form.qtpl:105
func (p *Form) Navigation() string {
//line template/token/form.qtpl:105
	qb422016 := qt422016.AcquireByteBuffer()
//line template/token/form.qtpl:105
	p.WriteNavigation(qb422016)
//line template/token/form.qtpl:105
	qs422016 := string(qb422016.B)
//line template/token/form.qtpl:105
	qt422016.ReleaseByteBuffer(qb422016)
//line template/token/form.qtpl:105
	return qs422016
//line template/token/form.qtpl:105
}