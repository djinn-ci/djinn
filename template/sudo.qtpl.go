// Code generated by qtc from "sudo.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/sudo.qtpl:2
package template

//line template/sudo.qtpl:2
import (
	"djinn-ci.com/alert"
	"djinn-ci.com/template/form"
)

//line template/sudo.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/sudo.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/sudo.qtpl:9
type SudoForm struct {
	*form.Form

	Alert       alert.Alert
	Email       string
	SudoURL     string
	SudoReferer string
	SudoToken   string
}

//line template/sudo.qtpl:21
func (p *SudoForm) StreamTitle(qw422016 *qt422016.Writer) {
//line template/sudo.qtpl:21
	qw422016.N().S(` Authorize action `)
//line template/sudo.qtpl:23
}

//line template/sudo.qtpl:23
func (p *SudoForm) WriteTitle(qq422016 qtio422016.Writer) {
//line template/sudo.qtpl:23
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/sudo.qtpl:23
	p.StreamTitle(qw422016)
//line template/sudo.qtpl:23
	qt422016.ReleaseWriter(qw422016)
//line template/sudo.qtpl:23
}

//line template/sudo.qtpl:23
func (p *SudoForm) Title() string {
//line template/sudo.qtpl:23
	qb422016 := qt422016.AcquireByteBuffer()
//line template/sudo.qtpl:23
	p.WriteTitle(qb422016)
//line template/sudo.qtpl:23
	qs422016 := string(qb422016.B)
//line template/sudo.qtpl:23
	qt422016.ReleaseByteBuffer(qb422016)
//line template/sudo.qtpl:23
	return qs422016
//line template/sudo.qtpl:23
}

//line template/sudo.qtpl:25
func (p *SudoForm) StreamFooter(qw422016 *qt422016.Writer) {
//line template/sudo.qtpl:25
	qw422016.N().S(` <style type="text/css">`)
//line template/sudo.qtpl:26
	qw422016.N().S(`*{margin:0;padding:0}body{font-family:sans-serif;font-size:14px;background:#383e51;color:#fff}a{text-decoration:none;cursor:pointer}a:hover{text-decoration:underline}button{cursor:pointer}h1{font-weight:400}.btn{border:none;border-radius:3px;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji","Sego UI Symbol";padding:10px;padding-left:15px;padding-right:15px;color:#fff}.btn svg{fill:#fff}.btn:hover{text-decoration:none}.btn:disabled{cursor:not-allowed;background:#b2b2b2!important}.btn-primary{background:#61a0ea}.btn-primary:hover{background:#5090d9}.btn-danger{background:#de4141}.btn-danger:hover{background:#cd3030}.logo .left{display:inline-block;width:0;height:0;border-style:solid;border-width:25px 0 0 15px;border-color:transparent transparent transparent #fff}.logo .right{display:inline-block;width:0;height:0;border-style:solid;border-width:0 0 25px 15px;border-color:transparent transparent #fff transparent}.auth-page a{color:#66c9ff}.auth-page .auth-form{margin:0 auto;margin-top:150px;max-width:400px;padding:20px}.auth-page .auth-form .auth-header{margin-bottom:20px;text-align:center}.auth-page .auth-form .auth-header .logo{margin:0 auto;margin-bottom:20px;width:0}.auth-page .auth-form .auth-header .logo .handle{margin-left:-20px;border-style:solid;border-width:5px 0 12px 10px;border-color:transparent transparent transparent #fff}.auth-page .auth-form .auth-header .logo .lid{margin-bottom:-30px;margin-left:5px;border-style:solid;border-width:5px 0 12px 10px;border-color:transparent transparent transparent #fff}.auth-page .auth-form .auth-header .logo .lantern{margin-left:-25px;border-style:solid;border-width:25px 25px 75px 0;border-color:transparent #fff transparent transparent}.auth-page .auth-form .form-error{margin-top:3px;display:block;color:#e74848;min-height:17px}.auth-page .auth-form .form-field{margin-top:10px;width:100%}.auth-page .auth-form .form-field label{margin-bottom:3px;display:block}.auth-page .auth-form .form-field .form-text{box-sizing:border-box;width:100%;font-family:sans-serif;font-size:14px;padding:7px;outline:0;border:solid 1px rgba(255,255,255,.3);border-radius:3px;background:rgba(0,0,0,.3);color:#fff}.auth-page .auth-form .form-field .form-text:focus{border:solid 1px rgba(255,255,255,.5)}.auth-page .auth-form .form-field .btn{color:#fff;width:100%;display:block;text-align:center;box-sizing:border-box}.provider-btn svg{margin-right:5px;fill:#fff;vertical-align:middle}.provider-btn span{display:inline-block;vertical-align:middle}.provider-btn:hover{text-decoration:none}.provider-github{background:#24292e}.provider-github:hover{background:#353a3f}.provider-gitlab{background:#fa7035}.provider-gitlab:hover{background:#e65328}.alert{margin-top:15px;overflow:auto;padding:15px;border-radius:3px;text-align:left}.alert .alert-message{float:left;color:rgba(0,0,0,.6)}.alert a.alert-close{float:right;display:inline-block}.alert a.alert-close svg{width:15px;height:15px;fill:rgba(0,0,0,.4)}.alert a:hover svg{fill:rgba(0,0,0,.5)}.alert-success{background:#caf5ca;border-bottom:solid 1px #a0dfa0}.alert-warn{background:#fff3cd;border-bottom:solid 1px #d9c995}.alert-danger{background:#ffd4d4;border-bottom:solid 1px #e19e9e}.scope-list h3{margin-bottom:15px}.scope-list .scope-item{margin-top:15px;overflow:auto;border-top:solid 1px rgba(255,255,255,.4);padding:15px}.scope-list .scope-item svg{display:inline-block;margin-right:15px;float:left;fill:rgba(255,255,255,.4)}.scope-list .scope-item span{display:inline-block}.scope-list .scope-item span strong{display:block}`)
//line template/sudo.qtpl:26
	qw422016.N().S(`</style> `)
//line template/sudo.qtpl:27
}

//line template/sudo.qtpl:27
func (p *SudoForm) WriteFooter(qq422016 qtio422016.Writer) {
//line template/sudo.qtpl:27
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/sudo.qtpl:27
	p.StreamFooter(qw422016)
//line template/sudo.qtpl:27
	qt422016.ReleaseWriter(qw422016)
//line template/sudo.qtpl:27
}

//line template/sudo.qtpl:27
func (p *SudoForm) Footer() string {
//line template/sudo.qtpl:27
	qb422016 := qt422016.AcquireByteBuffer()
//line template/sudo.qtpl:27
	p.WriteFooter(qb422016)
//line template/sudo.qtpl:27
	qs422016 := string(qb422016.B)
//line template/sudo.qtpl:27
	qt422016.ReleaseByteBuffer(qb422016)
//line template/sudo.qtpl:27
	return qs422016
//line template/sudo.qtpl:27
}

//line template/sudo.qtpl:29
func (p *SudoForm) StreamBody(qw422016 *qt422016.Writer) {
//line template/sudo.qtpl:29
	qw422016.N().S(` <div class="auth-page"> <div class="auth-form"> <div class="auth-header"> `)
//line template/sudo.qtpl:33
	StreamLogo(qw422016)
//line template/sudo.qtpl:33
	qw422016.N().S(` <h1>Authorize action</h1> </div> `)
//line template/sudo.qtpl:36
	StreamAlert(qw422016, p.Alert, "")
//line template/sudo.qtpl:36
	qw422016.N().S(` <form method="POST" action="/sudo"> `)
//line template/sudo.qtpl:38
	qw422016.N().V(p.CSRF)
//line template/sudo.qtpl:38
	qw422016.N().S(` <input type="hidden" name="sudo_url" value="`)
//line template/sudo.qtpl:39
	qw422016.E().S(p.SudoURL)
//line template/sudo.qtpl:39
	qw422016.N().S(`"/> <input type="hidden" name="sudo_referer" value="`)
//line template/sudo.qtpl:40
	qw422016.E().S(p.SudoReferer)
//line template/sudo.qtpl:40
	qw422016.N().S(`"/> <input type="hidden" name="sudo_token" value="`)
//line template/sudo.qtpl:41
	qw422016.E().S(p.SudoToken)
//line template/sudo.qtpl:41
	qw422016.N().S(`"/> `)
//line template/sudo.qtpl:42
	p.StreamField(qw422016, form.Field{
		ID:   "password",
		Name: "Password",
		Type: form.Password,
	})
//line template/sudo.qtpl:46
	qw422016.N().S(` <div class="form-field"> <button type="submit" class="btn btn-primary">Authorize</button> </div> <a href="/">Back</a> </form> </div> </div> `)
//line template/sudo.qtpl:54
}

//line template/sudo.qtpl:54
func (p *SudoForm) WriteBody(qq422016 qtio422016.Writer) {
//line template/sudo.qtpl:54
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/sudo.qtpl:54
	p.StreamBody(qw422016)
//line template/sudo.qtpl:54
	qt422016.ReleaseWriter(qw422016)
//line template/sudo.qtpl:54
}

//line template/sudo.qtpl:54
func (p *SudoForm) Body() string {
//line template/sudo.qtpl:54
	qb422016 := qt422016.AcquireByteBuffer()
//line template/sudo.qtpl:54
	p.WriteBody(qb422016)
//line template/sudo.qtpl:54
	qs422016 := string(qb422016.B)
//line template/sudo.qtpl:54
	qt422016.ReleaseByteBuffer(qb422016)
//line template/sudo.qtpl:54
	return qs422016
//line template/sudo.qtpl:54
}
