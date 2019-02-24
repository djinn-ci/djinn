// This file is automatically generated by qtc from "create.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/namespace/create.qtpl:2
package namespace

//line template/namespace/create.qtpl:2
import (
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/template"
)

//line template/namespace/create.qtpl:9
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/namespace/create.qtpl:9
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/namespace/create.qtpl:10
type CreatePage struct {
	*template.Page

	Errors form.Errors
	Form   form.Form
	Parent *model.Namespace
}

//line template/namespace/create.qtpl:19
func (p *CreatePage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/namespace/create.qtpl:19
	qw422016.N().S(`
`)
	//line template/namespace/create.qtpl:20
	p.Page.StreamTitle(qw422016)
	//line template/namespace/create.qtpl:20
	qw422016.N().S(` - Create Namespace
`)
//line template/namespace/create.qtpl:21
}

//line template/namespace/create.qtpl:21
func (p *CreatePage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/namespace/create.qtpl:21
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/create.qtpl:21
	p.StreamTitle(qw422016)
	//line template/namespace/create.qtpl:21
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/create.qtpl:21
}

//line template/namespace/create.qtpl:21
func (p *CreatePage) Title() string {
	//line template/namespace/create.qtpl:21
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/create.qtpl:21
	p.WriteTitle(qb422016)
	//line template/namespace/create.qtpl:21
	qs422016 := string(qb422016.B)
	//line template/namespace/create.qtpl:21
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/create.qtpl:21
	return qs422016
//line template/namespace/create.qtpl:21
}

//line template/namespace/create.qtpl:24
func (p *CreatePage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/namespace/create.qtpl:24
	qw422016.N().S(` <div class="dashboard-header"> <h1>Create Namespace</h1> </div> <div class="dashboard-body"> <div class="panel panel-slim"> <form method="POST" action="/namespaces"> `)
	//line template/namespace/create.qtpl:31
	if p.Parent != nil && !p.Parent.IsZero() {
		//line template/namespace/create.qtpl:31
		qw422016.N().S(` <input type="hidden" name="parent" value="`)
		//line template/namespace/create.qtpl:32
		qw422016.E().S(p.Parent.FullName)
		//line template/namespace/create.qtpl:32
		qw422016.N().S(`"/> `)
		//line template/namespace/create.qtpl:33
	}
	//line template/namespace/create.qtpl:33
	qw422016.N().S(` `)
	//line template/namespace/create.qtpl:34
	if p.Errors.First("namespace") != "" {
		//line template/namespace/create.qtpl:34
		qw422016.N().S(` <div class="form-error">Failed to create namespace: `)
		//line template/namespace/create.qtpl:35
		qw422016.E().S(p.Errors.First("namespace"))
		//line template/namespace/create.qtpl:35
		qw422016.N().S(`</div> `)
		//line template/namespace/create.qtpl:36
	}
	//line template/namespace/create.qtpl:36
	qw422016.N().S(` <div class="input-field"> <label class="input-field-label">Name</label> <input class="input-text" type="text" name="name" value="`)
	//line template/namespace/create.qtpl:39
	qw422016.E().S(p.Form.Get("name"))
	//line template/namespace/create.qtpl:39
	qw422016.N().S(`" autocomplete="off"/> <span class="error">`)
	//line template/namespace/create.qtpl:40
	qw422016.E().S(p.Errors.First("name"))
	//line template/namespace/create.qtpl:40
	qw422016.N().S(`</span> </div> <div class="input-field"> <label class="input-field-label">Description</label> <textarea class="input-text" name="description"></textarea> </div> <div class="input-field"> <label class="input-field-label">Visibility</label> <label class="input-option"> <input class="input-option-selector" type="radio" name="visibility" value="private" checked="true"/> <div class="input-option-description"> Private<br/> Only you will be able to view builds in the namespace. </div> </label> <label class="input-option"> <input class="input-option-selector" type="radio" name="visibility" value="internal"/> <div class="input-option-description"> Internal<br/> Anyone with an account will be able to view builds in the namespace </div> </label> <label class="input-option"> <input class="input-option-selector" type="radio" name="visibility" value="public"/> <div class="input-option-description"> Public<br/> Anyone will be able to view builds in the namespace. </div> </label> </div> <div class="input-field"> <button type="submit" class="button button-primary">Create</button> </div> </form> </div> </div> `)
//line template/namespace/create.qtpl:76
}

//line template/namespace/create.qtpl:76
func (p *CreatePage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/namespace/create.qtpl:76
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/create.qtpl:76
	p.StreamBody(qw422016)
	//line template/namespace/create.qtpl:76
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/create.qtpl:76
}

//line template/namespace/create.qtpl:76
func (p *CreatePage) Body() string {
	//line template/namespace/create.qtpl:76
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/create.qtpl:76
	p.WriteBody(qb422016)
	//line template/namespace/create.qtpl:76
	qs422016 := string(qb422016.B)
	//line template/namespace/create.qtpl:76
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/create.qtpl:76
	return qs422016
//line template/namespace/create.qtpl:76
}
