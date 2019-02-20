// This file is automatically generated by qtc from "page.qtpl".
// See https://github.com/valyala/quicktemplate for details.

//line template/page.qtpl:2
package template

//line template/page.qtpl:2
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/page.qtpl:2
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/page.qtpl:2
type page interface {
	//line template/page.qtpl:2
	Title() string
	//line template/page.qtpl:2
	StreamTitle(qw422016 *qt422016.Writer)
	//line template/page.qtpl:2
	WriteTitle(qq422016 qtio422016.Writer)
	//line template/page.qtpl:2
	Header() string
	//line template/page.qtpl:2
	StreamHeader(qw422016 *qt422016.Writer)
	//line template/page.qtpl:2
	WriteHeader(qq422016 qtio422016.Writer)
	//line template/page.qtpl:2
	Body() string
	//line template/page.qtpl:2
	StreamBody(qw422016 *qt422016.Writer)
	//line template/page.qtpl:2
	WriteBody(qq422016 qtio422016.Writer)
	//line template/page.qtpl:2
	Footer() string
	//line template/page.qtpl:2
	StreamFooter(qw422016 *qt422016.Writer)
	//line template/page.qtpl:2
	WriteFooter(qq422016 qtio422016.Writer)
//line template/page.qtpl:2
}

//line template/page.qtpl:14
type Page struct{}

//line template/page.qtpl:17
func StreamRender(qw422016 *qt422016.Writer, p page) {
	//line template/page.qtpl:17
	qw422016.N().S(`
<!DOCTYPE HTML>
<html lang="en">
	<head>
		<meta charset="utf-8">
		<meta content="width=device-width, initial-scale=1" name="viewport">
		<title>`)
	//line template/page.qtpl:23
	p.StreamTitle(qw422016)
	//line template/page.qtpl:23
	qw422016.N().S(`</title>
		`)
	//line template/page.qtpl:24
	p.StreamHeader(qw422016)
	//line template/page.qtpl:24
	qw422016.N().S(`
	</head>
	<body>`)
	//line template/page.qtpl:26
	p.StreamBody(qw422016)
	//line template/page.qtpl:26
	qw422016.N().S(`</body>
	<footer>`)
	//line template/page.qtpl:27
	p.StreamFooter(qw422016)
	//line template/page.qtpl:27
	qw422016.N().S(`</footer>
</html>
`)
//line template/page.qtpl:29
}

//line template/page.qtpl:29
func WriteRender(qq422016 qtio422016.Writer, p page) {
	//line template/page.qtpl:29
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/page.qtpl:29
	StreamRender(qw422016, p)
	//line template/page.qtpl:29
	qt422016.ReleaseWriter(qw422016)
//line template/page.qtpl:29
}

//line template/page.qtpl:29
func Render(p page) string {
	//line template/page.qtpl:29
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/page.qtpl:29
	WriteRender(qb422016, p)
	//line template/page.qtpl:29
	qs422016 := string(qb422016.B)
	//line template/page.qtpl:29
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/page.qtpl:29
	return qs422016
//line template/page.qtpl:29
}

//line template/page.qtpl:31
func (p *Page) StreamTitle(qw422016 *qt422016.Writer) {
	//line template/page.qtpl:31
	qw422016.N().S(`
Thrall
`)
//line template/page.qtpl:33
}

//line template/page.qtpl:33
func (p *Page) WriteTitle(qq422016 qtio422016.Writer) {
	//line template/page.qtpl:33
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/page.qtpl:33
	p.StreamTitle(qw422016)
	//line template/page.qtpl:33
	qt422016.ReleaseWriter(qw422016)
//line template/page.qtpl:33
}

//line template/page.qtpl:33
func (p *Page) Title() string {
	//line template/page.qtpl:33
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/page.qtpl:33
	p.WriteTitle(qb422016)
	//line template/page.qtpl:33
	qs422016 := string(qb422016.B)
	//line template/page.qtpl:33
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/page.qtpl:33
	return qs422016
//line template/page.qtpl:33
}

//line template/page.qtpl:35
func (p *Page) StreamHeader(qw422016 *qt422016.Writer) {
	//line template/page.qtpl:35
	qw422016.N().S(`
<link rel="stylesheet" type="text/css" href="/assets/css/main.css">
`)
//line template/page.qtpl:37
}

//line template/page.qtpl:37
func (p *Page) WriteHeader(qq422016 qtio422016.Writer) {
	//line template/page.qtpl:37
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/page.qtpl:37
	p.StreamHeader(qw422016)
	//line template/page.qtpl:37
	qt422016.ReleaseWriter(qw422016)
//line template/page.qtpl:37
}

//line template/page.qtpl:37
func (p *Page) Header() string {
	//line template/page.qtpl:37
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/page.qtpl:37
	p.WriteHeader(qb422016)
	//line template/page.qtpl:37
	qs422016 := string(qb422016.B)
	//line template/page.qtpl:37
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/page.qtpl:37
	return qs422016
//line template/page.qtpl:37
}

//line template/page.qtpl:39
func (p *Page) StreamBody(qw422016 *qt422016.Writer) {
	//line template/page.qtpl:39
	qw422016.N().S(`
`)
//line template/page.qtpl:40
}

//line template/page.qtpl:40
func (p *Page) WriteBody(qq422016 qtio422016.Writer) {
	//line template/page.qtpl:40
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/page.qtpl:40
	p.StreamBody(qw422016)
	//line template/page.qtpl:40
	qt422016.ReleaseWriter(qw422016)
//line template/page.qtpl:40
}

//line template/page.qtpl:40
func (p *Page) Body() string {
	//line template/page.qtpl:40
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/page.qtpl:40
	p.WriteBody(qb422016)
	//line template/page.qtpl:40
	qs422016 := string(qb422016.B)
	//line template/page.qtpl:40
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/page.qtpl:40
	return qs422016
//line template/page.qtpl:40
}

//line template/page.qtpl:42
func (p *Page) StreamFooter(qw422016 *qt422016.Writer) {
	//line template/page.qtpl:42
	qw422016.N().S(`
`)
//line template/page.qtpl:43
}

//line template/page.qtpl:43
func (p *Page) WriteFooter(qq422016 qtio422016.Writer) {
	//line template/page.qtpl:43
	qw422016 := qt422016.AcquireWriter(qq422016)
	//line template/page.qtpl:43
	p.StreamFooter(qw422016)
	//line template/page.qtpl:43
	qt422016.ReleaseWriter(qw422016)
//line template/page.qtpl:43
}

//line template/page.qtpl:43
func (p *Page) Footer() string {
	//line template/page.qtpl:43
	qb422016 := qt422016.AcquireByteBuffer()
	//line template/page.qtpl:43
	p.WriteFooter(qb422016)
	//line template/page.qtpl:43
	qs422016 := string(qb422016.B)
	//line template/page.qtpl:43
	qt422016.ReleaseByteBuffer(qb422016)
	//line template/page.qtpl:43
	return qs422016
//line template/page.qtpl:43
}