// This file is automatically generated by qtc from "register.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/auth/register.qtpl:2
package auth

//line template/auth/register.qtpl:2
import "github.com/andrewpillar/thrall/template"

//line template/auth/register.qtpl:5
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/auth/register.qtpl:5
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/auth/register.qtpl:6
type RegisterPage struct {
	template.Page
	template.Form
}

//line template/auth/register.qtpl:13
func (p *RegisterPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/auth/register.qtpl:13
	qw422016.N().S(` Register - Thrall `)
//line template/auth/register.qtpl:15
}

//line template/auth/register.qtpl:15
func (p *RegisterPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/auth/register.qtpl:15
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/auth/register.qtpl:15
	p.StreamTitle(qw422016)
	//line template/auth/register.qtpl:15
	qt422016.ReleaseWriter(qw422016)
//line template/auth/register.qtpl:15
}

//line template/auth/register.qtpl:15
func (p *RegisterPage) Title() string {
	//line template/auth/register.qtpl:15
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/auth/register.qtpl:15
	p.WriteTitle(qb422016)
	//line template/auth/register.qtpl:15
	qs422016 := string(qb422016.B)
	//line template/auth/register.qtpl:15
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/auth/register.qtpl:15
	return qs422016
//line template/auth/register.qtpl:15
}

//line template/auth/register.qtpl:17
func (p *RegisterPage) StreamStyles(qw422016 *qt422016.Writer) {
	//line template/auth/register.qtpl:17
	qw422016.N().S(` <link rel="stylesheet" type="text/css" href="/assets/css/auth.css"> `)
//line template/auth/register.qtpl:19
}

//line template/auth/register.qtpl:19
func (p *RegisterPage) WriteStyles(qq422016 qtio422016.Writer) {
	//line template/auth/register.qtpl:19
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/auth/register.qtpl:19
	p.StreamStyles(qw422016)
	//line template/auth/register.qtpl:19
	qt422016.ReleaseWriter(qw422016)
//line template/auth/register.qtpl:19
}

//line template/auth/register.qtpl:19
func (p *RegisterPage) Styles() string {
	//line template/auth/register.qtpl:19
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/auth/register.qtpl:19
	p.WriteStyles(qb422016)
	//line template/auth/register.qtpl:19
	qs422016 := string(qb422016.B)
	//line template/auth/register.qtpl:19
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/auth/register.qtpl:19
	return qs422016
//line template/auth/register.qtpl:19
}

//line template/auth/register.qtpl:21
func (p *RegisterPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/auth/register.qtpl:21
	qw422016.N().S(` <div class="auth-page"> <div class="auth-form"> <div class="auth-header"> <div class="brand"> <div class="left"></div> <div class="right"></div> </div> <h1>Signup to Thrall</h1> </div> <form method="POST" action="/register"> `)
	//line template/auth/register.qtpl:32
	qw422016.N().S(string(p.CSRF))
	//line template/auth/register.qtpl:32
	qw422016.N().S(` `)
	//line template/auth/register.qtpl:33
	if p.Errors.First("register") != "" {
		//line template/auth/register.qtpl:33
		qw422016.N().S(` <span class="error">Failed to register account: `)
		//line template/auth/register.qtpl:34
		qw422016.E().S(p.Errors.First("register"))
		//line template/auth/register.qtpl:34
		qw422016.N().S(`</span> `)
		//line template/auth/register.qtpl:35
	}
	//line template/auth/register.qtpl:35
	qw422016.N().S(` <div class="input-field"> <label>Email</label> <input class="text" type="text" name="email" value="`)
	//line template/auth/register.qtpl:38
	qw422016.E().S(p.Form.Get("email"))
	//line template/auth/register.qtpl:38
	qw422016.N().S(`" autocomplete="off"/> `)
	//line template/auth/register.qtpl:39
	p.StreamError(qw422016, "email")
	//line template/auth/register.qtpl:39
	qw422016.N().S(` </div> <div class="input-field"> <label>Username</label> <input class="text" type="text" name="username" value="`)
	//line template/auth/register.qtpl:43
	qw422016.E().S(p.Form.Get("username"))
	//line template/auth/register.qtpl:43
	qw422016.N().S(`" autocomplete="off"/> `)
	//line template/auth/register.qtpl:44
	p.StreamError(qw422016, "username")
	//line template/auth/register.qtpl:44
	qw422016.N().S(` </div> <div class="input-field"> <label>Password</label> <input class="text" type="password" name="password" autocomplete="off"/> `)
	//line template/auth/register.qtpl:49
	p.StreamError(qw422016, "password")
	//line template/auth/register.qtpl:49
	qw422016.N().S(` </div> <div class="input-field"> <label>Verify Password</label> <input class="text" type="password" name="verify_password" autocomplete="off"/> `)
	//line template/auth/register.qtpl:54
	p.StreamError(qw422016, "verify_password")
	//line template/auth/register.qtpl:54
	qw422016.N().S(` </div> <div class="input-field"> <button type="submit" class="btn btn-primary">Register</button> </div> <div class="input-field">Already have an account? <a href="/login">Login</a></div> </form> </div> </div> `)
//line template/auth/register.qtpl:63
}

//line template/auth/register.qtpl:63
func (p *RegisterPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/auth/register.qtpl:63
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/auth/register.qtpl:63
	p.StreamBody(qw422016)
	//line template/auth/register.qtpl:63
	qt422016.ReleaseWriter(qw422016)
//line template/auth/register.qtpl:63
}

//line template/auth/register.qtpl:63
func (p *RegisterPage) Body() string {
	//line template/auth/register.qtpl:63
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/auth/register.qtpl:63
	p.WriteBody(qb422016)
	//line template/auth/register.qtpl:63
	qs422016 := string(qb422016.B)
	//line template/auth/register.qtpl:63
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/auth/register.qtpl:63
	return qs422016
//line template/auth/register.qtpl:63
}
