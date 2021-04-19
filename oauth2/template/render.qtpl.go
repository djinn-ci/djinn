// Code generated by qtc from "render.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line oauth2/template/render.qtpl:2
package template

//line oauth2/template/render.qtpl:2
import (
	"strings"

	"djinn-ci.com/oauth2"
)

//line oauth2/template/render.qtpl:9
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line oauth2/template/render.qtpl:9
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line oauth2/template/render.qtpl:10
func renderPermissionList(p oauth2.Permission) string {
	var buf strings.Builder

	exp := p.Expand()
	l := len(exp) - 1

	for i, perm := range exp {
		buf.WriteString(strings.Title(perm.String()))

		if i+1 == l {
			buf.WriteString(", and")
		}

		if i != l {
			buf.WriteString(", ")
			continue
		}
	}
	return buf.String()
}

//line oauth2/template/render.qtpl:33
func streamrenderScope(qw422016 *qt422016.Writer, res oauth2.Resource, perm oauth2.Permission) {
//line oauth2/template/render.qtpl:33
	qw422016.N().S(` <div class="scope-item"> `)
//line oauth2/template/render.qtpl:35
	switch res {
//line oauth2/template/render.qtpl:36
	case oauth2.Build:
//line oauth2/template/render.qtpl:36
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:37
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M22.688 18.984c0.422 0.281 0.422 0.984-0.094 1.406l-2.297 2.297c-0.422 0.422-0.984 0.422-1.406 0l-9.094-9.094c-2.297 0.891-4.969 0.422-6.891-1.5-2.016-2.016-2.531-5.016-1.313-7.406l4.406 4.313 3-3-4.313-4.313c2.391-1.078 5.391-0.703 7.406 1.313 1.922 1.922 2.391 4.594 1.5 6.891z"></path>
</svg>
`)
//line oauth2/template/render.qtpl:37
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:38
	case oauth2.Invite:
//line oauth2/template/render.qtpl:38
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:39
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<title>mail</title>
<path d="M12 11.016l8.016-5.016h-16.031zM20.016 18v-9.984l-8.016 4.969-8.016-4.969v9.984h16.031zM20.016 3.984c1.078 0 1.969 0.938 1.969 2.016v12c0 1.078-0.891 2.016-1.969 2.016h-16.031c-1.078 0-1.969-0.938-1.969-2.016v-12c0-1.078 0.891-2.016 1.969-2.016h16.031z"></path>
</svg>
`)
//line oauth2/template/render.qtpl:39
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:40
	case oauth2.Image:
//line oauth2/template/render.qtpl:40
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:41
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M8.484 13.5l-3.469 4.5h13.969l-4.5-6-3.469 4.5zM21 18.984c0 1.078-0.938 2.016-2.016 2.016h-13.969c-1.078 0-2.016-0.938-2.016-2.016v-13.969c0-1.078 0.938-2.016 2.016-2.016h13.969c1.078 0 2.016 0.938 2.016 2.016v13.969z"></path>
</svg>
`)
//line oauth2/template/render.qtpl:41
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:42
	case oauth2.Namespace:
//line oauth2/template/render.qtpl:42
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:43
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M9.984 3.984l2.016 2.016h8.016c1.078 0 1.969 0.938 1.969 2.016v9.984c0 1.078-0.891 2.016-1.969 2.016h-16.031c-1.078 0-1.969-0.938-1.969-2.016v-12c0-1.078 0.891-2.016 1.969-2.016h6z"></path>
</svg>
`)
//line oauth2/template/render.qtpl:43
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:44
	case oauth2.Object:
//line oauth2/template/render.qtpl:44
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:45
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M5.016 18h13.969v2.016h-13.969v-2.016zM9 15.984v-6h-3.984l6.984-6.984 6.984 6.984h-3.984v6h-6z"></path>
</svg>
`)
//line oauth2/template/render.qtpl:45
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:46
	case oauth2.Variable:
//line oauth2/template/render.qtpl:46
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:47
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M14.578 16.594l4.641-4.594-4.641-4.594 1.406-1.406 6 6-6 6zM9.422 16.594l-1.406 1.406-6-6 6-6 1.406 1.406-4.641 4.594z"></path>
</svg>
`)
//line oauth2/template/render.qtpl:47
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:48
	case oauth2.Key:
//line oauth2/template/render.qtpl:48
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:49
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M6.984 14.016c1.078 0 2.016-0.938 2.016-2.016s-0.938-2.016-2.016-2.016-1.969 0.938-1.969 2.016 0.891 2.016 1.969 2.016zM12.656 9.984h10.359v4.031h-2.016v3.984h-3.984v-3.984h-4.359c-0.797 2.344-3.047 3.984-5.672 3.984-3.328 0-6-2.672-6-6s2.672-6 6-6c2.625 0 4.875 1.641 5.672 3.984z"></path>
</svg>
`)
//line oauth2/template/render.qtpl:49
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:50
	case oauth2.Cron:
//line oauth2/template/render.qtpl:50
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:51
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M21.984 12c0 5.156-3.938 9.422-8.953 9.938v-2.016c3.938-0.516 6.984-3.891 6.984-7.922s-3.047-7.406-6.984-7.922v-2.016c5.016 0.516 8.953 4.781 8.953 9.938zM5.672 19.734l1.406-1.406c1.125 0.844 2.484 1.406 3.938 1.594v2.016c-2.016-0.188-3.844-0.984-5.344-2.203zM4.078 12.984c0.188 1.453 0.75 2.813 1.594 3.891l-1.406 1.453c-1.219-1.5-2.016-3.328-2.203-5.344h2.016zM5.672 7.078c-0.844 1.125-1.406 2.484-1.594 3.938h-2.016c0.188-2.016 0.984-3.844 2.203-5.344zM11.016 4.078c-1.453 0.188-2.813 0.75-3.938 1.594l-1.406-1.406c1.5-1.219 3.328-2.016 5.344-2.203v2.016zM13.031 9.797l2.953 2.203c-2.007 1.493-4.007 2.993-6 4.5z"></path>
</svg>
`)
//line oauth2/template/render.qtpl:51
		qw422016.N().S(` `)
//line oauth2/template/render.qtpl:52
	}
//line oauth2/template/render.qtpl:52
	qw422016.N().S(` <span> <strong>`)
//line oauth2/template/render.qtpl:54
	qw422016.E().S(strings.Title(res.String()))
//line oauth2/template/render.qtpl:54
	qw422016.N().S(`</strong> `)
//line oauth2/template/render.qtpl:55
	qw422016.E().S(renderPermissionList(perm))
//line oauth2/template/render.qtpl:55
	qw422016.N().S(` </span> </div> `)
//line oauth2/template/render.qtpl:58
}

//line oauth2/template/render.qtpl:58
func writerenderScope(qq422016 qtio422016.Writer, res oauth2.Resource, perm oauth2.Permission) {
//line oauth2/template/render.qtpl:58
	qw422016 := qt422016.AcquireWriter(qq422016)
//line oauth2/template/render.qtpl:58
	streamrenderScope(qw422016, res, perm)
//line oauth2/template/render.qtpl:58
	qt422016.ReleaseWriter(qw422016)
//line oauth2/template/render.qtpl:58
}

//line oauth2/template/render.qtpl:58
func renderScope(res oauth2.Resource, perm oauth2.Permission) string {
//line oauth2/template/render.qtpl:58
	qb422016 := qt422016.AcquireByteBuffer()
//line oauth2/template/render.qtpl:58
	writerenderScope(qb422016, res, perm)
//line oauth2/template/render.qtpl:58
	qs422016 := string(qb422016.B)
//line oauth2/template/render.qtpl:58
	qt422016.ReleaseByteBuffer(qb422016)
//line oauth2/template/render.qtpl:58
	return qs422016
//line oauth2/template/render.qtpl:58
}
