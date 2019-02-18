// This file is automatically generated by qtc from "login.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/auth/login.qtpl:2
package auth

//line template/auth/login.qtpl:2
import (
	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/template"
)

//line template/auth/login.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/auth/login.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/auth/login.qtpl:9
type LoginPage struct {
	*template.Page

	Errors form.Errors
	Form   form.Form
}

//line template/auth/login.qtpl:17
func (p *LoginPage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/auth/login.qtpl:17
	qw422016.N().S(`
`)
	//line template/auth/login.qtpl:18
	p.Page.StreamTitle(qw422016)
	//line template/auth/login.qtpl:18
	qw422016.N().S(` - Login
`)
//line template/auth/login.qtpl:19
}

//line template/auth/login.qtpl:19
func (p *LoginPage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/auth/login.qtpl:19
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/auth/login.qtpl:19
	p.StreamTitle(qw422016)
	//line template/auth/login.qtpl:19
	qt422016.ReleaseWriter(qw422016)
//line template/auth/login.qtpl:19
}

//line template/auth/login.qtpl:19
func (p *LoginPage) Title() string {
	//line template/auth/login.qtpl:19
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/auth/login.qtpl:19
	p.WriteTitle(qb422016)
	//line template/auth/login.qtpl:19
	qs422016 := string(qb422016.B)
	//line template/auth/login.qtpl:19
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/auth/login.qtpl:19
	return qs422016
//line template/auth/login.qtpl:19
}

//line template/auth/login.qtpl:21
func (p *LoginPage) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/auth/login.qtpl:21
	qw422016.N().S(`
<link rel="stylesheet" type="text/css" href="/assets/css/auth.css">
`)
//line template/auth/login.qtpl:23
}

//line template/auth/login.qtpl:23
func (p *LoginPage) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/auth/login.qtpl:23
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/auth/login.qtpl:23
	p.StreamHeader(qw422016)
	//line template/auth/login.qtpl:23
	qt422016.ReleaseWriter(qw422016)
//line template/auth/login.qtpl:23
}

//line template/auth/login.qtpl:23
func (p *LoginPage) Header() string {
	//line template/auth/login.qtpl:23
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/auth/login.qtpl:23
	p.WriteHeader(qb422016)
	//line template/auth/login.qtpl:23
	qs422016 := string(qb422016.B)
	//line template/auth/login.qtpl:23
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/auth/login.qtpl:23
	return qs422016
//line template/auth/login.qtpl:23
}

//line template/auth/login.qtpl:25
func (p *LoginPage) StreamBody(qw422016 *qt422016.Writer) {
	//line template/auth/login.qtpl:25
	qw422016.N().S(`
<div class="auth-page">
	<div class="auth-form">
		<div class="auth-header">
			<div class="brand">
				<div class="left"></div>
				<div class="right"></div>
			</div>
			<h1>Login to Thrall</h1>
		</div>
		<form method="POST" action="/login">
			`)
	//line template/auth/login.qtpl:36
	if p.Errors.First("login") != "" {
		//line template/auth/login.qtpl:36
		qw422016.N().S(`
				<span class="error">Failed to login: `)
		//line template/auth/login.qtpl:37
		qw422016.E().S(p.Errors.First("login"))
		//line template/auth/login.qtpl:37
		qw422016.N().S(`</span>
			`)
		//line template/auth/login.qtpl:38
	}
	//line template/auth/login.qtpl:38
	qw422016.N().S(`
			<div class="input-field">
				<label>Email / Username</label>
				<input class="text" type="text" name="handle" value="`)
	//line template/auth/login.qtpl:41
	qw422016.E().S(p.Form.Get("handle"))
	//line template/auth/login.qtpl:41
	qw422016.N().S(`" autocomplete="off"/>
				<span class="error">`)
	//line template/auth/login.qtpl:42
	qw422016.E().S(p.Errors.First("handle"))
	//line template/auth/login.qtpl:42
	qw422016.N().S(`</span>
			</div>
			<div class="input-field">
				<label>Password</label>
				<input class="text" type="password" name="password" autocomplete="off"/>
				<span class="error">`)
	//line template/auth/login.qtpl:47
	qw422016.E().S(p.Errors.First("password"))
	//line template/auth/login.qtpl:47
	qw422016.N().S(`</span>
			</div>
			<div class="input-field">
				<label><input type="checkbox" name="remember_me" value="true"/> Remember Me</label>
			</div>
			<div class="input-field">
				<button type="submit" class="button button-primary">Login</button>
			</div>
			<div class="input-field">Don't have an account? <a href="/register">Register</a></div>
		</form>
	</div>
</div>
`)
//line template/auth/login.qtpl:59
}

//line template/auth/login.qtpl:59
func (p *LoginPage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/auth/login.qtpl:59
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/auth/login.qtpl:59
	p.StreamBody(qw422016)
	//line template/auth/login.qtpl:59
	qt422016.ReleaseWriter(qw422016)
//line template/auth/login.qtpl:59
}

//line template/auth/login.qtpl:59
func (p *LoginPage) Body() string {
	//line template/auth/login.qtpl:59
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/auth/login.qtpl:59
	p.WriteBody(qb422016)
	//line template/auth/login.qtpl:59
	qs422016 := string(qb422016.B)
	//line template/auth/login.qtpl:59
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/auth/login.qtpl:59
	return qs422016
//line template/auth/login.qtpl:59
}
