// Code generated by qtc from "file.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/file.qtpl:3
package template

//line template/file.qtpl:3
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/file.qtpl:3
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/file.qtpl:4
type FileForm struct {
	Form
}

//line template/file.qtpl:10
func (p *FileForm) StreamTitle(qw422016 *qt422016.Writer) {
//line template/file.qtpl:10
}

//line template/file.qtpl:10
func (p *FileForm) WriteTitle(qq422016 qtio422016.Writer) {
//line template/file.qtpl:10
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/file.qtpl:10
	p.StreamTitle(qw422016)
//line template/file.qtpl:10
	qt422016.ReleaseWriter(qw422016)
//line template/file.qtpl:10
}

//line template/file.qtpl:10
func (p *FileForm) Title() string {
//line template/file.qtpl:10
	qb422016 := qt422016.AcquireByteBuffer()
//line template/file.qtpl:10
	p.WriteTitle(qb422016)
//line template/file.qtpl:10
	qs422016 := string(qb422016.B)
//line template/file.qtpl:10
	qt422016.ReleaseByteBuffer(qb422016)
//line template/file.qtpl:10
	return qs422016
//line template/file.qtpl:10
}

//line template/file.qtpl:12
func (p *FileForm) StreamSection(qw422016 *qt422016.Writer) {
//line template/file.qtpl:12
	qw422016.N().S(` <div class="form-field"> <label class="label" for="file">File</label> <input type="file" id="file" name="file"/> `)
//line template/file.qtpl:16
	p.StreamError(qw422016, "file")
//line template/file.qtpl:16
	qw422016.N().S(` </div> `)
//line template/file.qtpl:18
}

//line template/file.qtpl:18
func (p *FileForm) WriteSection(qq422016 qtio422016.Writer) {
//line template/file.qtpl:18
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/file.qtpl:18
	p.StreamSection(qw422016)
//line template/file.qtpl:18
	qt422016.ReleaseWriter(qw422016)
//line template/file.qtpl:18
}

//line template/file.qtpl:18
func (p *FileForm) Section() string {
//line template/file.qtpl:18
	qb422016 := qt422016.AcquireByteBuffer()
//line template/file.qtpl:18
	p.WriteSection(qb422016)
//line template/file.qtpl:18
	qs422016 := string(qb422016.B)
//line template/file.qtpl:18
	qt422016.ReleaseByteBuffer(qb422016)
//line template/file.qtpl:18
	return qs422016
//line template/file.qtpl:18
}
