// Code generated by qtc from "form.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line oauth2/template/form.qtpl:2
package template

//line oauth2/template/form.qtpl:2
import (
	"github.com/andrewpillar/thrall/oauth2"
	"github.com/andrewpillar/thrall/template"
)

//line oauth2/template/form.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line oauth2/template/form.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line oauth2/template/form.qtpl:9
type AppForm struct {
	template.BasePage
	template.Form
}

type TokenForm struct {
	template.BasePage
	template.Form

	Token  *oauth2.Token
	Scopes map[string]struct{}
}

//line oauth2/template/form.qtpl:24
func (p *AppForm) StreamTitle(qw422016 *qt422016.Writer) {
//line oauth2/template/form.qtpl:24
	qw422016.N().S(` AppForm OAuth App - Thrall `)
//line oauth2/template/form.qtpl:26
}

//line oauth2/template/form.qtpl:26
func (p *AppForm) WriteTitle(qq422016 qtio422016.Writer) {
//line oauth2/template/form.qtpl:26
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/form.qtpl:26
	p.StreamTitle(qw422016)
//line oauth2/template/form.qtpl:26
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/form.qtpl:26
}

//line oauth2/template/form.qtpl:26
func (p *AppForm) Title() string {
//line oauth2/template/form.qtpl:26
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/form.qtpl:26
	p.WriteTitle(qb422016)
//line oauth2/template/form.qtpl:26
	qs422016 := string(qb422016.B)
//line oauth2/template/form.qtpl:26
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/form.qtpl:26
	return qs422016
//line oauth2/template/form.qtpl:26
}

//line oauth2/template/form.qtpl:28
func (p *AppForm) StreamBody(qw422016 *qt422016.Writer) {
//line oauth2/template/form.qtpl:28
	qw422016.N().S(` `)
//line oauth2/template/form.qtpl:30
}

//line oauth2/template/form.qtpl:30
func (p *AppForm) WriteBody(qq422016 qtio422016.Writer) {
//line oauth2/template/form.qtpl:30
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/form.qtpl:30
	p.StreamBody(qw422016)
//line oauth2/template/form.qtpl:30
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/form.qtpl:30
}

//line oauth2/template/form.qtpl:30
func (p *AppForm) Body() string {
//line oauth2/template/form.qtpl:30
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/form.qtpl:30
	p.WriteBody(qb422016)
//line oauth2/template/form.qtpl:30
	qs422016 := string(qb422016.B)
//line oauth2/template/form.qtpl:30
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/form.qtpl:30
	return qs422016
//line oauth2/template/form.qtpl:30
}

//line oauth2/template/form.qtpl:32
func (p *AppForm) StreamHeader(qw422016 *qt422016.Writer) {
//line oauth2/template/form.qtpl:32
	qw422016.N().S(` `)
//line oauth2/template/form.qtpl:34
}

//line oauth2/template/form.qtpl:34
func (p *AppForm) WriteHeader(qq422016 qtio422016.Writer) {
//line oauth2/template/form.qtpl:34
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/form.qtpl:34
	p.StreamHeader(qw422016)
//line oauth2/template/form.qtpl:34
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/form.qtpl:34
}

//line oauth2/template/form.qtpl:34
func (p *AppForm) Header() string {
//line oauth2/template/form.qtpl:34
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/form.qtpl:34
	p.WriteHeader(qb422016)
//line oauth2/template/form.qtpl:34
	qs422016 := string(qb422016.B)
//line oauth2/template/form.qtpl:34
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/form.qtpl:34
	return qs422016
//line oauth2/template/form.qtpl:34
}

//line oauth2/template/form.qtpl:36
func (p *AppForm) StreamActions(qw422016 *qt422016.Writer) {
//line oauth2/template/form.qtpl:36
}

//line oauth2/template/form.qtpl:36
func (p *AppForm) WriteActions(qq422016 qtio422016.Writer) {
//line oauth2/template/form.qtpl:36
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/form.qtpl:36
	p.StreamActions(qw422016)
//line oauth2/template/form.qtpl:36
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/form.qtpl:36
}

//line oauth2/template/form.qtpl:36
func (p *AppForm) Actions() string {
//line oauth2/template/form.qtpl:36
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/form.qtpl:36
	p.WriteActions(qb422016)
//line oauth2/template/form.qtpl:36
	qs422016 := string(qb422016.B)
//line oauth2/template/form.qtpl:36
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/form.qtpl:36
	return qs422016
//line oauth2/template/form.qtpl:36
}

//line oauth2/template/form.qtpl:37
func (p *AppForm) StreamNavigation(qw422016 *qt422016.Writer) {
//line oauth2/template/form.qtpl:37
}

//line oauth2/template/form.qtpl:37
func (p *AppForm) WriteNavigation(qq422016 qtio422016.Writer) {
//line oauth2/template/form.qtpl:37
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/form.qtpl:37
	p.StreamNavigation(qw422016)
//line oauth2/template/form.qtpl:37
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/form.qtpl:37
}

//line oauth2/template/form.qtpl:37
func (p *AppForm) Navigation() string {
//line oauth2/template/form.qtpl:37
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/form.qtpl:37
	p.WriteNavigation(qb422016)
//line oauth2/template/form.qtpl:37
	qs422016 := string(qb422016.B)
//line oauth2/template/form.qtpl:37
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/form.qtpl:37
	return qs422016
//line oauth2/template/form.qtpl:37
}

//line oauth2/template/form.qtpl:39
func (p *TokenForm) StreamTitle(qw422016 *qt422016.Writer) {
//line oauth2/template/form.qtpl:39
	qw422016.N().S(` AppForm OAuth App - Thrall `)
//line oauth2/template/form.qtpl:41
}

//line oauth2/template/form.qtpl:41
func (p *TokenForm) WriteTitle(qq422016 qtio422016.Writer) {
//line oauth2/template/form.qtpl:41
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/form.qtpl:41
	p.StreamTitle(qw422016)
//line oauth2/template/form.qtpl:41
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/form.qtpl:41
}

//line oauth2/template/form.qtpl:41
func (p *TokenForm) Title() string {
//line oauth2/template/form.qtpl:41
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/form.qtpl:41
	p.WriteTitle(qb422016)
//line oauth2/template/form.qtpl:41
	qs422016 := string(qb422016.B)
//line oauth2/template/form.qtpl:41
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/form.qtpl:41
	return qs422016
//line oauth2/template/form.qtpl:41
}

//line oauth2/template/form.qtpl:43
func (p *TokenForm) StreamBody(qw422016 *qt422016.Writer) {
//line oauth2/template/form.qtpl:43
	qw422016.N().S(` `)
//line oauth2/template/form.qtpl:45
}

//line oauth2/template/form.qtpl:45
func (p *TokenForm) WriteBody(qq422016 qtio422016.Writer) {
//line oauth2/template/form.qtpl:45
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/form.qtpl:45
	p.StreamBody(qw422016)
//line oauth2/template/form.qtpl:45
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/form.qtpl:45
}

//line oauth2/template/form.qtpl:45
func (p *TokenForm) Body() string {
//line oauth2/template/form.qtpl:45
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/form.qtpl:45
	p.WriteBody(qb422016)
//line oauth2/template/form.qtpl:45
	qs422016 := string(qb422016.B)
//line oauth2/template/form.qtpl:45
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/form.qtpl:45
	return qs422016
//line oauth2/template/form.qtpl:45
}

//line oauth2/template/form.qtpl:47
func (p *TokenForm) StreamHeader(qw422016 *qt422016.Writer) {
//line oauth2/template/form.qtpl:47
	qw422016.N().S(` `)
//line oauth2/template/form.qtpl:49
}

//line oauth2/template/form.qtpl:49
func (p *TokenForm) WriteHeader(qq422016 qtio422016.Writer) {
//line oauth2/template/form.qtpl:49
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/form.qtpl:49
	p.StreamHeader(qw422016)
//line oauth2/template/form.qtpl:49
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/form.qtpl:49
}

//line oauth2/template/form.qtpl:49
func (p *TokenForm) Header() string {
//line oauth2/template/form.qtpl:49
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/form.qtpl:49
	p.WriteHeader(qb422016)
//line oauth2/template/form.qtpl:49
	qs422016 := string(qb422016.B)
//line oauth2/template/form.qtpl:49
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/form.qtpl:49
	return qs422016
//line oauth2/template/form.qtpl:49
}

//line oauth2/template/form.qtpl:51
func (p *TokenForm) StreamActions(qw422016 *qt422016.Writer) {
//line oauth2/template/form.qtpl:51
}

//line oauth2/template/form.qtpl:51
func (p *TokenForm) WriteActions(qq422016 qtio422016.Writer) {
//line oauth2/template/form.qtpl:51
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/form.qtpl:51
	p.StreamActions(qw422016)
//line oauth2/template/form.qtpl:51
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/form.qtpl:51
}

//line oauth2/template/form.qtpl:51
func (p *TokenForm) Actions() string {
//line oauth2/template/form.qtpl:51
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/form.qtpl:51
	p.WriteActions(qb422016)
//line oauth2/template/form.qtpl:51
	qs422016 := string(qb422016.B)
//line oauth2/template/form.qtpl:51
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/form.qtpl:51
	return qs422016
//line oauth2/template/form.qtpl:51
}

//line oauth2/template/form.qtpl:52
func (p *TokenForm) StreamNavigation(qw422016 *qt422016.Writer) {
//line oauth2/template/form.qtpl:52
}

//line oauth2/template/form.qtpl:52
func (p *TokenForm) WriteNavigation(qq422016 qtio422016.Writer) {
//line oauth2/template/form.qtpl:52
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/form.qtpl:52
	p.StreamNavigation(qw422016)
//line oauth2/template/form.qtpl:52
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/form.qtpl:52
}

//line oauth2/template/form.qtpl:52
func (p *TokenForm) Navigation() string {
//line oauth2/template/form.qtpl:52
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/form.qtpl:52
	p.WriteNavigation(qb422016)
//line oauth2/template/form.qtpl:52
	qs422016 := string(qb422016.B)
//line oauth2/template/form.qtpl:52
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/form.qtpl:52
	return qs422016
//line oauth2/template/form.qtpl:52
}
