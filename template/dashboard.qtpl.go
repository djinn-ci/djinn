// Code generated by qtc from "dashboard.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/dashboard.qtpl:2
package template

//line template/dashboard.qtpl:2
import (
	"net/url"

	"github.com/andrewpillar/djinn/user"
)

//line template/dashboard.qtpl:10
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/dashboard.qtpl:10
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/dashboard.qtpl:10
type Dashboard interface {
//line template/dashboard.qtpl:10
	Title() string
//line template/dashboard.qtpl:10
	StreamTitle(qw422016 *qt422016.Writer)
//line template/dashboard.qtpl:10
	WriteTitle(qq422016 qtio422016.Writer)
//line template/dashboard.qtpl:10
	Body() string
//line template/dashboard.qtpl:10
	StreamBody(qw422016 *qt422016.Writer)
//line template/dashboard.qtpl:10
	WriteBody(qq422016 qtio422016.Writer)
//line template/dashboard.qtpl:10
	Footer() string
//line template/dashboard.qtpl:10
	StreamFooter(qw422016 *qt422016.Writer)
//line template/dashboard.qtpl:10
	WriteFooter(qq422016 qtio422016.Writer)
//line template/dashboard.qtpl:10
	Actions() string
//line template/dashboard.qtpl:10
	StreamActions(qw422016 *qt422016.Writer)
//line template/dashboard.qtpl:10
	WriteActions(qq422016 qtio422016.Writer)
//line template/dashboard.qtpl:10
	Header() string
//line template/dashboard.qtpl:10
	StreamHeader(qw422016 *qt422016.Writer)
//line template/dashboard.qtpl:10
	WriteHeader(qq422016 qtio422016.Writer)
//line template/dashboard.qtpl:10
	Navigation() string
//line template/dashboard.qtpl:10
	StreamNavigation(qw422016 *qt422016.Writer)
//line template/dashboard.qtpl:10
	WriteNavigation(qq422016 qtio422016.Writer)
//line template/dashboard.qtpl:10
}

//line template/dashboard.qtpl:26
type Section interface {
//line template/dashboard.qtpl:26
	Title() string
//line template/dashboard.qtpl:26
	StreamTitle(qw422016 *qt422016.Writer)
//line template/dashboard.qtpl:26
	WriteTitle(qq422016 qtio422016.Writer)
//line template/dashboard.qtpl:26
	Header() string
//line template/dashboard.qtpl:26
	StreamHeader(qw422016 *qt422016.Writer)
//line template/dashboard.qtpl:26
	WriteHeader(qq422016 qtio422016.Writer)
//line template/dashboard.qtpl:26
	Actions() string
//line template/dashboard.qtpl:26
	StreamActions(qw422016 *qt422016.Writer)
//line template/dashboard.qtpl:26
	WriteActions(qq422016 qtio422016.Writer)
//line template/dashboard.qtpl:26
	Body() string
//line template/dashboard.qtpl:26
	StreamBody(qw422016 *qt422016.Writer)
//line template/dashboard.qtpl:26
	WriteBody(qq422016 qtio422016.Writer)
//line template/dashboard.qtpl:26
}

//line template/dashboard.qtpl:38
type baseDashboard struct {
	Dashboard

	alert Alert
	URL   *url.URL
	User  *user.User
	CSRF  string
}

type Alert struct {
	Level   level
	Message string
}

type level uint8

const (
	success level = iota
	warn
	danger
)

var (
	NamespacesURI   = "(\\/namespaces\\/?|\\/n\\/[_\\-a-zA-Z0-9.]+\\/[\\-a-zA-Z0-9\\/]*\\/?)"
	BuildsURI       = "(^\\/$|^\\/builds\\/create$|^\\/b/[_\\-a-zA-Z0-9.]+\\/[0-9]+\\/?[a-z]*)"
	SettingsURI     = "\\/settings\\/?"
	RepositoriesURI = "\\/repos\\/?"
)

func NewDashboard(d Dashboard, url *url.URL, u *user.User, a Alert, csrf string) *baseDashboard {
	return &baseDashboard{
		Dashboard: d,
		alert:     a,
		URL:       url,
		User:      u,
		CSRF:      csrf,
	}
}

func Danger(msg string) Alert {
	return Alert{
		Level:   danger,
		Message: msg,
	}
}

func Success(msg string) Alert {
	return Alert{
		Level:   success,
		Message: msg,
	}
}

func Warn(msg string) Alert {
	return Alert{
		Level:   warn,
		Message: msg,
	}
}

func (a Alert) IsZero() bool { return a.Level == level(0) && a.Message == "" }

func (l level) String() string {
	switch l {
	case success:
		return "success"
	case warn:
		return "warn"
	case danger:
		return "danger"
	default:
		return ""
	}
}

//line template/dashboard.qtpl:115
func (p *baseDashboard) StreamBody(qw422016 *qt422016.Writer) {
//line template/dashboard.qtpl:115
	qw422016.N().S(` <div class="dashboard"> <div class="dashboard-content"> `)
//line template/dashboard.qtpl:118
	if !p.alert.IsZero() {
//line template/dashboard.qtpl:118
		qw422016.N().S(` <div class="alert alert-`)
//line template/dashboard.qtpl:119
		qw422016.E().S(p.alert.Level.String())
//line template/dashboard.qtpl:119
		qw422016.N().S(`"> <div class="alert-message">`)
//line template/dashboard.qtpl:120
		qw422016.E().S(p.alert.Message)
//line template/dashboard.qtpl:120
		qw422016.N().S(`</div> <a href="`)
//line template/dashboard.qtpl:121
		qw422016.E().S(p.URL.Path)
//line template/dashboard.qtpl:121
		qw422016.N().S(`">`)
//line template/dashboard.qtpl:121
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M18.984 6.422l-5.578 5.578 5.578 5.578-1.406 1.406-5.578-5.578-5.578 5.578-1.406-1.406 5.578-5.578-5.578-5.578 1.406-1.406 5.578 5.578 5.578-5.578z"></path>
</svg>
`)
//line template/dashboard.qtpl:121
		qw422016.N().S(`</a> </div> `)
//line template/dashboard.qtpl:123
	}
//line template/dashboard.qtpl:123
	qw422016.N().S(` <div class="dashboard-wrap"> <div class="dashboard-header"> <div class="overflow"> <h1>`)
//line template/dashboard.qtpl:127
	p.Dashboard.StreamHeader(qw422016)
//line template/dashboard.qtpl:127
	qw422016.N().S(`</h1> <ul class="dashboard-actions">`)
//line template/dashboard.qtpl:128
	p.Dashboard.StreamActions(qw422016)
//line template/dashboard.qtpl:128
	qw422016.N().S(`</ul> </div> <ul class="dashboard-nav">`)
//line template/dashboard.qtpl:130
	p.Dashboard.StreamNavigation(qw422016)
//line template/dashboard.qtpl:130
	qw422016.N().S(`</ul> </div> <div class="dashboard-body">`)
//line template/dashboard.qtpl:132
	p.Dashboard.StreamBody(qw422016)
//line template/dashboard.qtpl:132
	qw422016.N().S(`</div> </div> </div> <div class="sidebar"> <div class="sidebar-header"> <div class="logo"><div class="left"></div><div class="right"></div></div> <h2>Djinn</h2> </div> `)
//line template/dashboard.qtpl:140
	if p.User.IsZero() {
//line template/dashboard.qtpl:140
		qw422016.N().S(` <div class="sidebar-auth"> <a class="login" href="/login">Login</a> <a class="register" href="/register">Register</a> </div> `)
//line template/dashboard.qtpl:145
	} else {
//line template/dashboard.qtpl:145
		qw422016.N().S(` <ul class="sidebar-nav"> <li class="sidebar-nav-header">MANAGE</li> `)
//line template/dashboard.qtpl:148
		if Match(p.URL.Path, BuildsURI) {
//line template/dashboard.qtpl:148
			qw422016.N().S(` <li><a title="Builds" href="/" class="active">`)
//line template/dashboard.qtpl:149
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M22.688 18.984c0.422 0.281 0.422 0.984-0.094 1.406l-2.297 2.297c-0.422 0.422-0.984 0.422-1.406 0l-9.094-9.094c-2.297 0.891-4.969 0.422-6.891-1.5-2.016-2.016-2.531-5.016-1.313-7.406l4.406 4.313 3-3-4.313-4.313c2.391-1.078 5.391-0.703 7.406 1.313 1.922 1.922 2.391 4.594 1.5 6.891z"></path>
</svg>
`)
//line template/dashboard.qtpl:149
			qw422016.N().S(`<span>Builds</span></a></li> `)
//line template/dashboard.qtpl:150
		} else {
//line template/dashboard.qtpl:150
			qw422016.N().S(` <li><a title="Builds" href="/">`)
//line template/dashboard.qtpl:151
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M22.688 18.984c0.422 0.281 0.422 0.984-0.094 1.406l-2.297 2.297c-0.422 0.422-0.984 0.422-1.406 0l-9.094-9.094c-2.297 0.891-4.969 0.422-6.891-1.5-2.016-2.016-2.531-5.016-1.313-7.406l4.406 4.313 3-3-4.313-4.313c2.391-1.078 5.391-0.703 7.406 1.313 1.922 1.922 2.391 4.594 1.5 6.891z"></path>
</svg>
`)
//line template/dashboard.qtpl:151
			qw422016.N().S(`<span>Builds</span></a></li> `)
//line template/dashboard.qtpl:152
		}
//line template/dashboard.qtpl:152
		qw422016.N().S(` `)
//line template/dashboard.qtpl:153
		if Match(p.URL.Path, NamespacesURI) {
//line template/dashboard.qtpl:153
			qw422016.N().S(` <li><a title="Namespaces" href="/namespaces" class="active">`)
//line template/dashboard.qtpl:154
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M9.984 3.984l2.016 2.016h8.016c1.078 0 1.969 0.938 1.969 2.016v9.984c0 1.078-0.891 2.016-1.969 2.016h-16.031c-1.078 0-1.969-0.938-1.969-2.016v-12c0-1.078 0.891-2.016 1.969-2.016h6z"></path>
</svg>
`)
//line template/dashboard.qtpl:154
			qw422016.N().S(`<span>Namespaces</span></a></li> `)
//line template/dashboard.qtpl:155
		} else {
//line template/dashboard.qtpl:155
			qw422016.N().S(` <li><a title="Namespaces" href="/namespaces">`)
//line template/dashboard.qtpl:156
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M9.984 3.984l2.016 2.016h8.016c1.078 0 1.969 0.938 1.969 2.016v9.984c0 1.078-0.891 2.016-1.969 2.016h-16.031c-1.078 0-1.969-0.938-1.969-2.016v-12c0-1.078 0.891-2.016 1.969-2.016h6z"></path>
</svg>
`)
//line template/dashboard.qtpl:156
			qw422016.N().S(`<span>Namespaces</span></a></li> `)
//line template/dashboard.qtpl:157
		}
//line template/dashboard.qtpl:157
		qw422016.N().S(` `)
//line template/dashboard.qtpl:158
		if Match(p.URL.Path, RepositoriesURI) {
//line template/dashboard.qtpl:158
			qw422016.N().S(` <li><a title="Repositories" href="/repos" class="active">`)
//line template/dashboard.qtpl:159
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 12v-8.016h-5.016v8.016l2.484-1.5zM20.016 2.016c1.078 0 1.969 0.891 1.969 1.969v12c0 1.078-0.891 2.016-1.969 2.016h-12c-1.078 0-2.016-0.938-2.016-2.016v-12c0-1.078 0.938-1.969 2.016-1.969h12zM3.984 6v14.016h14.016v1.969h-14.016c-1.078 0-1.969-0.891-1.969-1.969v-14.016h1.969z"></path>
</svg>
`)
//line template/dashboard.qtpl:159
			qw422016.N().S(`<span>Repositories</span></a></li> `)
//line template/dashboard.qtpl:160
		} else {
//line template/dashboard.qtpl:160
			qw422016.N().S(` <li><a title="Repositories" href="/repos">`)
//line template/dashboard.qtpl:161
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 12v-8.016h-5.016v8.016l2.484-1.5zM20.016 2.016c1.078 0 1.969 0.891 1.969 1.969v12c0 1.078-0.891 2.016-1.969 2.016h-12c-1.078 0-2.016-0.938-2.016-2.016v-12c0-1.078 0.938-1.969 2.016-1.969h12zM3.984 6v14.016h14.016v1.969h-14.016c-1.078 0-1.969-0.891-1.969-1.969v-14.016h1.969z"></path>
</svg>
`)
//line template/dashboard.qtpl:161
			qw422016.N().S(`<span>Repositories</span></a></li> `)
//line template/dashboard.qtpl:162
		}
//line template/dashboard.qtpl:162
		qw422016.N().S(` `)
//line template/dashboard.qtpl:163
		if Match(p.URL.Path, pattern("images")) {
//line template/dashboard.qtpl:163
			qw422016.N().S(` <li><a title="Images" href="/images" class="active">`)
//line template/dashboard.qtpl:164
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M8.484 13.5l-3.469 4.5h13.969l-4.5-6-3.469 4.5zM21 18.984c0 1.078-0.938 2.016-2.016 2.016h-13.969c-1.078 0-2.016-0.938-2.016-2.016v-13.969c0-1.078 0.938-2.016 2.016-2.016h13.969c1.078 0 2.016 0.938 2.016 2.016v13.969z"></path>
</svg>
`)
//line template/dashboard.qtpl:164
			qw422016.N().S(`<span>Images</span></a></li> `)
//line template/dashboard.qtpl:165
		} else {
//line template/dashboard.qtpl:165
			qw422016.N().S(` <li><a title="Images" href="/images">`)
//line template/dashboard.qtpl:166
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M8.484 13.5l-3.469 4.5h13.969l-4.5-6-3.469 4.5zM21 18.984c0 1.078-0.938 2.016-2.016 2.016h-13.969c-1.078 0-2.016-0.938-2.016-2.016v-13.969c0-1.078 0.938-2.016 2.016-2.016h13.969c1.078 0 2.016 0.938 2.016 2.016v13.969z"></path>
</svg>
`)
//line template/dashboard.qtpl:166
			qw422016.N().S(`<span>Images</span></a></li> `)
//line template/dashboard.qtpl:167
		}
//line template/dashboard.qtpl:167
		qw422016.N().S(` `)
//line template/dashboard.qtpl:168
		if Match(p.URL.Path, pattern("objects")) {
//line template/dashboard.qtpl:168
			qw422016.N().S(` <li><a title="Objects" href="/objects" class="active">`)
//line template/dashboard.qtpl:169
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M5.016 18h13.969v2.016h-13.969v-2.016zM9 15.984v-6h-3.984l6.984-6.984 6.984 6.984h-3.984v6h-6z"></path>
</svg>
`)
//line template/dashboard.qtpl:169
			qw422016.N().S(`<span>Objects</span></a></li> `)
//line template/dashboard.qtpl:170
		} else {
//line template/dashboard.qtpl:170
			qw422016.N().S(` <li><a title="Objects" href="/objects">`)
//line template/dashboard.qtpl:171
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M5.016 18h13.969v2.016h-13.969v-2.016zM9 15.984v-6h-3.984l6.984-6.984 6.984 6.984h-3.984v6h-6z"></path>
</svg>
`)
//line template/dashboard.qtpl:171
			qw422016.N().S(`<span>Objects</span></a></li> `)
//line template/dashboard.qtpl:172
		}
//line template/dashboard.qtpl:172
		qw422016.N().S(` `)
//line template/dashboard.qtpl:173
		if Match(p.URL.Path, pattern("variables")) {
//line template/dashboard.qtpl:173
			qw422016.N().S(` <li><a title="Variables" href="/variables" class="active">`)
//line template/dashboard.qtpl:174
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M14.578 16.594l4.641-4.594-4.641-4.594 1.406-1.406 6 6-6 6zM9.422 16.594l-1.406 1.406-6-6 6-6 1.406 1.406-4.641 4.594z"></path>
</svg>
`)
//line template/dashboard.qtpl:174
			qw422016.N().S(`<span>Variables</span></a></li> `)
//line template/dashboard.qtpl:175
		} else {
//line template/dashboard.qtpl:175
			qw422016.N().S(` <li><a title="Variables" href="/variables">`)
//line template/dashboard.qtpl:176
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M14.578 16.594l4.641-4.594-4.641-4.594 1.406-1.406 6 6-6 6zM9.422 16.594l-1.406 1.406-6-6 6-6 1.406 1.406-4.641 4.594z"></path>
</svg>
`)
//line template/dashboard.qtpl:176
			qw422016.N().S(`<span>Variables</span></a></li> `)
//line template/dashboard.qtpl:177
		}
//line template/dashboard.qtpl:177
		qw422016.N().S(` `)
//line template/dashboard.qtpl:178
		if Match(p.URL.Path, pattern("keys")) {
//line template/dashboard.qtpl:178
			qw422016.N().S(` <li><a title="Keys" href="/keys" class="active">`)
//line template/dashboard.qtpl:179
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M6.984 14.016c1.078 0 2.016-0.938 2.016-2.016s-0.938-2.016-2.016-2.016-1.969 0.938-1.969 2.016 0.891 2.016 1.969 2.016zM12.656 9.984h10.359v4.031h-2.016v3.984h-3.984v-3.984h-4.359c-0.797 2.344-3.047 3.984-5.672 3.984-3.328 0-6-2.672-6-6s2.672-6 6-6c2.625 0 4.875 1.641 5.672 3.984z"></path>
</svg>
`)
//line template/dashboard.qtpl:179
			qw422016.N().S(`<span>Keys</span></a></li> `)
//line template/dashboard.qtpl:180
		} else {
//line template/dashboard.qtpl:180
			qw422016.N().S(` <li><a title="Keys" href="/keys">`)
//line template/dashboard.qtpl:181
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M6.984 14.016c1.078 0 2.016-0.938 2.016-2.016s-0.938-2.016-2.016-2.016-1.969 0.938-1.969 2.016 0.891 2.016 1.969 2.016zM12.656 9.984h10.359v4.031h-2.016v3.984h-3.984v-3.984h-4.359c-0.797 2.344-3.047 3.984-5.672 3.984-3.328 0-6-2.672-6-6s2.672-6 6-6c2.625 0 4.875 1.641 5.672 3.984z"></path>
</svg>
`)
//line template/dashboard.qtpl:181
			qw422016.N().S(`<span>Keys</span></a></li> `)
//line template/dashboard.qtpl:182
		}
//line template/dashboard.qtpl:182
		qw422016.N().S(` <li class="sidebar-nav-header">ACCOUNT</li> `)
//line template/dashboard.qtpl:184
		if Match(p.URL.Path, SettingsURI) {
//line template/dashboard.qtpl:184
			qw422016.N().S(` <li><a title="Settings" href="/settings" class="active">`)
//line template/dashboard.qtpl:185
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12 15.516c1.922 0 3.516-1.594 3.516-3.516s-1.594-3.516-3.516-3.516-3.516 1.594-3.516 3.516 1.594 3.516 3.516 3.516zM19.453 12.984l2.109 1.641c0.188 0.141 0.234 0.422 0.094 0.656l-2.016 3.469c-0.141 0.234-0.375 0.281-0.609 0.188l-2.484-0.984c-0.516 0.375-1.078 0.75-1.688 0.984l-0.375 2.625c-0.047 0.234-0.234 0.422-0.469 0.422h-4.031c-0.234 0-0.422-0.188-0.469-0.422l-0.375-2.625c-0.609-0.234-1.172-0.563-1.688-0.984l-2.484 0.984c-0.234 0.094-0.469 0.047-0.609-0.188l-2.016-3.469c-0.141-0.234-0.094-0.516 0.094-0.656l2.109-1.641c-0.047-0.328-0.047-0.656-0.047-0.984s0-0.656 0.047-0.984l-2.109-1.641c-0.188-0.141-0.234-0.422-0.094-0.656l2.016-3.469c0.141-0.234 0.375-0.281 0.609-0.188l2.484 0.984c0.516-0.375 1.078-0.75 1.688-0.984l0.375-2.625c0.047-0.234 0.234-0.422 0.469-0.422h4.031c0.234 0 0.422 0.188 0.469 0.422l0.375 2.625c0.609 0.234 1.172 0.563 1.688 0.984l2.484-0.984c0.234-0.094 0.469-0.047 0.609 0.188l2.016 3.469c0.141 0.234 0.094 0.516-0.094 0.656l-2.109 1.641c0.047 0.328 0.047 0.656 0.047 0.984s0 0.656-0.047 0.984z"></path>
</svg>
`)
//line template/dashboard.qtpl:185
			qw422016.N().S(`<span>Settings</span></a></li> `)
//line template/dashboard.qtpl:186
		} else {
//line template/dashboard.qtpl:186
			qw422016.N().S(` <li><a title="Settings" href="/settings">`)
//line template/dashboard.qtpl:187
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12 15.516c1.922 0 3.516-1.594 3.516-3.516s-1.594-3.516-3.516-3.516-3.516 1.594-3.516 3.516 1.594 3.516 3.516 3.516zM19.453 12.984l2.109 1.641c0.188 0.141 0.234 0.422 0.094 0.656l-2.016 3.469c-0.141 0.234-0.375 0.281-0.609 0.188l-2.484-0.984c-0.516 0.375-1.078 0.75-1.688 0.984l-0.375 2.625c-0.047 0.234-0.234 0.422-0.469 0.422h-4.031c-0.234 0-0.422-0.188-0.469-0.422l-0.375-2.625c-0.609-0.234-1.172-0.563-1.688-0.984l-2.484 0.984c-0.234 0.094-0.469 0.047-0.609-0.188l-2.016-3.469c-0.141-0.234-0.094-0.516 0.094-0.656l2.109-1.641c-0.047-0.328-0.047-0.656-0.047-0.984s0-0.656 0.047-0.984l-2.109-1.641c-0.188-0.141-0.234-0.422-0.094-0.656l2.016-3.469c0.141-0.234 0.375-0.281 0.609-0.188l2.484 0.984c0.516-0.375 1.078-0.75 1.688-0.984l0.375-2.625c0.047-0.234 0.234-0.422 0.469-0.422h4.031c0.234 0 0.422 0.188 0.469 0.422l0.375 2.625c0.609 0.234 1.172 0.563 1.688 0.984l2.484-0.984c0.234-0.094 0.469-0.047 0.609 0.188l2.016 3.469c0.141 0.234 0.094 0.516-0.094 0.656l-2.109 1.641c0.047 0.328 0.047 0.656 0.047 0.984s0 0.656-0.047 0.984z"></path>
</svg>
`)
//line template/dashboard.qtpl:187
			qw422016.N().S(`<span>Settings</span></a></li> `)
//line template/dashboard.qtpl:188
		}
//line template/dashboard.qtpl:188
		qw422016.N().S(` <li> <form method="POST" action="/logout"> `)
//line template/dashboard.qtpl:191
		qw422016.N().S(string(p.CSRF))
//line template/dashboard.qtpl:191
		qw422016.N().S(` <button title="Logout" type="submit">`)
//line template/dashboard.qtpl:192
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M18.984 3c1.078 0 2.016 0.938 2.016 2.016v13.969c0 1.078-0.938 2.016-2.016 2.016h-13.969c-1.125 0-2.016-0.938-2.016-2.016v-3.984h2.016v3.984h13.969v-13.969h-13.969v3.984h-2.016v-3.984c0-1.078 0.891-2.016 2.016-2.016h13.969zM10.078 15.609l2.578-2.625h-9.656v-1.969h9.656l-2.578-2.625 1.406-1.406 5.016 5.016-5.016 5.016z"></path>
</svg>
`)
//line template/dashboard.qtpl:192
		qw422016.N().S(`<span>Logout</span></button> </form> </li> </ul> `)
//line template/dashboard.qtpl:196
	}
//line template/dashboard.qtpl:196
	qw422016.N().S(` </div> </div> `)
//line template/dashboard.qtpl:199
}

//line template/dashboard.qtpl:199
func (p *baseDashboard) WriteBody(qq422016 qtio422016.Writer) {
//line template/dashboard.qtpl:199
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/dashboard.qtpl:199
	p.StreamBody(qw422016)
//line template/dashboard.qtpl:199
	qt422016.ReleaseWriter(qw422016)
//line template/dashboard.qtpl:199
}

//line template/dashboard.qtpl:199
func (p *baseDashboard) Body() string {
//line template/dashboard.qtpl:199
	qb422016 := qt422016.AcquireByteBuffer()
//line template/dashboard.qtpl:199
	p.WriteBody(qb422016)
//line template/dashboard.qtpl:199
	qs422016 := string(qb422016.B)
//line template/dashboard.qtpl:199
	qt422016.ReleaseByteBuffer(qb422016)
//line template/dashboard.qtpl:199
	return qs422016
//line template/dashboard.qtpl:199
}
