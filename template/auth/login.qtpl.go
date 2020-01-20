// Code generated by qtc from "login.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/auth/login.qtpl:2
package auth

//line template/auth/login.qtpl:2
import "github.com/andrewpillar/thrall/template"

//line template/auth/login.qtpl:5
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/auth/login.qtpl:5
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/auth/login.qtpl:6
type LoginPage struct {
	template.BasePage
	template.Form
}

//line template/auth/login.qtpl:13
func (p *LoginPage) StreamTitle(qw422016 *qt422016.Writer) {
//line template/auth/login.qtpl:13
	qw422016.N().S(` Login - Thrall `)
//line template/auth/login.qtpl:15
}

//line template/auth/login.qtpl:15
func (p *LoginPage) WriteTitle(qq422016 qtio422016.Writer) {
//line template/auth/login.qtpl:15
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/auth/login.qtpl:15
	p.StreamTitle(qw422016)
//line template/auth/login.qtpl:15
	qt422016.ReleaseWriter(qw422016)
//line template/auth/login.qtpl:15
}

//line template/auth/login.qtpl:15
func (p *LoginPage) Title() string {
//line template/auth/login.qtpl:15
	qb422016 := qt422016.AcquireByteBuffer()
//line template/auth/login.qtpl:15
	p.WriteTitle(qb422016)
//line template/auth/login.qtpl:15
	qs422016 := string(qb422016.B)
//line template/auth/login.qtpl:15
	qt422016.ReleaseByteBuffer(qb422016)
//line template/auth/login.qtpl:15
	return qs422016
//line template/auth/login.qtpl:15
}

//line template/auth/login.qtpl:17
func (p *LoginPage) StreamFooter(qw422016 *qt422016.Writer) {
//line template/auth/login.qtpl:17
	qw422016.N().S(` <style type="text/css">`)
//line template/auth/login.qtpl:18
	qw422016.N().S(`*{margin:0;padding:0}body{font-family:sans-serif;font-size:14px;background:#383e51;color:#fff}a{text-decoration:none;cursor:pointer}a:hover{text-decoration:underline}button{cursor:pointer}h1{font-weight:400}.btn{border:none;border-radius:3px;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji","Sego UI Symbol";padding:10px;padding-left:15px;padding-right:15px;color:#fff}.btn svg{fill:#fff}.btn:hover{text-decoration:none}.btn-primary{background:#61a0ea}.btn-primary:hover{background:#5090d9}.btn-danger{background:#de4141}.btn-danger:hover{background:#cd3030}.logo .left{display:inline-block;width:0;height:0;border-style:solid;border-width:25px 0 0 15px;border-color:transparent transparent transparent #fff}.logo .right{display:inline-block;width:0;height:0;border-style:solid;border-width:0 0 25px 15px;border-color:transparent transparent #fff transparent}.auth-page a{color:#66c9ff}.auth-page .auth-form{margin:0 auto;margin-top:150px;max-width:400px;padding:20px}.auth-page .auth-form .auth-header{margin-bottom:20px;text-align:center}.auth-page .auth-form .auth-header .brand{margin:0 auto}.auth-page .auth-form .auth-header .brand .left{border-width:45px 0 0 25px}.auth-page .auth-form .auth-header .brand .right{border-width:0 0 45px 25px}.auth-page .auth-form .form-error{margin-top:3px;display:block;color:#e74848;min-height:17px}.auth-page .auth-form .input-field{margin-top:10px;width:100%}.auth-page .auth-form .input-field label{margin-bottom:3px;display:block}.auth-page .auth-form .input-field .text{box-sizing:border-box;width:100%;font-family:sans-serif;font-size:14px;padding:7px;outline:0;border:solid 1px rgba(255,255,255,.3);border-radius:3px;background:rgba(0,0,0,.3);color:#fff}.auth-page .auth-form .input-field .text:focus{border:solid 1px rgba(255,255,255,.5)}`)
//line template/auth/login.qtpl:18
	qw422016.N().S(`</style> `)
//line template/auth/login.qtpl:19
}

//line template/auth/login.qtpl:19
func (p *LoginPage) WriteFooter(qq422016 qtio422016.Writer) {
//line template/auth/login.qtpl:19
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/auth/login.qtpl:19
	p.StreamFooter(qw422016)
//line template/auth/login.qtpl:19
	qt422016.ReleaseWriter(qw422016)
//line template/auth/login.qtpl:19
}

//line template/auth/login.qtpl:19
func (p *LoginPage) Footer() string {
//line template/auth/login.qtpl:19
	qb422016 := qt422016.AcquireByteBuffer()
//line template/auth/login.qtpl:19
	p.WriteFooter(qb422016)
//line template/auth/login.qtpl:19
	qs422016 := string(qb422016.B)
//line template/auth/login.qtpl:19
	qt422016.ReleaseByteBuffer(qb422016)
//line template/auth/login.qtpl:19
	return qs422016
//line template/auth/login.qtpl:19
}

//line template/auth/login.qtpl:21
func (p *LoginPage) StreamBody(qw422016 *qt422016.Writer) {
//line template/auth/login.qtpl:21
	qw422016.N().S(` <div class="auth-page"> <div class="auth-form"> <div class="auth-header"> <div class="brand"> <div class="left"></div> <div class="right"></div> </div> <h1>Login to Thrall</h1> </div> <form method="POST" action="/login"> `)
//line template/auth/login.qtpl:32
	qw422016.N().S(string(p.CSRF))
//line template/auth/login.qtpl:32
	qw422016.N().S(` `)
//line template/auth/login.qtpl:33
	if p.Errors.First("login") != "" {
//line template/auth/login.qtpl:33
		qw422016.N().S(` `)
//line template/auth/login.qtpl:34
		p.StreamError(qw422016, "login")
//line template/auth/login.qtpl:34
		qw422016.N().S(` `)
//line template/auth/login.qtpl:35
	}
//line template/auth/login.qtpl:35
	qw422016.N().S(` <div class="input-field"> <label>Email / Username</label> <input class="text" type="text" name="handle" value="`)
//line template/auth/login.qtpl:38
	qw422016.E().S(p.Fields["handle"])
//line template/auth/login.qtpl:38
	qw422016.N().S(`" autocomplete="off"/> `)
//line template/auth/login.qtpl:39
	p.StreamError(qw422016, "handle")
//line template/auth/login.qtpl:39
	qw422016.N().S(` </div> <div class="input-field"> <label>Password</label> <input class="text" type="password" name="password" autocomplete="off"/> `)
//line template/auth/login.qtpl:44
	p.StreamError(qw422016, "password")
//line template/auth/login.qtpl:44
	qw422016.N().S(` </div> <div class="input-field"> <button type="submit" class="btn btn-primary">Login</button> </div> <div class="input-field">Don't have an account? <a href="/register">Register</a></div> </form> </div> </div> `)
//line template/auth/login.qtpl:53
}

//line template/auth/login.qtpl:53
func (p *LoginPage) WriteBody(qq422016 qtio422016.Writer) {
//line template/auth/login.qtpl:53
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/auth/login.qtpl:53
	p.StreamBody(qw422016)
//line template/auth/login.qtpl:53
	qt422016.ReleaseWriter(qw422016)
//line template/auth/login.qtpl:53
}

//line template/auth/login.qtpl:53
func (p *LoginPage) Body() string {
//line template/auth/login.qtpl:53
	qb422016 := qt422016.AcquireByteBuffer()
//line template/auth/login.qtpl:53
	p.WriteBody(qb422016)
//line template/auth/login.qtpl:53
	qs422016 := string(qb422016.B)
//line template/auth/login.qtpl:53
	qt422016.ReleaseByteBuffer(qb422016)
//line template/auth/login.qtpl:53
	return qs422016
//line template/auth/login.qtpl:53
}
