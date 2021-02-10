// Code generated by qtc from "template.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/template.qtpl:2
package template

//line template/template.qtpl:2
import (
	"html/template"
	"net/url"
	"regexp"
	"strconv"

	"github.com/andrewpillar/djinn/database"
	"github.com/andrewpillar/djinn/user"

	"github.com/andrewpillar/webutil"
)

//line template/template.qtpl:16
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/template.qtpl:16
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/template.qtpl:16
type Page interface {
//line template/template.qtpl:16
	Title() string
//line template/template.qtpl:16
	StreamTitle(qw422016 *qt422016.Writer)
//line template/template.qtpl:16
	WriteTitle(qq422016 qtio422016.Writer)
//line template/template.qtpl:16
	Body() string
//line template/template.qtpl:16
	StreamBody(qw422016 *qt422016.Writer)
//line template/template.qtpl:16
	WriteBody(qq422016 qtio422016.Writer)
//line template/template.qtpl:16
	Footer() string
//line template/template.qtpl:16
	StreamFooter(qw422016 *qt422016.Writer)
//line template/template.qtpl:16
	WriteFooter(qq422016 qtio422016.Writer)
//line template/template.qtpl:16
}

//line template/template.qtpl:26
type BasePage struct {
	URL  *url.URL
	User *user.User
}

type Form struct {
	CSRF   template.HTML
	Errors *webutil.Errors
	Fields map[string]string
}

func pattern(name string) string { return "(^\\/" + name + "\\/?[a-z0-9\\/?]*$)" }

func Active(condition bool) string {
	if condition {
		return "active"
	}
	return ""
}

func Match(uri, pattern string) bool {
	matched, err := regexp.Match(pattern, []byte(uri))

	if err != nil {
		return false
	}
	return matched
}

func pageURL(url *url.URL, page int64) string {
	q := url.Query()
	q.Set("page", strconv.FormatInt(page, 10))

	url.RawQuery = q.Encode()
	return url.String()
}

//line template/template.qtpl:65
func StreamFileInput(qw422016 *qt422016.Writer, f Form) {
//line template/template.qtpl:65
	qw422016.N().S(` <div class="form-field"> <label class="label" for="file">File</label> <input type="file" id="file" name="file"/> `)
//line template/template.qtpl:69
	f.StreamError(qw422016, "file")
//line template/template.qtpl:69
	qw422016.N().S(` </div> `)
//line template/template.qtpl:71
}

//line template/template.qtpl:71
func WriteFileInput(qq422016 qtio422016.Writer, f Form) {
//line template/template.qtpl:71
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/template.qtpl:71
	StreamFileInput(qw422016, f)
//line template/template.qtpl:71
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:71
}

//line template/template.qtpl:71
func FileInput(f Form) string {
//line template/template.qtpl:71
	qb422016 := qt422016.AcquireByteBuffer()
//line template/template.qtpl:71
	WriteFileInput(qb422016, f)
//line template/template.qtpl:71
	qs422016 := string(qb422016.B)
//line template/template.qtpl:71
	qt422016.ReleaseByteBuffer(qb422016)
//line template/template.qtpl:71
	return qs422016
//line template/template.qtpl:71
}

//line template/template.qtpl:73
func StreamRender(qw422016 *qt422016.Writer, p Page) {
//line template/template.qtpl:73
	qw422016.N().S(` <!DOCTYPE HTML> <html lang="en"> <head> <meta charset="utf-8"> <meta content="width=device-width, initial-scale=1" name="viewport"> <title>`)
//line template/template.qtpl:79
	p.StreamTitle(qw422016)
//line template/template.qtpl:79
	qw422016.N().S(`</title> <style type="text/css">`)
//line template/template.qtpl:80
	qw422016.N().S(`*{margin:0;padding:0}body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji","Sego UI Symbol";font-size:14px;background:#eee;color:#444}a{color:#146de0;cursor:pointer;text-decoration:none}a:hover{text-decoration:underline}button{cursor:pointer}h1,h2,h3,h4,h5,h6{font-weight:400}.btn{border:none;border-radius:3px;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji","Sego UI Symbol";padding:10px;padding-left:15px;padding-right:15px;color:#fff}.btn svg{fill:#fff}.btn:hover{text-decoration:none}.btn:disabled{cursor:not-allowed;background:#b2b2b2!important}.btn-primary{background:#61a0ea}.btn-primary:hover{background:#5090d9}.btn-danger{background:#de4141}.btn-danger:hover{background:#cd3030}.code{padding:3px;border-radius:3px;background:#e6f0f5;font-family:monospace}.code-wrap{overflow-x:auto;overflow-y:hidden}table.code{background:#272b39;color:#fff;width:100%;font-family:monospace;font-size:12px;border-collapse:collapse;border-spacing:0}table.code .line-number{text-align:right;-moz-user-select:none;-ms-user-select:none;-webkit-user-select:none;min-width:30px;width:1%;padding-left:10px;padding-right:10px;line-height:20px}table.code .line-number a{display:block;color:rgba(255,255,255,.3)}table.code .line{padding-left:10px;padding-right:10px;line-height:20px;white-space:pre;overflow:visible;word-wrap:normal}table.code .line:target{background:#383e51}.col-75{width:75%;box-sizing:border-box}.col-25{width:25%;box-sizing:border-box}.col-left{float:left;padding-right:5px}.col-right{float:right;padding-left:5px}@media (max-width:1100px){.col-75{margin-bottom:10px;width:100%}.col-25{margin-bottom:10px;width:100%}.col-left{padding-right:0;float:none}.col-right{padding-left:0;float:none}}.dashboard .sidebar{position:fixed;top:0;left:0;height:100%;width:225px;background:#383e51}.dashboard .sidebar .sidebar-header{color:#fff;padding:20px;background:#272b39}.dashboard .sidebar .sidebar-header .logo{margin-top:-5px;margin-right:20px;display:inline-block;vertical-align:middle;width:0}.dashboard .sidebar .sidebar-header .logo .handle{margin-left:-3px;border-style:solid;border-width:2px 0 8px 7px;border-color:transparent transparent transparent #fff}.dashboard .sidebar .sidebar-header .logo .lid{margin-bottom:-20px;margin-left:13px;border-style:solid;border-width:5px 0 7px 5px;border-color:transparent transparent transparent #fff}.dashboard .sidebar .sidebar-header .logo .lantern{margin-left:-5px;border-style:solid;border-width:15px 15px 35px 0;border-color:transparent #fff transparent transparent}.dashboard .sidebar .sidebar-header .logo .left{margin-left:3px;display:inline-block;width:0;height:0;border-style:solid;border-width:25px 0 0 15px;border-color:transparent transparent transparent #fff}.dashboard .sidebar .sidebar-header .logo .right{margin-left:3px;display:inline-block;width:0;height:0;border-style:solid;border-width:0 0 25px 15px;border-color:transparent transparent #fff transparent}.dashboard .sidebar .sidebar-header h2{display:inline-block}.dashboard .sidebar .sidebar-auth a{display:block;color:rgba(255,255,255,.5);padding:15px;text-align:center}.dashboard .sidebar .sidebar-auth a.active,.dashboard .sidebar .sidebar-auth a:hover,.dashboard .sidebar .sidebar-auth button:hover{text-decoration:none;background:#272b39;color:#fff}.dashboard .sidebar .sidebar-nav{list-style:none}.dashboard .sidebar .sidebar-nav li{display:block}.dashboard .sidebar .sidebar-nav li a,.dashboard .sidebar .sidebar-nav li button{display:block;color:rgba(255,255,255,.5);padding:15px}.dashboard .sidebar .sidebar-nav li a svg,.dashboard .sidebar .sidebar-nav li button svg{margin-right:3px;display:inline-block;vertical-align:middle;fill:rgba(255,255,255,.5);width:15px}.dashboard .sidebar .sidebar-nav li a span,.dashboard .sidebar .sidebar-nav li button span{margin-top:2px;display:inline-block;vertical-align:middle}.dashboard .sidebar .sidebar-nav li button{width:100%;border:none;text-align:left;background:rgba(0,0,0,0)}.dashboard .sidebar .sidebar-nav li a.active,.dashboard .sidebar .sidebar-nav li a:hover,.dashboard .sidebar .sidebar-nav li button:hover{text-decoration:none;background:#272b39;color:#fff}.dashboard .sidebar .sidebar-nav li a.active svg,.dashboard .sidebar .sidebar-nav li a:hover svg,.dashboard .sidebar .sidebar-nav li button:hover svg{fill:#fff}.dashboard .sidebar .sidebar-nav li.sidebar-nav-header{padding:15px;font-weight:700;color:#fff}.dashboard-header{margin-bottom:10px}.dashboard-header h1{float:left}.dashboard-header h1 .back{margin-top:2px;display:inline-block;vertical-align:middle}.dashboard-header h1 .back svg{fill:#7f7f7f}.dashboard-header h1 .back:hover{text-decoration:none}.dashboard-header h1 .back:hover svg{fill:#444}.dashboard-header h1 small{margin-top:10px;display:block;font-size:16px;color:rgba(0,0,0,.5)}.dashboard-header .pill{margin-top:-5px;margin-left:10px}.dashboard-header .dashboard-actions{float:right;list-style:none}.dashboard-header .dashboard-actions li{display:inline}.dashboard-header .dashboard-actions li form{display:inline-block}.dashboard-header .dashboard-actions li a{cursor:pointer;display:inline-block}.dashboard-nav{list-style:none}.dashboard-nav li{display:inline}.dashboard-nav li a{display:inline-block;padding:15px;color:#9f9f9f}.dashboard-nav li a svg{margin-right:3px;width:20px;vertical-align:middle;display:inline-block;fill:#9f9f9f}.dashboard-nav li a span{margin-top:2px;display:inline-block;vertical-align:middle}.dashboard-nav li a.active,.dashboard-nav li a:hover{text-decoration:none;color:#272b39}.dashboard-nav li a.active svg,.dashboard-nav li a:hover svg{fill:#272b39}.dashboard-content{margin-left:225px}.dashboard-content .alert{overflow:auto;padding:15px}.dashboard-content .alert .alert-message{float:left;color:rgba(0,0,0,.6)}.dashboard-content .alert a.alert-close{float:right;display:inline-block}.dashboard-content .alert a.alert-close svg{width:15px;height:15px;fill:rgba(0,0,0,.4)}.dashboard-content .alert a.alert-close:hover svg{fill:rgba(0,0,0,.5)}.dashboard-content .alert-success{background:#caf5ca;border:solid 1px #a0dfa0}.dashboard-content .alert-warn{background:#fff3cd;border:solid 1px #d9c995}.dashboard-content .alert-danger{background:#ffd4d4;border:solid 1px #e19e9e}.dashboard-content .dashboard-wrap{margin:0 auto;max-width:1300px;padding:20px}@media (max-width:1500px){.dashboard .sidebar{width:70px}.dashboard .sidebar .sidebar-header{padding:15px}.dashboard .sidebar .sidebar-header .logo{margin-top:0;margin-right:0}.dashboard .sidebar .sidebar-header h2{display:none}.dashboard .sidebar .sidebar-nav .sidebar-nav-header{display:none}.dashboard .sidebar .sidebar-nav li a,.dashboard .sidebar .sidebar-nav li button{text-align:center}.dashboard .sidebar .sidebar-nav li a span,.dashboard .sidebar .sidebar-nav li button span{display:none}.dashboard .dashboard-content{margin-left:70px}}@media (max-width:1000px){.dashboard .dashboard-content .dashboard-header .dashboard-nav li a span{display:none}}.form-field+.form-field{margin-top:15px}.form-field{overflow:auto}.form-field .label{margin-bottom:5px;display:block;font-weight:700}.form-field .label small{color:rgba(0,0,0,.5)}.form-field .form-error{margin-top:5px;color:#ff4343;min-height:20px}.form-field .form-text{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji","Sego UI Symbol";font-size:14px;padding:10px;outline:0;border-radius:3px;box-sizing:border-box;width:100%;border:solid 1px #e4e4e4}.form-field .form-text:focus{border:solid 1px #c2c2c2}.form-field .form-code{min-height:250px;font-family:monospace}.form-field textarea.form-text{min-width:100%;max-width:100%}.form-field .form-option+.form-option{margin-top:10px}.form-field .form-option{display:block;cursor:pointer;overflow:auto}.form-field .form-option .form-selector{margin-right:5px;display:inline-block;float:left;outline:0}.form-field .form-option svg{margin-right:5px;float:left;fill:rgba(0,0,0,.4)}.form-field .disabled{color:#aaa;cursor:not-allowed}.form-field .disabled svg{fill:#aaa}.form-search{float:right;padding:7px}.form-search .form-text{width:auto}.form-search a svg{margin-top:-3px;fill:#e4e4e4;width:20px;vertical-align:middle;display:inline-block}.form-search a:hover svg{fill:#c2c2c2}.form-field-inline .form-text{display:inline-block;width:auto}.form-field-inline .form-error{display:inline-block}form h2{margin-bottom:15px}.panel+.panel{margin-top:15px}.panel{background:#fff;border-radius:3px;box-shadow:0 2px 4px 0 rgba(0,0,0,.1)}.panel .panel-body{padding:15px}.panel .panel-message{font-size:20px;padding:150px;text-align:center}.panel .panel-footer{border-top:solid 1px #e4e4e4;padding:15px}.panel table.code{border-radius:0 0 3px 3px}.panel-header{border-bottom:solid 1px #e4e4e4;overflow:auto}.panel-header h3{float:left;padding:15px;font-weight:700}.panel-header .panel-nav{list-style:none;float:left}.panel-header .panel-nav li{display:inline}.panel-header .panel-nav li a{display:inline-block;padding:15px;padding-left:17px;padding-right:17px;color:rgba(0,0,0,.4)}.panel-header .panel-nav li a svg{margin-right:3px;width:15px;vertical-align:middle;display:inline-block;fill:rgba(0,0,0,.4)}.panel-header .panel-nav li a span{margin-top:2px;vertical-align:middle;display:inline-block}.panel-header .panel-nav li a.active,.panel-header .panel-nav li a:hover{text-decoration:none;border-bottom:solid 2px #383e51;color:#383e51}.panel-header .panel-nav li a.active svg,.panel-header .panel-nav li a:hover svg{fill:#383e51}.panel-header .panel-actions{float:right;list-style:none;padding:7px}.panel-header .panel-actions .btn{padding:5px;padding-left:12px;padding-right:12px}.panel-header .panel-actions li{display:inline}.panel-header .panel-actions li a{display:inline-block}.panel-header .panel-actions li a svg{margin-right:3px;width:15px;vertical-align:middle;display:inline-block}.panel-header .panel-actions li a span{margin-top:2px;vertical-align:middle;display:inline-block}@media (max-width:1100px){.panel .panel-header .panel-nav li a span{display:none}}@media (max-width:700px){.panel .panel-header .form-search{display:none}}.pill{display:inline-block;text-align:center;padding:3px;padding-left:10px;padding-right:10px;border-radius:25px;color:#fff;font-size:14px;vertical-align:middle}.pill a{text-decoration:none}.pill svg{margin-top:-2px;display:inline-block;vertical-align:middle;width:15px;fill:#fff}.pill-bubble{margin-right:5px;border-radius:100%;width:25px;height:25px;text-align:center;display:inline-block}.pill-bubble svg{width:15px;fill:#fff;vertical-align:middle}a.pill:hover{text-decoration:none}.pill-light{background:#61a0ea}a.pill-light:hover{background:#5090d9}.pill-gray{background:#6a7393}.pill-dark{background:#272b39}.pill-red{background:#c64242}.pill-green{background:#269326}.pill-blue{background:#61a0ea}.pill-orange{background:#ff7400}@media (max-width:950px){.pill{width:25px!important}.pill span{display:none}}.providers{margin-top:15px;margin-bottom:15px}.provider-btn{display:inline-block;border-radius:3px;color:#fff;border:none;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,Helvetica,Arial,sans-serif,"Apple Color Emoji","Segoe UI Emoji","Sego UI Symbol";padding:10px;padding-left:15px;padding-right:15px}.provider-btn svg{margin-right:5px;fill:#fff;vertical-align:middle}.provider-btn span{display:inline-block;vertical-align:middle}.provider-btn:hover{text-decoration:none}.provider-github{background:#24292e}.provider-github:hover{background:#353a3f}.provider-gitlab{background:#fa7035}.provider-gitlab:hover{background:#e65328}.table{border-collapse:collapse;width:100%}.table td,.table th{padding:10px}.table td svg,.table th svg{width:15px;display:inline-block;vertical-align:middle}.table td form,.table th form{display:inline-block}.table td.success{color:#269326}.table td.success svg{fill:#269326}.table td.warning{color:#ff7400}.table td.warning svg{fill:#ff7400}.table td.error{color:#c64242}.table td.error svg{fill:#c64242}.table th{background:rgba(0,0,0,.03);border-bottom:solid 1px #e4e4e4;text-align:left;color:rgba(0,0,0,.5);font-weight:400}.table th.align-right{text-align:right}.table tr{border-bottom:solid 1px #e4e4e4}.table tr:last-child{border-bottom:none}.table .cell-pill{width:100px}.table .cell-date{text-align:right!important;width:250px}@media (max-width:900px){th.hide-mobile{display:none}td.hide-mobile{display:none}}.overflow{overflow:auto;padding-bottom:5px}.muted{color:#9f9f9f}.muted svg{fill:#9f9f9f}.align-center{text-align:center}.align-right{text-align:right}.inline-block{display:inline-block}.separator{margin-top:20px;margin-bottom:20px;border-bottom:solid 1px #cfcfcf}.slim{margin:0 auto;max-width:600px}.left{float:left}.right{float:right}.w-90{width:90px}.mt-5{margin-top:5px}.mb-10{margin-bottom:10px}.pr-5{padding-right:5px}.pl-5{padding-left:5px}.progress-wrap .progress-bg{padding:3px;border-radius:3px;width:100%;background:#e4e4e4}.progress-wrap .progress{margin-top:-6px;padding:3px;border-radius:3px;background:#61a0ea}.svg-red svg{fill:#c64242}.svg-green svg{fill:#269326}.paginator{margin:0 auto;list-style:none;max-width:250px}.paginator li{display:inline}.paginator li a{display:inline-block;box-sizing:border-box;text-align:center;padding:10px;width:50%}.paginator li a.disabled{cursor:not-allowed;color:rgba(0,0,0,.5)}.paginator li a:hover{text-decoration:none}.paginator li .prev:hover{border-radius:3px 0 0 3px;background:#61a0ea;color:#fff}.paginator li .next:hover{border-radius:0 3px 3px 0;background:#61a0ea;color:#fff}.scope-list h3{margin-bottom:15px}.scope-list .scope-item{margin-top:15px;overflow:auto;border-top:solid 1px #cfcfcf;padding:15px}.scope-list .scope-item svg{display:inline-block;margin-right:15px;float:left;fill:rgba(0,0,0,.4)}.scope-list .scope-item span{display:inline-block}.scope-list .scope-item span strong{display:block}`)
//line template/template.qtpl:80
	qw422016.N().S(`</style> </head> <body>`)
//line template/template.qtpl:82
	p.StreamBody(qw422016)
//line template/template.qtpl:82
	qw422016.N().S(`</body> <footer>`)
//line template/template.qtpl:83
	p.StreamFooter(qw422016)
//line template/template.qtpl:83
	qw422016.N().S(`</footer> </html> `)
//line template/template.qtpl:85
}

//line template/template.qtpl:85
func WriteRender(qq422016 qtio422016.Writer, p Page) {
//line template/template.qtpl:85
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/template.qtpl:85
	StreamRender(qw422016, p)
//line template/template.qtpl:85
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:85
}

//line template/template.qtpl:85
func Render(p Page) string {
//line template/template.qtpl:85
	qb422016 := qt422016.AcquireByteBuffer()
//line template/template.qtpl:85
	WriteRender(qb422016, p)
//line template/template.qtpl:85
	qs422016 := string(qb422016.B)
//line template/template.qtpl:85
	qt422016.ReleaseByteBuffer(qb422016)
//line template/template.qtpl:85
	return qs422016
//line template/template.qtpl:85
}

//line template/template.qtpl:88
func StreamRenderPaginator(qw422016 *qt422016.Writer, url *url.URL, p database.Paginator) {
//line template/template.qtpl:89
	if len(p.Pages) > 1 {
//line template/template.qtpl:89
		qw422016.N().S(`<ul class="paginator panel">`)
//line template/template.qtpl:91
		if p.Page == p.Prev {
//line template/template.qtpl:91
			qw422016.N().S(`<li><a class="disabled">Previous</a></li>`)
//line template/template.qtpl:93
		} else {
//line template/template.qtpl:93
			qw422016.N().S(`<li><a href="`)
//line template/template.qtpl:94
			qw422016.E().S(pageURL(url, p.Prev))
//line template/template.qtpl:94
			qw422016.N().S(`" class="prev">Previous</a></li>`)
//line template/template.qtpl:95
		}
//line template/template.qtpl:96
		if p.Page == p.Next {
//line template/template.qtpl:96
			qw422016.N().S(`<li><a class="disabled">Next</a></li>`)
//line template/template.qtpl:98
		} else {
//line template/template.qtpl:98
			qw422016.N().S(`<li><a href="`)
//line template/template.qtpl:99
			qw422016.E().S(pageURL(url, p.Next))
//line template/template.qtpl:99
			qw422016.N().S(`" class="next">Next</a></li>`)
//line template/template.qtpl:100
		}
//line template/template.qtpl:100
		qw422016.N().S(`</ul>`)
//line template/template.qtpl:102
	}
//line template/template.qtpl:103
}

//line template/template.qtpl:103
func WriteRenderPaginator(qq422016 qtio422016.Writer, url *url.URL, p database.Paginator) {
//line template/template.qtpl:103
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/template.qtpl:103
	StreamRenderPaginator(qw422016, url, p)
//line template/template.qtpl:103
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:103
}

//line template/template.qtpl:103
func RenderPaginator(url *url.URL, p database.Paginator) string {
//line template/template.qtpl:103
	qb422016 := qt422016.AcquireByteBuffer()
//line template/template.qtpl:103
	WriteRenderPaginator(qb422016, url, p)
//line template/template.qtpl:103
	qs422016 := string(qb422016.B)
//line template/template.qtpl:103
	qt422016.ReleaseByteBuffer(qb422016)
//line template/template.qtpl:103
	return qs422016
//line template/template.qtpl:103
}

//line template/template.qtpl:106
func (f Form) StreamError(qw422016 *qt422016.Writer, field string) {
//line template/template.qtpl:106
	qw422016.N().S(` <div class="form-error">`)
//line template/template.qtpl:106
	qw422016.E().S(f.Errors.First(field))
//line template/template.qtpl:106
	qw422016.N().S(`</div> `)
//line template/template.qtpl:106
}

//line template/template.qtpl:106
func (f Form) WriteError(qq422016 qtio422016.Writer, field string) {
//line template/template.qtpl:106
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/template.qtpl:106
	f.StreamError(qw422016, field)
//line template/template.qtpl:106
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:106
}

//line template/template.qtpl:106
func (f Form) Error(field string) string {
//line template/template.qtpl:106
	qb422016 := qt422016.AcquireByteBuffer()
//line template/template.qtpl:106
	f.WriteError(qb422016, field)
//line template/template.qtpl:106
	qs422016 := string(qb422016.B)
//line template/template.qtpl:106
	qt422016.ReleaseByteBuffer(qb422016)
//line template/template.qtpl:106
	return qs422016
//line template/template.qtpl:106
}

//line template/template.qtpl:108
func (p *BasePage) StreamTitle(qw422016 *qt422016.Writer) {
//line template/template.qtpl:108
	qw422016.N().S(` Djinn CI `)
//line template/template.qtpl:108
}

//line template/template.qtpl:108
func (p *BasePage) WriteTitle(qq422016 qtio422016.Writer) {
//line template/template.qtpl:108
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/template.qtpl:108
	p.StreamTitle(qw422016)
//line template/template.qtpl:108
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:108
}

//line template/template.qtpl:108
func (p *BasePage) Title() string {
//line template/template.qtpl:108
	qb422016 := qt422016.AcquireByteBuffer()
//line template/template.qtpl:108
	p.WriteTitle(qb422016)
//line template/template.qtpl:108
	qs422016 := string(qb422016.B)
//line template/template.qtpl:108
	qt422016.ReleaseByteBuffer(qb422016)
//line template/template.qtpl:108
	return qs422016
//line template/template.qtpl:108
}

//line template/template.qtpl:109
func (p *BasePage) StreamBody(qw422016 *qt422016.Writer) {
//line template/template.qtpl:109
}

//line template/template.qtpl:109
func (p *BasePage) WriteBody(qq422016 qtio422016.Writer) {
//line template/template.qtpl:109
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/template.qtpl:109
	p.StreamBody(qw422016)
//line template/template.qtpl:109
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:109
}

//line template/template.qtpl:109
func (p *BasePage) Body() string {
//line template/template.qtpl:109
	qb422016 := qt422016.AcquireByteBuffer()
//line template/template.qtpl:109
	p.WriteBody(qb422016)
//line template/template.qtpl:109
	qs422016 := string(qb422016.B)
//line template/template.qtpl:109
	qt422016.ReleaseByteBuffer(qb422016)
//line template/template.qtpl:109
	return qs422016
//line template/template.qtpl:109
}

//line template/template.qtpl:110
func (p *BasePage) StreamFooter(qw422016 *qt422016.Writer) {
//line template/template.qtpl:110
	qw422016.N().S(` `)
//line template/template.qtpl:110
}

//line template/template.qtpl:110
func (p *BasePage) WriteFooter(qq422016 qtio422016.Writer) {
//line template/template.qtpl:110
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/template.qtpl:110
	p.StreamFooter(qw422016)
//line template/template.qtpl:110
	qt422016.ReleaseWriter(qw422016)
//line template/template.qtpl:110
}

//line template/template.qtpl:110
func (p *BasePage) Footer() string {
//line template/template.qtpl:110
	qb422016 := qt422016.AcquireByteBuffer()
//line template/template.qtpl:110
	p.WriteFooter(qb422016)
//line template/template.qtpl:110
	qs422016 := string(qb422016.B)
//line template/template.qtpl:110
	qt422016.ReleaseByteBuffer(qb422016)
//line template/template.qtpl:110
	return qs422016
//line template/template.qtpl:110
}
