// Code generated by qtc from "auth.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line oauth2/template/auth.qtpl:2
package template

//line oauth2/template/auth.qtpl:2
import (
	"github.com/andrewpillar/djinn/oauth2"
	"github.com/andrewpillar/djinn/template"
	"github.com/andrewpillar/djinn/user"
)

//line oauth2/template/auth.qtpl:10
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line oauth2/template/auth.qtpl:10
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line oauth2/template/auth.qtpl:11
type Auth struct {
	template.Form

	User        *user.User
	Name        string
	ClientID    string
	RedirectURI string
	State       string
	Scope       oauth2.Scope
}

//line oauth2/template/auth.qtpl:24
func (p *Auth) StreamTitle(qw422016 *qt422016.Writer) {
//line oauth2/template/auth.qtpl:24
	qw422016.N().S(` Authenticate App - Thrall `)
//line oauth2/template/auth.qtpl:26
}

//line oauth2/template/auth.qtpl:26
func (p *Auth) WriteTitle(qq422016 qtio422016.Writer) {
//line oauth2/template/auth.qtpl:26
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/auth.qtpl:26
	p.StreamTitle(qw422016)
//line oauth2/template/auth.qtpl:26
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/auth.qtpl:26
}

//line oauth2/template/auth.qtpl:26
func (p *Auth) Title() string {
//line oauth2/template/auth.qtpl:26
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/auth.qtpl:26
	p.WriteTitle(qb422016)
//line oauth2/template/auth.qtpl:26
	qs422016 := string(qb422016.B)
//line oauth2/template/auth.qtpl:26
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/auth.qtpl:26
	return qs422016
//line oauth2/template/auth.qtpl:26
}

//line oauth2/template/auth.qtpl:28
func (p *Auth) StreamFooter(qw422016 *qt422016.Writer) {
//line oauth2/template/auth.qtpl:28
	qw422016.N().S(` <style type="text/css">`)
//line oauth2/template/auth.qtpl:29
	qw422016.N().S(`*{margin:0;padding:0}body{font-family:sans-serif;font-size:14px;background:#383e51;color:#fff}a{text-decoration:none;cursor:pointer}a:hover{text-decoration:underline}button{cursor:pointer}h1{font-weight:400}.btn{border:none;border-radius:3px;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji","Sego UI Symbol";padding:10px;padding-left:15px;padding-right:15px;color:#fff}.btn svg{fill:#fff}.btn:hover{text-decoration:none}.btn-primary{background:#61a0ea}.btn-primary:hover{background:#5090d9}.btn-danger{background:#de4141}.btn-danger:hover{background:#cd3030}.logo .left{display:inline-block;width:0;height:0;border-style:solid;border-width:25px 0 0 15px;border-color:transparent transparent transparent #fff}.logo .right{display:inline-block;width:0;height:0;border-style:solid;border-width:0 0 25px 15px;border-color:transparent transparent #fff transparent}.auth-page a{color:#66c9ff}.auth-page .auth-form{margin:0 auto;margin-top:150px;max-width:400px;padding:20px}.auth-page .auth-form .auth-header{margin-bottom:20px;text-align:center}.auth-page .auth-form .auth-header .brand{margin:0 auto}.auth-page .auth-form .auth-header .brand .left{border-width:45px 0 0 25px}.auth-page .auth-form .auth-header .brand .right{border-width:0 0 45px 25px}.auth-page .auth-form .form-error{margin-top:3px;display:block;color:#e74848;min-height:17px}.auth-page .auth-form .input-field{margin-top:10px;width:100%}.auth-page .auth-form .input-field label{margin-bottom:3px;display:block}.auth-page .auth-form .input-field .text{box-sizing:border-box;width:100%;font-family:sans-serif;font-size:14px;padding:7px;outline:0;border:solid 1px rgba(255,255,255,.3);border-radius:3px;background:rgba(0,0,0,.3);color:#fff}.auth-page .auth-form .input-field .text:focus{border:solid 1px rgba(255,255,255,.5)}.auth-page .auth-form .input-field .btn{color:#fff;width:100%;display:block;text-align:center;box-sizing:border-box}.provider-btn svg{margin-right:5px;fill:#fff;vertical-align:middle}.provider-btn span{display:inline-block;vertical-align:middle}.provider-btn:hover{text-decoration:none}.provider-github{background:#24292e}.provider-github:hover{background:#353a3f}.provider-gitlab{background:#fa7035}.provider-gitlab:hover{background:#e65328}`)
//line oauth2/template/auth.qtpl:29
	qw422016.N().S(`</style> `)
//line oauth2/template/auth.qtpl:30
}

//line oauth2/template/auth.qtpl:30
func (p *Auth) WriteFooter(qq422016 qtio422016.Writer) {
//line oauth2/template/auth.qtpl:30
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/auth.qtpl:30
	p.StreamFooter(qw422016)
//line oauth2/template/auth.qtpl:30
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/auth.qtpl:30
}

//line oauth2/template/auth.qtpl:30
func (p *Auth) Footer() string {
//line oauth2/template/auth.qtpl:30
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/auth.qtpl:30
	p.WriteFooter(qb422016)
//line oauth2/template/auth.qtpl:30
	qs422016 := string(qb422016.B)
//line oauth2/template/auth.qtpl:30
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/auth.qtpl:30
	return qs422016
//line oauth2/template/auth.qtpl:30
}

//line oauth2/template/auth.qtpl:32
func (p *Auth) StreamBody(qw422016 *qt422016.Writer) {
//line oauth2/template/auth.qtpl:32
	qw422016.N().S(` <div class="auth-page"> <div class="auth-form"> <div class="auth-header"> <div class="brand"> <div class="left"></div> <div class="right"></div> </div> <h1>Authorize `)
//line oauth2/template/auth.qtpl:40
	qw422016.E().S(p.Name)
//line oauth2/template/auth.qtpl:40
	qw422016.N().S(`</h1> </div> </div> <form method="POST" action="/oauth/authorize"> `)
//line oauth2/template/auth.qtpl:44
	qw422016.N().S(p.CSRF)
//line oauth2/template/auth.qtpl:44
	qw422016.N().S(` <input type="hidden" name="client_id" value="`)
//line oauth2/template/auth.qtpl:45
	qw422016.E().S(p.ClientID)
//line oauth2/template/auth.qtpl:45
	qw422016.N().S(`"/> `)
//line oauth2/template/auth.qtpl:46
	if p.User == nil {
//line oauth2/template/auth.qtpl:46
		qw422016.N().S(` <input type="hidden" name="authenticate" value="true"/> <div class="input-field"> <label>Email / Username</label> <input class="text" type="text" name="handle" value="`)
//line oauth2/template/auth.qtpl:50
		qw422016.E().S(p.Fields["handle"])
//line oauth2/template/auth.qtpl:50
		qw422016.N().S(`" autocomplete="off"/> `)
//line oauth2/template/auth.qtpl:51
		p.StreamError(qw422016, "handle")
//line oauth2/template/auth.qtpl:51
		qw422016.N().S(` </div> <div class="input-field"> <label>Password</label> <input class="text" type="password" name="password" autocomplete="off"/> `)
//line oauth2/template/auth.qtpl:56
		p.StreamError(qw422016, "password")
//line oauth2/template/auth.qtpl:56
		qw422016.N().S(` </div> `)
//line oauth2/template/auth.qtpl:58
	}
//line oauth2/template/auth.qtpl:58
	qw422016.N().S(` <div class="input-field"> <label>Requested Scopes</label> <ul> `)
//line oauth2/template/auth.qtpl:62
	for _, sc := range p.Scope.Spread() {
//line oauth2/template/auth.qtpl:62
		qw422016.N().S(` <li>`)
//line oauth2/template/auth.qtpl:63
		qw422016.E().S(sc)
//line oauth2/template/auth.qtpl:63
		qw422016.N().S(` `)
//line oauth2/template/auth.qtpl:64
	}
//line oauth2/template/auth.qtpl:64
	qw422016.N().S(` </ul> </div> <div class="input-field"> <button type="submit" class="btn btn-primary">Authorize</button> </div> </form> </div> `)
//line oauth2/template/auth.qtpl:72
}

//line oauth2/template/auth.qtpl:72
func (p *Auth) WriteBody(qq422016 qtio422016.Writer) {
//line oauth2/template/auth.qtpl:72
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/auth.qtpl:72
	p.StreamBody(qw422016)
//line oauth2/template/auth.qtpl:72
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/auth.qtpl:72
}

//line oauth2/template/auth.qtpl:72
func (p *Auth) Body() string {
//line oauth2/template/auth.qtpl:72
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/auth.qtpl:72
	p.WriteBody(qb422016)
//line oauth2/template/auth.qtpl:72
	qs422016 := string(qb422016.B)
//line oauth2/template/auth.qtpl:72
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/auth.qtpl:72
	return qs422016
//line oauth2/template/auth.qtpl:72
}
