// Code generated by qtc from "render.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line namespace/template/render.qtpl:1
package template

//line namespace/template/render.qtpl:1
import "strings"

//line namespace/template/render.qtpl:4
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line namespace/template/render.qtpl:4
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line namespace/template/render.qtpl:4
func streamrenderPath(qw422016 *qt422016.Writer, username, fullName string) {
//line namespace/template/render.qtpl:4
	qw422016.N().S(` `)
//line namespace/template/render.qtpl:5
	parts := strings.Split(fullName, "/")

//line namespace/template/render.qtpl:5
	qw422016.N().S(` `)
//line namespace/template/render.qtpl:6
	for i, p := range parts {
//line namespace/template/render.qtpl:6
		qw422016.N().S(` <a href="/n/`)
//line namespace/template/render.qtpl:7
		qw422016.E().S(username)
//line namespace/template/render.qtpl:7
		qw422016.N().S(`/`)
//line namespace/template/render.qtpl:7
		qw422016.E().S(strings.Join(parts[:i+1], "/"))
//line namespace/template/render.qtpl:7
		qw422016.N().S(`">`)
//line namespace/template/render.qtpl:7
		qw422016.E().S(p)
//line namespace/template/render.qtpl:7
		qw422016.N().S(`</a> `)
//line namespace/template/render.qtpl:8
		if i != len(parts)-1 {
//line namespace/template/render.qtpl:8
			qw422016.N().S(`<span> / </span>`)
//line namespace/template/render.qtpl:8
		}
//line namespace/template/render.qtpl:8
		qw422016.N().S(` `)
//line namespace/template/render.qtpl:9
	}
//line namespace/template/render.qtpl:9
	qw422016.N().S(` `)
//line namespace/template/render.qtpl:10
}

//line namespace/template/render.qtpl:10
func writerenderPath(qq422016 qtio422016.Writer, username, fullName string) {
//line namespace/template/render.qtpl:10
	qw422016 := qt422016.AcquireWriter(qq422016)
//line namespace/template/render.qtpl:10
	streamrenderPath(qw422016, username, fullName)
//line namespace/template/render.qtpl:10
	qt422016.ReleaseWriter(qw422016)
//line namespace/template/render.qtpl:10
}

//line namespace/template/render.qtpl:10
func renderPath(username, fullName string) string {
//line namespace/template/render.qtpl:10
	qb422016 := qt422016.AcquireByteBuffer()
//line namespace/template/render.qtpl:10
	writerenderPath(qb422016, username, fullName)
//line namespace/template/render.qtpl:10
	qs422016 := string(qb422016.B)
//line namespace/template/render.qtpl:10
	qt422016.ReleaseByteBuffer(qb422016)
//line namespace/template/render.qtpl:10
	return qs422016
//line namespace/template/render.qtpl:10
}
