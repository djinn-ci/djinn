// This file is automatically generated by qtc from "form.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/key/form.qtpl:2
package key

//line template/key/form.qtpl:2
import (
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

//line template/key/form.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/key/form.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/key/form.qtpl:9
type Form struct {
	template.Page
	template.Form

	Key *model.Key
}

func (p *Form) action() string {
	if p.Key == nil {
		return "/keys"
	}

	return p.Key.UIEndpoint()
}

//line template/key/form.qtpl:26
func (p *Form) StreamBody(qw422016 *qt422016.Writer) {
	//line template/key/form.qtpl:26
	qw422016.N().S(` <div class="panel"> <div class="panel-body slim"> <form method="POST" action="`)
	//line template/key/form.qtpl:29
	qw422016.E().S(p.action())
	//line template/key/form.qtpl:29
	qw422016.N().S(`"> `)
	//line template/key/form.qtpl:30
	qw422016.N().S(string(p.CSRF))
	//line template/key/form.qtpl:30
	qw422016.N().S(` `)
	//line template/key/form.qtpl:31
	if p.Key != nil {
		//line template/key/form.qtpl:31
		qw422016.N().S(` <input type="hidden" name="_method" value="PATCH"/> `)
		//line template/key/form.qtpl:33
	}
	//line template/key/form.qtpl:33
	qw422016.N().S(` `)
	//line template/key/form.qtpl:34
	if p.Key == nil {
		//line template/key/form.qtpl:34
		qw422016.N().S(` <div class="form-field"> <label class="label" for="name">Name</label> <input class="form-text" type="text" id="name" name="name" value="`)
		//line template/key/form.qtpl:37
		qw422016.E().S(p.Field("name"))
		//line template/key/form.qtpl:37
		qw422016.N().S(`" autocomplete="off"/> `)
		//line template/key/form.qtpl:38
		p.StreamError(qw422016, "name")
		//line template/key/form.qtpl:38
		qw422016.N().S(` </div> <div class="form-field"> <label class="label" for="key">Key</label> <textarea class="form-text form-code" id="key" name="key">`)
		//line template/key/form.qtpl:42
		qw422016.E().S(p.Field("key"))
		//line template/key/form.qtpl:42
		qw422016.N().S(`</textarea> `)
		//line template/key/form.qtpl:43
		p.StreamError(qw422016, "key")
		//line template/key/form.qtpl:43
		qw422016.N().S(` </div> `)
		//line template/key/form.qtpl:45
	}
	//line template/key/form.qtpl:45
	qw422016.N().S(` <div class="form-field"> <label class="label" for="config">Config <small>(optional)</small></label> `)
	//line template/key/form.qtpl:48
	if p.Key != nil {
		//line template/key/form.qtpl:48
		qw422016.N().S(` <textarea class="form-text form-code" id="config" name="config">`)
		//line template/key/form.qtpl:49
		qw422016.E().S(p.Key.Config)
		//line template/key/form.qtpl:49
		qw422016.N().S(`</textarea> `)
		//line template/key/form.qtpl:50
	} else {
		//line template/key/form.qtpl:50
		qw422016.N().S(` <textarea class="form-text form-code" id="config" name="config">`)
		//line template/key/form.qtpl:51
		qw422016.E().S(p.Field("config"))
		//line template/key/form.qtpl:51
		qw422016.N().S(`</textarea> `)
		//line template/key/form.qtpl:52
	}
	//line template/key/form.qtpl:52
	qw422016.N().S(` </div> <div class="form-field"> `)
	//line template/key/form.qtpl:55
	if p.Key != nil {
		//line template/key/form.qtpl:55
		qw422016.N().S(` <button type="submit" class="btn btn-primary">Save</button> `)
		//line template/key/form.qtpl:57
	} else {
		//line template/key/form.qtpl:57
		qw422016.N().S(` <button type="submit" class="btn btn-primary">Submit</button> `)
		//line template/key/form.qtpl:59
	}
	//line template/key/form.qtpl:59
	qw422016.N().S(` </div> </form> </div> </div> `)
//line template/key/form.qtpl:64
}

//line template/key/form.qtpl:64
func (p *Form) WriteBody(qq422016 qtio422016.Writer) {
	//line template/key/form.qtpl:64
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/key/form.qtpl:64
	p.StreamBody(qw422016)
	//line template/key/form.qtpl:64
	qt422016.ReleaseWriter(qw422016)
//line template/key/form.qtpl:64
}

//line template/key/form.qtpl:64
func (p *Form) Body() string {
	//line template/key/form.qtpl:64
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/key/form.qtpl:64
	p.WriteBody(qb422016)
	//line template/key/form.qtpl:64
	qs422016 := string(qb422016.B)
	//line template/key/form.qtpl:64
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/key/form.qtpl:64
	return qs422016
//line template/key/form.qtpl:64
}

//line template/key/form.qtpl:66
func (p *Form) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/key/form.qtpl:66
	qw422016.N().S(` <a class="back" href="/keys">`)
	//line template/key/form.qtpl:67
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
	//line template/key/form.qtpl:67
	qw422016.N().S(`</a> `)
	//line template/key/form.qtpl:68
	if p.Key != nil {
		//line template/key/form.qtpl:68
		qw422016.N().S(` `)
		//line template/key/form.qtpl:69
		qw422016.E().S(p.Key.Name)
		//line template/key/form.qtpl:69
		qw422016.N().S(` - Edit `)
		//line template/key/form.qtpl:70
	} else {
		//line template/key/form.qtpl:70
		qw422016.N().S(` Create SSH Key `)
		//line template/key/form.qtpl:72
	}
	//line template/key/form.qtpl:72
	qw422016.N().S(` `)
//line template/key/form.qtpl:73
}

//line template/key/form.qtpl:73
func (p *Form) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/key/form.qtpl:73
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/key/form.qtpl:73
	p.StreamHeader(qw422016)
	//line template/key/form.qtpl:73
	qt422016.ReleaseWriter(qw422016)
//line template/key/form.qtpl:73
}

//line template/key/form.qtpl:73
func (p *Form) Header() string {
	//line template/key/form.qtpl:73
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/key/form.qtpl:73
	p.WriteHeader(qb422016)
	//line template/key/form.qtpl:73
	qs422016 := string(qb422016.B)
	//line template/key/form.qtpl:73
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/key/form.qtpl:73
	return qs422016
//line template/key/form.qtpl:73
}

//line template/key/form.qtpl:75
func (p *Form) StreamActions(qw422016 *qt422016.Writer) {
//line template/key/form.qtpl:75
}

//line template/key/form.qtpl:75
func (p *Form) WriteActions(qq422016 qtio422016.Writer) {
	//line template/key/form.qtpl:75
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/key/form.qtpl:75
	p.StreamActions(qw422016)
	//line template/key/form.qtpl:75
	qt422016.ReleaseWriter(qw422016)
//line template/key/form.qtpl:75
}

//line template/key/form.qtpl:75
func (p *Form) Actions() string {
	//line template/key/form.qtpl:75
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/key/form.qtpl:75
	p.WriteActions(qb422016)
	//line template/key/form.qtpl:75
	qs422016 := string(qb422016.B)
	//line template/key/form.qtpl:75
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/key/form.qtpl:75
	return qs422016
//line template/key/form.qtpl:75
}

//line template/key/form.qtpl:76
func (p *Form) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/key/form.qtpl:76
}

//line template/key/form.qtpl:76
func (p *Form) WriteNavigation(qq422016 qtio422016.Writer) {
	//line template/key/form.qtpl:76
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/key/form.qtpl:76
	p.StreamNavigation(qw422016)
	//line template/key/form.qtpl:76
	qt422016.ReleaseWriter(qw422016)
//line template/key/form.qtpl:76
}

//line template/key/form.qtpl:76
func (p *Form) Navigation() string {
	//line template/key/form.qtpl:76
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/key/form.qtpl:76
	p.WriteNavigation(qb422016)
	//line template/key/form.qtpl:76
	qs422016 := string(qb422016.B)
	//line template/key/form.qtpl:76
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/key/form.qtpl:76
	return qs422016
//line template/key/form.qtpl:76
}