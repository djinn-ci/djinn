// Code generated by qtc from "password.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line user/template/password.qtpl:2
package template

//line user/template/password.qtpl:2
import "github.com/andrewpillar/djinn/template"

//line user/template/password.qtpl:5
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line user/template/password.qtpl:5
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line user/template/password.qtpl:6
type PasswordReset struct {
	template.BasePage
	template.Form

	Alert template.Alert
}

type NewPassword struct {
	template.BasePage
	template.Form

	Token string
	Alert template.Alert
}

//line user/template/password.qtpl:23
func (p *PasswordReset) StreamTitle(qw422016 *qt422016.Writer) {
//line user/template/password.qtpl:23
	qw422016.N().S(` Reset Password - Djinn `)
//line user/template/password.qtpl:25
}

//line user/template/password.qtpl:25
func (p *PasswordReset) WriteTitle(qq422016 qtio422016.Writer) {
//line user/template/password.qtpl:25
	qw422016 := qt422016.AcquireWriter(qq422016)
//line user/template/password.qtpl:25
	p.StreamTitle(qw422016)
//line user/template/password.qtpl:25
	qt422016.ReleaseWriter(qw422016)
//line user/template/password.qtpl:25
}

//line user/template/password.qtpl:25
func (p *PasswordReset) Title() string {
//line user/template/password.qtpl:25
	qb422016 := qt422016.AcquireByteBuffer()
//line user/template/password.qtpl:25
	p.WriteTitle(qb422016)
//line user/template/password.qtpl:25
	qs422016 := string(qb422016.B)
//line user/template/password.qtpl:25
	qt422016.ReleaseByteBuffer(qb422016)
//line user/template/password.qtpl:25
	return qs422016
//line user/template/password.qtpl:25
}

//line user/template/password.qtpl:27
func (p *PasswordReset) StreamFooter(qw422016 *qt422016.Writer) {
//line user/template/password.qtpl:27
	qw422016.N().S(` <style type="text/css">`)
//line user/template/password.qtpl:28
	qw422016.N().S(`*{margin:0;padding:0}body{font-family:sans-serif;font-size:14px;background:#383e51;color:#fff}a{text-decoration:none;cursor:pointer}a:hover{text-decoration:underline}button{cursor:pointer}h1{font-weight:400}.btn{border:none;border-radius:3px;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji","Sego UI Symbol";padding:10px;padding-left:15px;padding-right:15px;color:#fff}.btn svg{fill:#fff}.btn:hover{text-decoration:none}.btn-primary{background:#61a0ea}.btn-primary:hover{background:#5090d9}.btn-danger{background:#de4141}.btn-danger:hover{background:#cd3030}.logo .left{display:inline-block;width:0;height:0;border-style:solid;border-width:25px 0 0 15px;border-color:transparent transparent transparent #fff}.logo .right{display:inline-block;width:0;height:0;border-style:solid;border-width:0 0 25px 15px;border-color:transparent transparent #fff transparent}.auth-page a{color:#66c9ff}.auth-page .auth-form{margin:0 auto;margin-top:150px;max-width:400px;padding:20px}.auth-page .auth-form .auth-header{margin-bottom:20px;text-align:center}.auth-page .auth-form .auth-header .brand{margin:0 auto}.auth-page .auth-form .auth-header .brand .left{border-width:45px 0 0 25px}.auth-page .auth-form .auth-header .brand .right{border-width:0 0 45px 25px}.auth-page .auth-form .form-error{margin-top:3px;display:block;color:#e74848;min-height:17px}.auth-page .auth-form .input-field{margin-top:10px;width:100%}.auth-page .auth-form .input-field label{margin-bottom:3px;display:block}.auth-page .auth-form .input-field .text{box-sizing:border-box;width:100%;font-family:sans-serif;font-size:14px;padding:7px;outline:0;border:solid 1px rgba(255,255,255,.3);border-radius:3px;background:rgba(0,0,0,.3);color:#fff}.auth-page .auth-form .input-field .text:focus{border:solid 1px rgba(255,255,255,.5)}.auth-page .auth-form .input-field .btn{color:#fff;width:100%;display:block;text-align:center;box-sizing:border-box}.provider-btn svg{margin-right:5px;fill:#fff;vertical-align:middle}.provider-btn span{display:inline-block;vertical-align:middle}.provider-btn:hover{text-decoration:none}.provider-github{background:#24292e}.provider-github:hover{background:#353a3f}.provider-gitlab{background:#fa7035}.provider-gitlab:hover{background:#e65328}.alert{margin-top:15px;overflow:auto;padding:15px;border-radius:3px;text-align:left}.alert .alert-message{float:left;color:rgba(0,0,0,.6)}.alert a{float:right;display:inline-block}.alert a svg{width:15px;height:15px;fill:rgba(0,0,0,.4)}.alert a:hover svg{fill:rgba(0,0,0,.5)}.alert-success{background:#caf5ca;border-bottom:solid 1px #a0dfa0}.alert-warn{background:#fff3cd;border-bottom:solid 1px #d9c995}.alert-danger{background:#ffd4d4;border-bottom:solid 1px #e19e9e}`)
//line user/template/password.qtpl:28
	qw422016.N().S(`</style> `)
//line user/template/password.qtpl:29
}

//line user/template/password.qtpl:29
func (p *PasswordReset) WriteFooter(qq422016 qtio422016.Writer) {
//line user/template/password.qtpl:29
	qw422016 := qt422016.AcquireWriter(qq422016)
//line user/template/password.qtpl:29
	p.StreamFooter(qw422016)
//line user/template/password.qtpl:29
	qt422016.ReleaseWriter(qw422016)
//line user/template/password.qtpl:29
}

//line user/template/password.qtpl:29
func (p *PasswordReset) Footer() string {
//line user/template/password.qtpl:29
	qb422016 := qt422016.AcquireByteBuffer()
//line user/template/password.qtpl:29
	p.WriteFooter(qb422016)
//line user/template/password.qtpl:29
	qs422016 := string(qb422016.B)
//line user/template/password.qtpl:29
	qt422016.ReleaseByteBuffer(qb422016)
//line user/template/password.qtpl:29
	return qs422016
//line user/template/password.qtpl:29
}

//line user/template/password.qtpl:31
func (p *PasswordReset) StreamBody(qw422016 *qt422016.Writer) {
//line user/template/password.qtpl:31
	qw422016.N().S(` <div class="auth-page"> <div class="auth-form"> <div class="auth-header"> <div class="brand"> <div class="left"></div> <div class="right"></div> </div> <h1>Reset your password</h1> `)
//line user/template/password.qtpl:40
	template.StreamRenderAlert(qw422016, p.Alert, "")
//line user/template/password.qtpl:40
	qw422016.N().S(` </div> <form method="POST" action="/password_reset"> `)
//line user/template/password.qtpl:43
	qw422016.N().S(string(p.CSRF))
//line user/template/password.qtpl:43
	qw422016.N().S(` <div class="input-field"> <label>Email</label> <input class="text" type="text" name="email" value="`)
//line user/template/password.qtpl:46
	qw422016.E().S(p.Fields["email"])
//line user/template/password.qtpl:46
	qw422016.N().S(`" autocomplete="off"/> `)
//line user/template/password.qtpl:47
	p.StreamError(qw422016, "email")
//line user/template/password.qtpl:47
	qw422016.N().S(` </div> <div class="input-field"> <button class="btn btn-primary" type="submit">Reset</button> </div> <div class="input-field">Already have an account? <a href="/login">Login</a></div> </form> </div> </div> `)
//line user/template/password.qtpl:56
}

//line user/template/password.qtpl:56
func (p *PasswordReset) WriteBody(qq422016 qtio422016.Writer) {
//line user/template/password.qtpl:56
	qw422016 := qt422016.AcquireWriter(qq422016)
//line user/template/password.qtpl:56
	p.StreamBody(qw422016)
//line user/template/password.qtpl:56
	qt422016.ReleaseWriter(qw422016)
//line user/template/password.qtpl:56
}

//line user/template/password.qtpl:56
func (p *PasswordReset) Body() string {
//line user/template/password.qtpl:56
	qb422016 := qt422016.AcquireByteBuffer()
//line user/template/password.qtpl:56
	p.WriteBody(qb422016)
//line user/template/password.qtpl:56
	qs422016 := string(qb422016.B)
//line user/template/password.qtpl:56
	qt422016.ReleaseByteBuffer(qb422016)
//line user/template/password.qtpl:56
	return qs422016
//line user/template/password.qtpl:56
}

//line user/template/password.qtpl:58
func (p *NewPassword) StreamTitle(qw422016 *qt422016.Writer) {
//line user/template/password.qtpl:58
	qw422016.N().S(` Reset Password - Djinn `)
//line user/template/password.qtpl:60
}

//line user/template/password.qtpl:60
func (p *NewPassword) WriteTitle(qq422016 qtio422016.Writer) {
//line user/template/password.qtpl:60
	qw422016 := qt422016.AcquireWriter(qq422016)
//line user/template/password.qtpl:60
	p.StreamTitle(qw422016)
//line user/template/password.qtpl:60
	qt422016.ReleaseWriter(qw422016)
//line user/template/password.qtpl:60
}

//line user/template/password.qtpl:60
func (p *NewPassword) Title() string {
//line user/template/password.qtpl:60
	qb422016 := qt422016.AcquireByteBuffer()
//line user/template/password.qtpl:60
	p.WriteTitle(qb422016)
//line user/template/password.qtpl:60
	qs422016 := string(qb422016.B)
//line user/template/password.qtpl:60
	qt422016.ReleaseByteBuffer(qb422016)
//line user/template/password.qtpl:60
	return qs422016
//line user/template/password.qtpl:60
}

//line user/template/password.qtpl:62
func (p *NewPassword) StreamFooter(qw422016 *qt422016.Writer) {
//line user/template/password.qtpl:62
	qw422016.N().S(` <style type="text/css">`)
//line user/template/password.qtpl:63
	qw422016.N().S(`*{margin:0;padding:0}body{font-family:sans-serif;font-size:14px;background:#383e51;color:#fff}a{text-decoration:none;cursor:pointer}a:hover{text-decoration:underline}button{cursor:pointer}h1{font-weight:400}.btn{border:none;border-radius:3px;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji","Sego UI Symbol";padding:10px;padding-left:15px;padding-right:15px;color:#fff}.btn svg{fill:#fff}.btn:hover{text-decoration:none}.btn-primary{background:#61a0ea}.btn-primary:hover{background:#5090d9}.btn-danger{background:#de4141}.btn-danger:hover{background:#cd3030}.logo .left{display:inline-block;width:0;height:0;border-style:solid;border-width:25px 0 0 15px;border-color:transparent transparent transparent #fff}.logo .right{display:inline-block;width:0;height:0;border-style:solid;border-width:0 0 25px 15px;border-color:transparent transparent #fff transparent}.auth-page a{color:#66c9ff}.auth-page .auth-form{margin:0 auto;margin-top:150px;max-width:400px;padding:20px}.auth-page .auth-form .auth-header{margin-bottom:20px;text-align:center}.auth-page .auth-form .auth-header .brand{margin:0 auto}.auth-page .auth-form .auth-header .brand .left{border-width:45px 0 0 25px}.auth-page .auth-form .auth-header .brand .right{border-width:0 0 45px 25px}.auth-page .auth-form .form-error{margin-top:3px;display:block;color:#e74848;min-height:17px}.auth-page .auth-form .input-field{margin-top:10px;width:100%}.auth-page .auth-form .input-field label{margin-bottom:3px;display:block}.auth-page .auth-form .input-field .text{box-sizing:border-box;width:100%;font-family:sans-serif;font-size:14px;padding:7px;outline:0;border:solid 1px rgba(255,255,255,.3);border-radius:3px;background:rgba(0,0,0,.3);color:#fff}.auth-page .auth-form .input-field .text:focus{border:solid 1px rgba(255,255,255,.5)}.auth-page .auth-form .input-field .btn{color:#fff;width:100%;display:block;text-align:center;box-sizing:border-box}.provider-btn svg{margin-right:5px;fill:#fff;vertical-align:middle}.provider-btn span{display:inline-block;vertical-align:middle}.provider-btn:hover{text-decoration:none}.provider-github{background:#24292e}.provider-github:hover{background:#353a3f}.provider-gitlab{background:#fa7035}.provider-gitlab:hover{background:#e65328}.alert{margin-top:15px;overflow:auto;padding:15px;border-radius:3px;text-align:left}.alert .alert-message{float:left;color:rgba(0,0,0,.6)}.alert a{float:right;display:inline-block}.alert a svg{width:15px;height:15px;fill:rgba(0,0,0,.4)}.alert a:hover svg{fill:rgba(0,0,0,.5)}.alert-success{background:#caf5ca;border-bottom:solid 1px #a0dfa0}.alert-warn{background:#fff3cd;border-bottom:solid 1px #d9c995}.alert-danger{background:#ffd4d4;border-bottom:solid 1px #e19e9e}`)
//line user/template/password.qtpl:63
	qw422016.N().S(`</style> `)
//line user/template/password.qtpl:64
}

//line user/template/password.qtpl:64
func (p *NewPassword) WriteFooter(qq422016 qtio422016.Writer) {
//line user/template/password.qtpl:64
	qw422016 := qt422016.AcquireWriter(qq422016)
//line user/template/password.qtpl:64
	p.StreamFooter(qw422016)
//line user/template/password.qtpl:64
	qt422016.ReleaseWriter(qw422016)
//line user/template/password.qtpl:64
}

//line user/template/password.qtpl:64
func (p *NewPassword) Footer() string {
//line user/template/password.qtpl:64
	qb422016 := qt422016.AcquireByteBuffer()
//line user/template/password.qtpl:64
	p.WriteFooter(qb422016)
//line user/template/password.qtpl:64
	qs422016 := string(qb422016.B)
//line user/template/password.qtpl:64
	qt422016.ReleaseByteBuffer(qb422016)
//line user/template/password.qtpl:64
	return qs422016
//line user/template/password.qtpl:64
}

//line user/template/password.qtpl:66
func (p *NewPassword) StreamBody(qw422016 *qt422016.Writer) {
//line user/template/password.qtpl:66
	qw422016.N().S(` <div class="auth-page"> <div class="auth-form"> <div class="auth-header"> <div class="brand"> <div class="left"></div> <div class="right"></div> </div> <h1>Reset your password</h1> `)
//line user/template/password.qtpl:75
	template.StreamRenderAlert(qw422016, p.Alert, "")
//line user/template/password.qtpl:75
	qw422016.N().S(` </div> <form method="POST" action="/new_password"> `)
//line user/template/password.qtpl:78
	qw422016.N().S(string(p.CSRF))
//line user/template/password.qtpl:78
	qw422016.N().S(` <input type="hidden" name="token" value="`)
//line user/template/password.qtpl:79
	qw422016.E().S(p.Token)
//line user/template/password.qtpl:79
	qw422016.N().S(`"/> <div class="input-field"> <label>Password</label> <input class="text" type="password" name="password" autocomplete="off"/> `)
//line user/template/password.qtpl:83
	p.StreamError(qw422016, "password")
//line user/template/password.qtpl:83
	qw422016.N().S(` </div> <div class="input-field"> <label>Verify Password</label> <input class="text" type="password" name="verify_password" autocomplete="off"/> `)
//line user/template/password.qtpl:88
	p.StreamError(qw422016, "verify_password")
//line user/template/password.qtpl:88
	qw422016.N().S(` </div> <div class="input-field"> <button class="btn btn-primary" type="submit">Update password</button> </div> </form> </div> </div> `)
//line user/template/password.qtpl:96
}

//line user/template/password.qtpl:96
func (p *NewPassword) WriteBody(qq422016 qtio422016.Writer) {
//line user/template/password.qtpl:96
	qw422016 := qt422016.AcquireWriter(qq422016)
//line user/template/password.qtpl:96
	p.StreamBody(qw422016)
//line user/template/password.qtpl:96
	qt422016.ReleaseWriter(qw422016)
//line user/template/password.qtpl:96
}

//line user/template/password.qtpl:96
func (p *NewPassword) Body() string {
//line user/template/password.qtpl:96
	qb422016 := qt422016.AcquireByteBuffer()
//line user/template/password.qtpl:96
	p.WriteBody(qb422016)
//line user/template/password.qtpl:96
	qs422016 := string(qb422016.B)
//line user/template/password.qtpl:96
	qt422016.ReleaseByteBuffer(qb422016)
//line user/template/password.qtpl:96
	return qs422016
//line user/template/password.qtpl:96
}
