// This file is automatically generated by qtc from "render.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/namespace/render.qtpl:2
package namespace

//line template/namespace/render.qtpl:2
import "strings"

//line template/namespace/render.qtpl:6
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/namespace/render.qtpl:6
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/namespace/render.qtpl:6
func streamrenderPath(qw422016 *qt422016.Writer, username, fullName string) {
	//line template/namespace/render.qtpl:6
	qw422016.N().S(` `)
	//line template/namespace/render.qtpl:8
	parts := strings.Split(fullName, "/")

	//line template/namespace/render.qtpl:9
	qw422016.N().S(` `)
	//line template/namespace/render.qtpl:10
	for i, p := range parts {
		//line template/namespace/render.qtpl:10
		qw422016.N().S(` <a href="/n/`)
		//line template/namespace/render.qtpl:11
		qw422016.E().S(username)
		//line template/namespace/render.qtpl:11
		qw422016.N().S(`/`)
		//line template/namespace/render.qtpl:11
		qw422016.E().S(strings.Join(parts[:i+1], "/"))
		//line template/namespace/render.qtpl:11
		qw422016.N().S(`">`)
		//line template/namespace/render.qtpl:11
		qw422016.E().S(p)
		//line template/namespace/render.qtpl:11
		qw422016.N().S(`</a> `)
		//line template/namespace/render.qtpl:12
		if i != len(parts)-1 {
			//line template/namespace/render.qtpl:12
			qw422016.N().S(`<span> / </span>`)
			//line template/namespace/render.qtpl:12
		}
		//line template/namespace/render.qtpl:12
		qw422016.N().S(` `)
		//line template/namespace/render.qtpl:13
	}
	//line template/namespace/render.qtpl:13
	qw422016.N().S(` `)
//line template/namespace/render.qtpl:14
}

//line template/namespace/render.qtpl:14
func writerenderPath(qq422016 qtio422016.Writer, username, fullName string) {
	//line template/namespace/render.qtpl:14
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/namespace/render.qtpl:14
	streamrenderPath(qw422016, username, fullName)
	//line template/namespace/render.qtpl:14
	qt422016.ReleaseWriter(qw422016)
//line template/namespace/render.qtpl:14
}

//line template/namespace/render.qtpl:14
func renderPath(username, fullName string) string {
	//line template/namespace/render.qtpl:14
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/namespace/render.qtpl:14
	writerenderPath(qb422016, username, fullName)
	//line template/namespace/render.qtpl:14
	qs422016 := string(qb422016.B)
	//line template/namespace/render.qtpl:14
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/namespace/render.qtpl:14
	return qs422016
//line template/namespace/render.qtpl:14
}