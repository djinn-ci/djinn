// Code generated by qtc from "paginator.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/paginator.qtpl:2
package template

//line template/paginator.qtpl:2
import (
	"net/url"
	"strconv"
)

//line template/paginator.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/paginator.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/paginator.qtpl:9
type Paginator struct {
	*Page

	Pages []int
	Prev  int
	Curr  int
	Next  int
	Query url.Values
}

func (p Paginator) url(page int) string {
	q := p.URL.Query()
	q.Set("page", strconv.Itoa(page))

	p.URL.RawQuery = q.Encode()
	return p.URL.String()
}

//line template/paginator.qtpl:30
func (p Paginator) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/paginator.qtpl:31
	if len(p.Pages) > 1 {
//line template/paginator.qtpl:31
		qw422016.N().S(`<ul class="paginator panel">`)
//line template/paginator.qtpl:33
		if p.Prev == 0 {
//line template/paginator.qtpl:33
			qw422016.N().S(`<li><a class="disabled">Previous</a></li>`)
//line template/paginator.qtpl:35
		} else {
//line template/paginator.qtpl:35
			qw422016.N().S(`<li><a href="`)
//line template/paginator.qtpl:36
			qw422016.E().S(p.url(p.Prev))
//line template/paginator.qtpl:36
			qw422016.N().S(`" class="prev">Previous</a></li>`)
//line template/paginator.qtpl:37
		}
//line template/paginator.qtpl:38
		if p.Next == 0 {
//line template/paginator.qtpl:38
			qw422016.N().S(`<li><a class="disabled">Next</a></li>`)
//line template/paginator.qtpl:40
		} else {
//line template/paginator.qtpl:40
			qw422016.N().S(`<li><a href="`)
//line template/paginator.qtpl:41
			qw422016.E().S(p.url(p.Next))
//line template/paginator.qtpl:41
			qw422016.N().S(`" class="next">Next</a></li>`)
//line template/paginator.qtpl:42
		}
//line template/paginator.qtpl:42
		qw422016.N().S(`</ul>`)
//line template/paginator.qtpl:44
	}
//line template/paginator.qtpl:45
	qw422016.N().S(` `)
//line template/paginator.qtpl:46
}

//line template/paginator.qtpl:46
func (p Paginator) WriteNavigation(qq422016 qtio422016.Writer) {
//line template/paginator.qtpl:46
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/paginator.qtpl:46
	p.StreamNavigation(qw422016)
//line template/paginator.qtpl:46
	qt422016.ReleaseWriter(qw422016)
//line template/paginator.qtpl:46
}

//line template/paginator.qtpl:46
func (p Paginator) Navigation() string {
//line template/paginator.qtpl:46
	qb422016 := qt422016.AcquireByteBuffer()
//line template/paginator.qtpl:46
	p.WriteNavigation(qb422016)
//line template/paginator.qtpl:46
	qs422016 := string(qb422016.B)
//line template/paginator.qtpl:46
	qt422016.ReleaseByteBuffer(qb422016)
//line template/paginator.qtpl:46
	return qs422016
//line template/paginator.qtpl:46
}

//line template/paginator.qtpl:48
func (p Paginator) StreamSearch(qw422016 *qt422016.Writer, prompt string) {
//line template/paginator.qtpl:48
	qw422016.N().S(` <form class="form-field form-search"> <input type="text" name="search" class="form-text" placeholder="`)
//line template/paginator.qtpl:50
	qw422016.E().S(prompt)
//line template/paginator.qtpl:50
	qw422016.N().S(`" autocomplete="off" value="`)
//line template/paginator.qtpl:50
	qw422016.E().S(p.Query.Get("search"))
//line template/paginator.qtpl:50
	qw422016.N().S(`"/> `)
//line template/paginator.qtpl:51
	if p.Query.Get("search") != "" {
//line template/paginator.qtpl:51
		qw422016.N().S(` <a class="muted" href="`)
//line template/paginator.qtpl:52
		qw422016.E().S(p.URL.Path)
//line template/paginator.qtpl:52
		qw422016.N().S(`">`)
//line template/paginator.qtpl:52
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M18.984 6.422l-5.578 5.578 5.578 5.578-1.406 1.406-5.578-5.578-5.578 5.578-1.406-1.406 5.578-5.578-5.578-5.578 1.406-1.406 5.578 5.578 5.578-5.578z"></path>
</svg>
`)
//line template/paginator.qtpl:52
		qw422016.N().S(`</a> `)
//line template/paginator.qtpl:53
	}
//line template/paginator.qtpl:53
	qw422016.N().S(` </form> `)
//line template/paginator.qtpl:55
}

//line template/paginator.qtpl:55
func (p Paginator) WriteSearch(qq422016 qtio422016.Writer, prompt string) {
//line template/paginator.qtpl:55
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/paginator.qtpl:55
	p.StreamSearch(qw422016, prompt)
//line template/paginator.qtpl:55
	qt422016.ReleaseWriter(qw422016)
//line template/paginator.qtpl:55
}

//line template/paginator.qtpl:55
func (p Paginator) Search(prompt string) string {
//line template/paginator.qtpl:55
	qb422016 := qt422016.AcquireByteBuffer()
//line template/paginator.qtpl:55
	p.WriteSearch(qb422016, prompt)
//line template/paginator.qtpl:55
	qs422016 := string(qb422016.B)
//line template/paginator.qtpl:55
	qt422016.ReleaseByteBuffer(qb422016)
//line template/paginator.qtpl:55
	return qs422016
//line template/paginator.qtpl:55
}
