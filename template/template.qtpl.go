// This file is automatically generated by qtc from "template.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/template.qtpl:2
package template

//line template/template.qtpl:2
import (
	"net/url"
	"regexp"

	"github.com/andrewpillar/thrall/form"
	"github.com/andrewpillar/thrall/model"
)

//line template/template.qtpl:12
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/template.qtpl:12
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/template.qtpl:12
type Dashboard interface {
	//line template/template.qtpl:12
	Title() string
	//line template/template.qtpl:12
	StreamTitle(qw422016 *qt422016.Writer)
	//line template/template.qtpl:12
	WriteTitle(qq422016 qtio422016.Writer)
	//line template/template.qtpl:12
	Styles() string
	//line template/template.qtpl:12
	StreamStyles(qw422016 *qt422016.Writer)
	//line template/template.qtpl:12
	WriteStyles(qq422016 qtio422016.Writer)
	//line template/template.qtpl:12
	Body() string
	//line template/template.qtpl:12
	StreamBody(qw422016 *qt422016.Writer)
	//line template/template.qtpl:12
	WriteBody(qq422016 qtio422016.Writer)
	//line template/template.qtpl:12
	Footer() string
	//line template/template.qtpl:12
	StreamFooter(qw422016 *qt422016.Writer)
	//line template/template.qtpl:12
	WriteFooter(qq422016 qtio422016.Writer)
	//line template/template.qtpl:12
	Actions() string
	//line template/template.qtpl:12
	StreamActions(qw422016 *qt422016.Writer)
	//line template/template.qtpl:12
	WriteActions(qq422016 qtio422016.Writer)
	//line template/template.qtpl:12
	Header() string
	//line template/template.qtpl:12
	StreamHeader(qw422016 *qt422016.Writer)
	//line template/template.qtpl:12
	WriteHeader(qq422016 qtio422016.Writer)
	//line template/template.qtpl:12
	Navigation() string
	//line template/template.qtpl:12
	StreamNavigation(qw422016 *qt422016.Writer)
	//line template/template.qtpl:12
	WriteNavigation(qq422016 qtio422016.Writer)
//line template/template.qtpl:12
}

//line template/template.qtpl:30
type Page interface {
	//line template/template.qtpl:30
	Title() string
	//line template/template.qtpl:30
	StreamTitle(qw422016 *qt422016.Writer)
	//line template/template.qtpl:30
	WriteTitle(qq422016 qtio422016.Writer)
	//line template/template.qtpl:30
	Styles() string
	//line template/template.qtpl:30
	StreamStyles(qw422016 *qt422016.Writer)
	//line template/template.qtpl:30
	WriteStyles(qq422016 qtio422016.Writer)
	//line template/template.qtpl:30
	Body() string
	//line template/template.qtpl:30
	StreamBody(qw422016 *qt422016.Writer)
	//line template/template.qtpl:30
	WriteBody(qq422016 qtio422016.Writer)
	//line template/template.qtpl:30
	Footer() string
	//line template/template.qtpl:30
	StreamFooter(qw422016 *qt422016.Writer)
	//line template/template.qtpl:30
	WriteFooter(qq422016 qtio422016.Writer)
//line template/template.qtpl:30
}

//line template/template.qtpl:42
type Alert struct {
	Level   level
	Message string
}

type BasePage struct {
	URL  *url.URL
	User *model.User
}

type baseDashboard struct {
	Dashboard

	alert Alert
	URL   *url.URL
}

type Form struct {
	CSRF   string
	Errors form.Errors
	Fields map[string]string
}

type level uint8

const (
	success level = iota
	warn
	danger
)

var (
	NamespacesURI = "(\\/namespaces\\/?|\\/n\\/[_\\-a-zA-Z0-9.]+\\/[\\-a-zA-Z0-9\\/]*\\/?)"
	BuildsURI     = "(^\\/$|^\\/builds\\/create$|^\\/b/[_-a-zA-Z0-9.]+/[0-9]+\\/?[a-z]*)"
	SettingsURI   = "\\/settings\\/?"
)

func pattern(name string) string {
	return "(^\\/" + name + "\\/?[a-z0-9\\/?]*$)"
}

func NewDashboard(d Dashboard, url *url.URL, a Alert) *baseDashboard {
	return &baseDashboard{
		Dashboard: d,
		alert:     a,
		URL:       url,
	}
}

func Active(condition bool) string {
	if condition {
		return "active"
	}

	return ""
}

func Danger(msg string) Alert {
	return Alert{
		Level:   danger,
		Message: msg,
	}
}

func Match(uri, pattern string) bool {
	matched, err := regexp.Match(pattern, []byte(uri))

	if err != nil {
		return false
	}

	return matched
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

func (a Alert) IsZero() bool {
	return a.Level == level(0) && a.Message == ""
}

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

//line template/template.qtpl:149
func StreamRender(qw422016 *qt422016.Writer, p Page) {
	//line template/template.qtpl:149
	qw422016.N().S(` <!DOCTYPE HTML> <html lang="en"> <head> <meta charset="utf-8"> <meta content="width=device-width, initial-scal=1" name="viewport"> <title>`)
	//line template/template.qtpl:155
	p.StreamTitle(qw422016)
	//line template/template.qtpl:155
	qw422016.N().S(`</title> `)
	//line template/template.qtpl:156
	p.StreamStyles(qw422016)
	//line template/template.qtpl:156
	qw422016.N().S(` </head> <body>`)
	//line template/template.qtpl:158
	p.StreamBody(qw422016)
	//line template/template.qtpl:158
	qw422016.N().S(`</body> <footer>`)
	//line template/template.qtpl:159
	p.StreamFooter(qw422016)
	//line template/template.qtpl:159
	qw422016.N().S(`</footer> </html> `)
//line template/template.qtpl:161
}

//line template/template.qtpl:161
func WriteRender(qq422016 qtio422016.Writer, p Page) {
	//line template/template.qtpl:161
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/template.qtpl:161
	StreamRender(qw422016, p)
	//line template/template.qtpl:161
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:161
}

//line template/template.qtpl:161
func Render(p Page) string {
	//line template/template.qtpl:161
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/template.qtpl:161
	WriteRender(qb422016, p)
	//line template/template.qtpl:161
	qs422016 := string(qb422016.B)
	//line template/template.qtpl:161
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/template.qtpl:161
	return qs422016
//line template/template.qtpl:161
}

//line template/template.qtpl:163
func (f Form) StreamError(qw422016 *qt422016.Writer, field string) {
	//line template/template.qtpl:163
	qw422016.N().S(` <div class="form-error">`)
	//line template/template.qtpl:164
	qw422016.E().S(f.Errors.First(field))
	//line template/template.qtpl:164
	qw422016.N().S(`</div> `)
//line template/template.qtpl:165
}

//line template/template.qtpl:165
func (f Form) WriteError(qq422016 qtio422016.Writer, field string) {
	//line template/template.qtpl:165
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/template.qtpl:165
	f.StreamError(qw422016, field)
	//line template/template.qtpl:165
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:165
}

//line template/template.qtpl:165
func (f Form) Error(field string) string {
	//line template/template.qtpl:165
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/template.qtpl:165
	f.WriteError(qb422016, field)
	//line template/template.qtpl:165
	qs422016 := string(qb422016.B)
	//line template/template.qtpl:165
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/template.qtpl:165
	return qs422016
//line template/template.qtpl:165
}

//line template/template.qtpl:167
func (p *BasePage) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/template.qtpl:167
	qw422016.N().S(` Thrall `)
//line template/template.qtpl:169
}

//line template/template.qtpl:169
func (p *BasePage) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/template.qtpl:169
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/template.qtpl:169
	p.StreamTitle(qw422016)
	//line template/template.qtpl:169
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:169
}

//line template/template.qtpl:169
func (p *BasePage) Title() string {
	//line template/template.qtpl:169
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/template.qtpl:169
	p.WriteTitle(qb422016)
	//line template/template.qtpl:169
	qs422016 := string(qb422016.B)
	//line template/template.qtpl:169
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/template.qtpl:169
	return qs422016
//line template/template.qtpl:169
}

//line template/template.qtpl:171
func (p *BasePage) StreamStyles(qw422016 *qt422016.Writer) {
	//line template/template.qtpl:171
	qw422016.N().S(` <link rel="stylesheet" type="text/css" href="/assets/css/main.css"> `)
//line template/template.qtpl:173
}

//line template/template.qtpl:173
func (p *BasePage) WriteStyles(qq422016 qtio422016.Writer) {
	//line template/template.qtpl:173
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/template.qtpl:173
	p.StreamStyles(qw422016)
	//line template/template.qtpl:173
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:173
}

//line template/template.qtpl:173
func (p *BasePage) Styles() string {
	//line template/template.qtpl:173
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/template.qtpl:173
	p.WriteStyles(qb422016)
	//line template/template.qtpl:173
	qs422016 := string(qb422016.B)
	//line template/template.qtpl:173
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/template.qtpl:173
	return qs422016
//line template/template.qtpl:173
}

//line template/template.qtpl:175
func (p *BasePage) StreamBody(qw422016 *qt422016.Writer) {
//line template/template.qtpl:175
}

//line template/template.qtpl:175
func (p *BasePage) WriteBody(qq422016 qtio422016.Writer) {
	//line template/template.qtpl:175
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/template.qtpl:175
	p.StreamBody(qw422016)
	//line template/template.qtpl:175
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:175
}

//line template/template.qtpl:175
func (p *BasePage) Body() string {
	//line template/template.qtpl:175
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/template.qtpl:175
	p.WriteBody(qb422016)
	//line template/template.qtpl:175
	qs422016 := string(qb422016.B)
	//line template/template.qtpl:175
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/template.qtpl:175
	return qs422016
//line template/template.qtpl:175
}

//line template/template.qtpl:176
func (p *BasePage) StreamFooter(qw422016 *qt422016.Writer) {
//line template/template.qtpl:176
}

//line template/template.qtpl:176
func (p *BasePage) WriteFooter(qq422016 qtio422016.Writer) {
	//line template/template.qtpl:176
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/template.qtpl:176
	p.StreamFooter(qw422016)
	//line template/template.qtpl:176
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:176
}

//line template/template.qtpl:176
func (p *BasePage) Footer() string {
	//line template/template.qtpl:176
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/template.qtpl:176
	p.WriteFooter(qb422016)
	//line template/template.qtpl:176
	qs422016 := string(qb422016.B)
	//line template/template.qtpl:176
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/template.qtpl:176
	return qs422016
//line template/template.qtpl:176
}

//line template/template.qtpl:178
func (p *baseDashboard) StreamBody(qw422016 *qt422016.Writer) {
	//line template/template.qtpl:178
	qw422016.N().S(` <div class="dashboard"> <div class="dashboard-content"> `)
	//line template/template.qtpl:181
	if !p.alert.IsZero() {
		//line template/template.qtpl:181
		qw422016.N().S(` <div class="alert alert-`)
		//line template/template.qtpl:182
		qw422016.E().S(p.alert.Level.String())
		//line template/template.qtpl:182
		qw422016.N().S(`"> <div class="alert-message">`)
		//line template/template.qtpl:183
		qw422016.E().S(p.alert.Message)
		//line template/template.qtpl:183
		qw422016.N().S(`</div> <a href="`)
		//line template/template.qtpl:184
		qw422016.E().S(p.URL.Path)
		//line template/template.qtpl:184
		qw422016.N().S(`">`)
		//line template/template.qtpl:184
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M18.984 6.422l-5.578 5.578 5.578 5.578-1.406 1.406-5.578-5.578-5.578 5.578-1.406-1.406 5.578-5.578-5.578-5.578 1.406-1.406 5.578 5.578 5.578-5.578z"></path>
</svg>
`)
		//line template/template.qtpl:184
		qw422016.N().S(`</a> </div> `)
		//line template/template.qtpl:186
	}
	//line template/template.qtpl:186
	qw422016.N().S(` <div class="dashboard-wrap"> <div class="dashboard-header"> <div class="overflow"> <h1>`)
	//line template/template.qtpl:190
	p.Dashboard.StreamHeader(qw422016)
	//line template/template.qtpl:190
	qw422016.N().S(`</h1> <ul class="dashboard-actions">`)
	//line template/template.qtpl:191
	p.Dashboard.StreamActions(qw422016)
	//line template/template.qtpl:191
	qw422016.N().S(`</ul> </div> <ul class="dashboard-nav">`)
	//line template/template.qtpl:193
	p.Dashboard.StreamNavigation(qw422016)
	//line template/template.qtpl:193
	qw422016.N().S(`</ul> </div> <div class="dashboard-body">`)
	//line template/template.qtpl:195
	p.Dashboard.StreamBody(qw422016)
	//line template/template.qtpl:195
	qw422016.N().S(`</div> </div> </div> <div class="sidebar"> <div class="sidebar-header"> <div class="logo"><div class="left"></div><div class="right"></div></div> <h2>Thrall</h2> </div> <ul class="sidebar-nav"> <li class="sidebar-nav-header">MANAGE</li> `)
	//line template/template.qtpl:205
	if Match(p.URL.Path, BuildsURI) {
		//line template/template.qtpl:205
		qw422016.N().S(` <li><a href="/" class="active">`)
		//line template/template.qtpl:206
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M22.688 18.984c0.422 0.281 0.422 0.984-0.094 1.406l-2.297 2.297c-0.422 0.422-0.984 0.422-1.406 0l-9.094-9.094c-2.297 0.891-4.969 0.422-6.891-1.5-2.016-2.016-2.531-5.016-1.313-7.406l4.406 4.313 3-3-4.313-4.313c2.391-1.078 5.391-0.703 7.406 1.313 1.922 1.922 2.391 4.594 1.5 6.891z"></path>
</svg>
`)
		//line template/template.qtpl:206
		qw422016.N().S(`<span>Builds</span></a></li> `)
		//line template/template.qtpl:207
	} else {
		//line template/template.qtpl:207
		qw422016.N().S(` <li><a href="/">`)
		//line template/template.qtpl:208
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M22.688 18.984c0.422 0.281 0.422 0.984-0.094 1.406l-2.297 2.297c-0.422 0.422-0.984 0.422-1.406 0l-9.094-9.094c-2.297 0.891-4.969 0.422-6.891-1.5-2.016-2.016-2.531-5.016-1.313-7.406l4.406 4.313 3-3-4.313-4.313c2.391-1.078 5.391-0.703 7.406 1.313 1.922 1.922 2.391 4.594 1.5 6.891z"></path>
</svg>
`)
		//line template/template.qtpl:208
		qw422016.N().S(`<span>Builds</span></a></li> `)
		//line template/template.qtpl:209
	}
	//line template/template.qtpl:209
	qw422016.N().S(` `)
	//line template/template.qtpl:210
	if Match(p.URL.Path, NamespacesURI) {
		//line template/template.qtpl:210
		qw422016.N().S(` <li><a href="/namespaces" class="active">`)
		//line template/template.qtpl:211
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M9.984 3.984l2.016 2.016h8.016c1.078 0 1.969 0.938 1.969 2.016v9.984c0 1.078-0.891 2.016-1.969 2.016h-16.031c-1.078 0-1.969-0.938-1.969-2.016v-12c0-1.078 0.891-2.016 1.969-2.016h6z"></path>
</svg>
`)
		//line template/template.qtpl:211
		qw422016.N().S(`<span>Namespaces</span></a></li> `)
		//line template/template.qtpl:212
	} else {
		//line template/template.qtpl:212
		qw422016.N().S(` <li><a href="/namespaces">`)
		//line template/template.qtpl:213
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M9.984 3.984l2.016 2.016h8.016c1.078 0 1.969 0.938 1.969 2.016v9.984c0 1.078-0.891 2.016-1.969 2.016h-16.031c-1.078 0-1.969-0.938-1.969-2.016v-12c0-1.078 0.891-2.016 1.969-2.016h6z"></path>
</svg>
`)
		//line template/template.qtpl:213
		qw422016.N().S(`<span>Namespaces</span></a></li> `)
		//line template/template.qtpl:214
	}
	//line template/template.qtpl:214
	qw422016.N().S(` `)
	//line template/template.qtpl:215
	if Match(p.URL.Path, pattern("objects")) {
		//line template/template.qtpl:215
		qw422016.N().S(` <li><a href="/objects" class="active">`)
		//line template/template.qtpl:216
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M5.016 18h13.969v2.016h-13.969v-2.016zM9 15.984v-6h-3.984l6.984-6.984 6.984 6.984h-3.984v6h-6z"></path>
</svg>
`)
		//line template/template.qtpl:216
		qw422016.N().S(`<span>Objects</span></a></li> `)
		//line template/template.qtpl:217
	} else {
		//line template/template.qtpl:217
		qw422016.N().S(` <li><a href="/objects">`)
		//line template/template.qtpl:218
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M5.016 18h13.969v2.016h-13.969v-2.016zM9 15.984v-6h-3.984l6.984-6.984 6.984 6.984h-3.984v6h-6z"></path>
</svg>
`)
		//line template/template.qtpl:218
		qw422016.N().S(`<span>Objects</span></a></li> `)
		//line template/template.qtpl:219
	}
	//line template/template.qtpl:219
	qw422016.N().S(` `)
	//line template/template.qtpl:220
	if Match(p.URL.Path, pattern("variables")) {
		//line template/template.qtpl:220
		qw422016.N().S(` <li><a href="/variables" class="active">`)
		//line template/template.qtpl:221
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M14.578 16.594l4.641-4.594-4.641-4.594 1.406-1.406 6 6-6 6zM9.422 16.594l-1.406 1.406-6-6 6-6 1.406 1.406-4.641 4.594z"></path>
</svg>
`)
		//line template/template.qtpl:221
		qw422016.N().S(`<span>Variables</span></a></li> `)
		//line template/template.qtpl:222
	} else {
		//line template/template.qtpl:222
		qw422016.N().S(` <li><a href="/variables">`)
		//line template/template.qtpl:223
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M14.578 16.594l4.641-4.594-4.641-4.594 1.406-1.406 6 6-6 6zM9.422 16.594l-1.406 1.406-6-6 6-6 1.406 1.406-4.641 4.594z"></path>
</svg>
`)
		//line template/template.qtpl:223
		qw422016.N().S(`<span>Variables</span></a></li> `)
		//line template/template.qtpl:224
	}
	//line template/template.qtpl:224
	qw422016.N().S(` `)
	//line template/template.qtpl:225
	if Match(p.URL.Path, pattern("keys")) {
		//line template/template.qtpl:225
		qw422016.N().S(` <li><a href="/keys" class="active">`)
		//line template/template.qtpl:226
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M6.984 14.016c1.078 0 2.016-0.938 2.016-2.016s-0.938-2.016-2.016-2.016-1.969 0.938-1.969 2.016 0.891 2.016 1.969 2.016zM12.656 9.984h10.359v4.031h-2.016v3.984h-3.984v-3.984h-4.359c-0.797 2.344-3.047 3.984-5.672 3.984-3.328 0-6-2.672-6-6s2.672-6 6-6c2.625 0 4.875 1.641 5.672 3.984z"></path>
</svg>
`)
		//line template/template.qtpl:226
		qw422016.N().S(`<span>Keys</span></a></li> `)
		//line template/template.qtpl:227
	} else {
		//line template/template.qtpl:227
		qw422016.N().S(` <li><a href="/keys">`)
		//line template/template.qtpl:228
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M6.984 14.016c1.078 0 2.016-0.938 2.016-2.016s-0.938-2.016-2.016-2.016-1.969 0.938-1.969 2.016 0.891 2.016 1.969 2.016zM12.656 9.984h10.359v4.031h-2.016v3.984h-3.984v-3.984h-4.359c-0.797 2.344-3.047 3.984-5.672 3.984-3.328 0-6-2.672-6-6s2.672-6 6-6c2.625 0 4.875 1.641 5.672 3.984z"></path>
</svg>
`)
		//line template/template.qtpl:228
		qw422016.N().S(`<span>Keys</span></a></li> `)
		//line template/template.qtpl:229
	}
	//line template/template.qtpl:229
	qw422016.N().S(` <li class="sidebar-nav-header">ACCOUNT</li> `)
	//line template/template.qtpl:231
	if Match(p.URL.Path, SettingsURI) {
		//line template/template.qtpl:231
		qw422016.N().S(` <li><a href="/settings" class="active">`)
		//line template/template.qtpl:232
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12 15.516c1.922 0 3.516-1.594 3.516-3.516s-1.594-3.516-3.516-3.516-3.516 1.594-3.516 3.516 1.594 3.516 3.516 3.516zM19.453 12.984l2.109 1.641c0.188 0.141 0.234 0.422 0.094 0.656l-2.016 3.469c-0.141 0.234-0.375 0.281-0.609 0.188l-2.484-0.984c-0.516 0.375-1.078 0.75-1.688 0.984l-0.375 2.625c-0.047 0.234-0.234 0.422-0.469 0.422h-4.031c-0.234 0-0.422-0.188-0.469-0.422l-0.375-2.625c-0.609-0.234-1.172-0.563-1.688-0.984l-2.484 0.984c-0.234 0.094-0.469 0.047-0.609-0.188l-2.016-3.469c-0.141-0.234-0.094-0.516 0.094-0.656l2.109-1.641c-0.047-0.328-0.047-0.656-0.047-0.984s0-0.656 0.047-0.984l-2.109-1.641c-0.188-0.141-0.234-0.422-0.094-0.656l2.016-3.469c0.141-0.234 0.375-0.281 0.609-0.188l2.484 0.984c0.516-0.375 1.078-0.75 1.688-0.984l0.375-2.625c0.047-0.234 0.234-0.422 0.469-0.422h4.031c0.234 0 0.422 0.188 0.469 0.422l0.375 2.625c0.609 0.234 1.172 0.563 1.688 0.984l2.484-0.984c0.234-0.094 0.469-0.047 0.609 0.188l2.016 3.469c0.141 0.234 0.094 0.516-0.094 0.656l-2.109 1.641c0.047 0.328 0.047 0.656 0.047 0.984s0 0.656-0.047 0.984z"></path>
</svg>
`)
		//line template/template.qtpl:232
		qw422016.N().S(`<span>Settings</span></a></li> `)
		//line template/template.qtpl:233
	} else {
		//line template/template.qtpl:233
		qw422016.N().S(` <li><a href="/settings">`)
		//line template/template.qtpl:234
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12 15.516c1.922 0 3.516-1.594 3.516-3.516s-1.594-3.516-3.516-3.516-3.516 1.594-3.516 3.516 1.594 3.516 3.516 3.516zM19.453 12.984l2.109 1.641c0.188 0.141 0.234 0.422 0.094 0.656l-2.016 3.469c-0.141 0.234-0.375 0.281-0.609 0.188l-2.484-0.984c-0.516 0.375-1.078 0.75-1.688 0.984l-0.375 2.625c-0.047 0.234-0.234 0.422-0.469 0.422h-4.031c-0.234 0-0.422-0.188-0.469-0.422l-0.375-2.625c-0.609-0.234-1.172-0.563-1.688-0.984l-2.484 0.984c-0.234 0.094-0.469 0.047-0.609-0.188l-2.016-3.469c-0.141-0.234-0.094-0.516 0.094-0.656l2.109-1.641c-0.047-0.328-0.047-0.656-0.047-0.984s0-0.656 0.047-0.984l-2.109-1.641c-0.188-0.141-0.234-0.422-0.094-0.656l2.016-3.469c0.141-0.234 0.375-0.281 0.609-0.188l2.484 0.984c0.516-0.375 1.078-0.75 1.688-0.984l0.375-2.625c0.047-0.234 0.234-0.422 0.469-0.422h4.031c0.234 0 0.422 0.188 0.469 0.422l0.375 2.625c0.609 0.234 1.172 0.563 1.688 0.984l2.484-0.984c0.234-0.094 0.469-0.047 0.609 0.188l2.016 3.469c0.141 0.234 0.094 0.516-0.094 0.656l-2.109 1.641c0.047 0.328 0.047 0.656 0.047 0.984s0 0.656-0.047 0.984z"></path>
</svg>
`)
		//line template/template.qtpl:234
		qw422016.N().S(`<span>Settings</span></a></li> `)
		//line template/template.qtpl:235
	}
	//line template/template.qtpl:235
	qw422016.N().S(` <li> <form method="POST" action="/logout"> <button type="submit">`)
	//line template/template.qtpl:238
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M18.984 3c1.078 0 2.016 0.938 2.016 2.016v13.969c0 1.078-0.938 2.016-2.016 2.016h-13.969c-1.125 0-2.016-0.938-2.016-2.016v-3.984h2.016v3.984h13.969v-13.969h-13.969v3.984h-2.016v-3.984c0-1.078 0.891-2.016 2.016-2.016h13.969zM10.078 15.609l2.578-2.625h-9.656v-1.969h9.656l-2.578-2.625 1.406-1.406 5.016 5.016-5.016 5.016z"></path>
</svg>
`)
	//line template/template.qtpl:238
	qw422016.N().S(`<span>Logout</span></button> </form> </li> </ul> </div> </div> `)
//line template/template.qtpl:244
}

//line template/template.qtpl:244
func (p *baseDashboard) WriteBody(qq422016 qtio422016.Writer) {
	//line template/template.qtpl:244
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/template.qtpl:244
	p.StreamBody(qw422016)
	//line template/template.qtpl:244
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:244
}

//line template/template.qtpl:244
func (p *baseDashboard) Body() string {
	//line template/template.qtpl:244
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/template.qtpl:244
	p.WriteBody(qb422016)
	//line template/template.qtpl:244
	qs422016 := string(qb422016.B)
	//line template/template.qtpl:244
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/template.qtpl:244
	return qs422016
//line template/template.qtpl:244
}

//line template/template.qtpl:247
func StreamRenderPaginator(qw422016 *qt422016.Writer, uri string, p model.Paginator) {
	//line template/template.qtpl:248
	if len(p.Pages) > 1 {
		//line template/template.qtpl:248
		qw422016.N().S(`<ul class="paginator panel">`)
		//line template/template.qtpl:250
		if p.Page == p.Prev {
			//line template/template.qtpl:250
			qw422016.N().S(`<li><a class="disabled">Previous</a></li>`)
			//line template/template.qtpl:252
		} else {
			//line template/template.qtpl:252
			qw422016.N().S(`<li><a href="`)
			//line template/template.qtpl:253
			qw422016.E().S(uri)
			//line template/template.qtpl:253
			qw422016.N().S(`?page=`)
			//line template/template.qtpl:253
			qw422016.E().V(p.Prev)
			//line template/template.qtpl:253
			qw422016.N().S(`" class="prev">Previous</a></li>`)
			//line template/template.qtpl:254
		}
		//line template/template.qtpl:255
		if p.Page == p.Next {
			//line template/template.qtpl:255
			qw422016.N().S(`<li><a class="disabled">Next</a></li>`)
			//line template/template.qtpl:257
		} else {
			//line template/template.qtpl:257
			qw422016.N().S(`<li><a href="`)
			//line template/template.qtpl:258
			qw422016.E().S(uri)
			//line template/template.qtpl:258
			qw422016.N().S(`?page=`)
			//line template/template.qtpl:258
			qw422016.E().V(p.Next)
			//line template/template.qtpl:258
			qw422016.N().S(`" class="next">Next</a></li>`)
			//line template/template.qtpl:259
		}
		//line template/template.qtpl:259
		qw422016.N().S(`</ul>`)
		//line template/template.qtpl:261
	}
//line template/template.qtpl:262
}

//line template/template.qtpl:262
func WriteRenderPaginator(qq422016 qtio422016.Writer, uri string, p model.Paginator) {
	//line template/template.qtpl:262
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/template.qtpl:262
	StreamRenderPaginator(qw422016, uri, p)
	//line template/template.qtpl:262
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:262
}

//line template/template.qtpl:262
func RenderPaginator(uri string, p model.Paginator) string {
	//line template/template.qtpl:262
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/template.qtpl:262
	WriteRenderPaginator(qb422016, uri, p)
	//line template/template.qtpl:262
	qs422016 := string(qb422016.B)
	//line template/template.qtpl:262
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/template.qtpl:262
	return qs422016
//line template/template.qtpl:262
}
