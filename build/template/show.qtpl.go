// Code generated by qtc from "show.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line build/template/show.qtpl:2
package template

//line build/template/show.qtpl:2
import (
	htmltemplate "html/template"
	"time"

	"github.com/andrewpillar/djinn/build"
	"github.com/andrewpillar/djinn/runner"
	"github.com/andrewpillar/djinn/template"

	"github.com/hako/durafmt"
)

//line build/template/show.qtpl:14
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line build/template/show.qtpl:14
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line build/template/show.qtpl:15
type Show struct {
	template.BasePage
	template.Section

	Build *build.Build
	CSRF  htmltemplate.HTML
}

type Job struct {
	template.BasePage

	Job *build.Job
}

var timeFormat = "Jan 02, 2006, at 15:04:05"

//line build/template/show.qtpl:34
func streamrenderTimestamp(qw422016 *qt422016.Writer, t time.Time) {
//line build/template/show.qtpl:34
	qw422016.N().S(` `)
//line build/template/show.qtpl:36
	duration := time.Now().Sub(t)
	formatted := t.Format(timeFormat)

//line build/template/show.qtpl:38
	qw422016.N().S(` `)
//line build/template/show.qtpl:39
	if duration < (24 * time.Hour) {
//line build/template/show.qtpl:39
		qw422016.N().S(` <span title="`)
//line build/template/show.qtpl:40
		qw422016.E().S(formatted)
//line build/template/show.qtpl:40
		qw422016.N().S(`">`)
//line build/template/show.qtpl:40
		qw422016.E().V(durafmt.Parse(duration).LimitFirstN(2))
//line build/template/show.qtpl:40
		qw422016.N().S(`</span> ago `)
//line build/template/show.qtpl:41
	} else {
//line build/template/show.qtpl:41
		qw422016.N().S(` `)
//line build/template/show.qtpl:42
		qw422016.E().S(formatted)
//line build/template/show.qtpl:42
		qw422016.N().S(` `)
//line build/template/show.qtpl:43
	}
//line build/template/show.qtpl:43
	qw422016.N().S(` `)
//line build/template/show.qtpl:44
}

//line build/template/show.qtpl:44
func writerenderTimestamp(qq422016 qtio422016.Writer, t time.Time) {
//line build/template/show.qtpl:44
	qw422016 := qt422016.AcquireWriter(qq422016)
//line build/template/show.qtpl:44
	streamrenderTimestamp(qw422016, t)
//line build/template/show.qtpl:44
	qt422016.ReleaseWriter(qw422016)
//line build/template/show.qtpl:44
}

//line build/template/show.qtpl:44
func renderTimestamp(t time.Time) string {
//line build/template/show.qtpl:44
	qb422016 := qt422016.AcquireByteBuffer()
//line build/template/show.qtpl:44
	writerenderTimestamp(qb422016, t)
//line build/template/show.qtpl:44
	qs422016 := string(qb422016.B)
//line build/template/show.qtpl:44
	qt422016.ReleaseByteBuffer(qb422016)
//line build/template/show.qtpl:44
	return qs422016
//line build/template/show.qtpl:44
}

//line build/template/show.qtpl:46
func StreamRenderTrigger(qw422016 *qt422016.Writer, b *build.Build) {
//line build/template/show.qtpl:46
	qw422016.N().S(` `)
//line build/template/show.qtpl:48
	title := b.Trigger.CommentTitle()
	comment := b.Trigger.CommentBody()

//line build/template/show.qtpl:50
	qw422016.N().S(` <div class="panel"> <div class="panel-body"> <div class="comment-header"> <div class="comment-title"> `)
//line build/template/show.qtpl:55
	template.StreamRenderShortStatus(qw422016, b.Status)
//line build/template/show.qtpl:55
	qw422016.N().S(` `)
//line build/template/show.qtpl:56
	if b.Trigger.Comment != "" {
//line build/template/show.qtpl:56
		qw422016.N().S(` <strong>`)
//line build/template/show.qtpl:57
		qw422016.E().S(title)
//line build/template/show.qtpl:57
		qw422016.N().S(`</strong> `)
//line build/template/show.qtpl:58
	} else {
//line build/template/show.qtpl:58
		qw422016.N().S(` <em class="muted">No build comment.</em> `)
//line build/template/show.qtpl:60
	}
//line build/template/show.qtpl:60
	qw422016.N().S(` </div> </div> `)
//line build/template/show.qtpl:63
	if b.Trigger.Comment != "" {
//line build/template/show.qtpl:63
		qw422016.N().S(`<pre>`)
//line build/template/show.qtpl:63
		qw422016.E().S(comment)
//line build/template/show.qtpl:63
		qw422016.N().S(`</pre>`)
//line build/template/show.qtpl:63
	}
//line build/template/show.qtpl:63
	qw422016.N().S(` </div> <div class="panel-footer"> <strong>`)
//line build/template/show.qtpl:66
	qw422016.E().S(b.Trigger.Data["username"])
//line build/template/show.qtpl:66
	qw422016.N().S(`</strong> `)
//line build/template/show.qtpl:67
	switch b.Trigger.Type {
//line build/template/show.qtpl:68
	case build.Manual:
//line build/template/show.qtpl:68
		qw422016.N().S(` submitted `)
//line build/template/show.qtpl:70
	case build.Push:
//line build/template/show.qtpl:70
		qw422016.N().S(` committed <a target="_blank" href="`)
//line build/template/show.qtpl:71
		qw422016.E().S(b.Trigger.Data["url"])
//line build/template/show.qtpl:71
		qw422016.N().S(`">`)
//line build/template/show.qtpl:71
		qw422016.E().S(b.Trigger.Data["sha"][:7])
//line build/template/show.qtpl:71
		qw422016.N().S(`</a> to <span class="code">`)
//line build/template/show.qtpl:71
		qw422016.E().S(b.Trigger.Data["ref"])
//line build/template/show.qtpl:71
		qw422016.N().S(`</span> `)
//line build/template/show.qtpl:72
	case build.Pull:
//line build/template/show.qtpl:72
		qw422016.N().S(` `)
//line build/template/show.qtpl:73
		qw422016.E().S(b.Trigger.Data["action"])
//line build/template/show.qtpl:73
		qw422016.N().S(` pull request <a target="_blank" href="`)
//line build/template/show.qtpl:73
		qw422016.E().S(b.Trigger.Data["url"])
//line build/template/show.qtpl:73
		qw422016.N().S(`">#`)
//line build/template/show.qtpl:73
		qw422016.E().S(b.Trigger.Data["id"])
//line build/template/show.qtpl:73
		qw422016.N().S(`</a> to <span class="code">`)
//line build/template/show.qtpl:73
		qw422016.E().S(b.Trigger.Data["ref"])
//line build/template/show.qtpl:73
		qw422016.N().S(`</span> with commit <span class="code">`)
//line build/template/show.qtpl:73
		qw422016.E().S(b.Trigger.Data["sha"][:7])
//line build/template/show.qtpl:73
		qw422016.N().S(`</span> `)
//line build/template/show.qtpl:74
	}
//line build/template/show.qtpl:74
	qw422016.N().S(` `)
//line build/template/show.qtpl:75
	streamrenderTimestamp(qw422016, b.CreatedAt)
//line build/template/show.qtpl:75
	qw422016.N().S(` </div> `)
//line build/template/show.qtpl:77
	if len(b.Tags) > 0 {
//line build/template/show.qtpl:77
		qw422016.N().S(` <div class="panel-footer"> `)
//line build/template/show.qtpl:79
		for _, t := range b.Tags {
//line build/template/show.qtpl:79
			qw422016.N().S(` <a href="/builds?tag=`)
//line build/template/show.qtpl:80
			qw422016.E().S(t.Name)
//line build/template/show.qtpl:80
			qw422016.N().S(`" class="pill pill-light">`)
//line build/template/show.qtpl:80
			qw422016.E().S(t.Name)
//line build/template/show.qtpl:80
			qw422016.N().S(`</a> `)
//line build/template/show.qtpl:81
		}
//line build/template/show.qtpl:81
		qw422016.N().S(` </div> `)
//line build/template/show.qtpl:83
	}
//line build/template/show.qtpl:83
	qw422016.N().S(` </div> `)
//line build/template/show.qtpl:85
}

//line build/template/show.qtpl:85
func WriteRenderTrigger(qq422016 qtio422016.Writer, b *build.Build) {
//line build/template/show.qtpl:85
	qw422016 := qt422016.AcquireWriter(qq422016)
//line build/template/show.qtpl:85
	StreamRenderTrigger(qw422016, b)
//line build/template/show.qtpl:85
	qt422016.ReleaseWriter(qw422016)
//line build/template/show.qtpl:85
}

//line build/template/show.qtpl:85
func RenderTrigger(b *build.Build) string {
//line build/template/show.qtpl:85
	qb422016 := qt422016.AcquireByteBuffer()
//line build/template/show.qtpl:85
	WriteRenderTrigger(qb422016, b)
//line build/template/show.qtpl:85
	qs422016 := string(qb422016.B)
//line build/template/show.qtpl:85
	qt422016.ReleaseByteBuffer(qb422016)
//line build/template/show.qtpl:85
	return qs422016
//line build/template/show.qtpl:85
}

//line build/template/show.qtpl:87
func (p *Show) StreamTitle(qw422016 *qt422016.Writer) {
//line build/template/show.qtpl:87
	qw422016.N().S(` `)
//line build/template/show.qtpl:88
	if p.Section != nil {
//line build/template/show.qtpl:88
		qw422016.N().S(` `)
//line build/template/show.qtpl:89
		p.Section.StreamTitle(qw422016)
//line build/template/show.qtpl:89
		qw422016.N().S(` - Djinn CI `)
//line build/template/show.qtpl:90
	} else {
//line build/template/show.qtpl:90
		qw422016.N().S(` Build #`)
//line build/template/show.qtpl:91
		qw422016.E().V(p.Build.Number)
//line build/template/show.qtpl:91
		qw422016.N().S(` `)
//line build/template/show.qtpl:91
		qw422016.E().S(p.Build.Trigger.CommentTitle())
//line build/template/show.qtpl:91
		qw422016.N().S(` - Djinn CI `)
//line build/template/show.qtpl:92
	}
//line build/template/show.qtpl:92
	qw422016.N().S(` `)
//line build/template/show.qtpl:93
}

//line build/template/show.qtpl:93
func (p *Show) WriteTitle(qq422016 qtio422016.Writer) {
//line build/template/show.qtpl:93
	qw422016 := qt422016.AcquireWriter(qq422016)
//line build/template/show.qtpl:93
	p.StreamTitle(qw422016)
//line build/template/show.qtpl:93
	qt422016.ReleaseWriter(qw422016)
//line build/template/show.qtpl:93
}

//line build/template/show.qtpl:93
func (p *Show) Title() string {
//line build/template/show.qtpl:93
	qb422016 := qt422016.AcquireByteBuffer()
//line build/template/show.qtpl:93
	p.WriteTitle(qb422016)
//line build/template/show.qtpl:93
	qs422016 := string(qb422016.B)
//line build/template/show.qtpl:93
	qt422016.ReleaseByteBuffer(qb422016)
//line build/template/show.qtpl:93
	return qs422016
//line build/template/show.qtpl:93
}

//line build/template/show.qtpl:95
func (p *Show) StreamBody(qw422016 *qt422016.Writer) {
//line build/template/show.qtpl:95
	qw422016.N().S(` <div class="overflow"> <div class="col-25 col-left"> <div class="panel"> <table class="table"> <tr> <td>Started at:</td> <td class="align-right"> `)
//line build/template/show.qtpl:103
	if p.Build.StartedAt.Valid {
//line build/template/show.qtpl:103
		qw422016.N().S(` `)
//line build/template/show.qtpl:104
		qw422016.E().S(p.Build.StartedAt.Time.Format(timeFormat))
//line build/template/show.qtpl:104
		qw422016.N().S(` `)
//line build/template/show.qtpl:105
	} else {
//line build/template/show.qtpl:105
		qw422016.N().S(` <span class="muted">--</span> `)
//line build/template/show.qtpl:107
	}
//line build/template/show.qtpl:107
	qw422016.N().S(` </td> </tr> <tr> <td>Finished at:</td> <td class="align-right"> `)
//line build/template/show.qtpl:113
	if p.Build.FinishedAt.Valid {
//line build/template/show.qtpl:113
		qw422016.N().S(` `)
//line build/template/show.qtpl:114
		qw422016.E().S(p.Build.FinishedAt.Time.Format(timeFormat))
//line build/template/show.qtpl:114
		qw422016.N().S(` `)
//line build/template/show.qtpl:115
	} else {
//line build/template/show.qtpl:115
		qw422016.N().S(` <span class="muted">--</span> `)
//line build/template/show.qtpl:117
	}
//line build/template/show.qtpl:117
	qw422016.N().S(` </td> </tr> <tr> <td>Duration:</td> <td class="align-right"> `)
//line build/template/show.qtpl:123
	if !p.Build.FinishedAt.Valid || !p.Build.StartedAt.Valid {
//line build/template/show.qtpl:123
		qw422016.N().S(` <span class="muted">--</span> `)
//line build/template/show.qtpl:125
	} else {
//line build/template/show.qtpl:125
		qw422016.N().S(` `)
//line build/template/show.qtpl:126
		qw422016.E().V(durafmt.Parse(p.Build.FinishedAt.Time.Sub(p.Build.StartedAt.Time)).LimitFirstN(1))
//line build/template/show.qtpl:126
		qw422016.N().S(` `)
//line build/template/show.qtpl:127
	}
//line build/template/show.qtpl:127
	qw422016.N().S(` </td> </tr> </table> </div> `)
//line build/template/show.qtpl:132
	for _, s := range p.Build.Stages {
//line build/template/show.qtpl:132
		qw422016.N().S(` <div class="panel"> <div class="panel-header"><h3>`)
//line build/template/show.qtpl:134
		qw422016.E().S(s.Name)
//line build/template/show.qtpl:134
		qw422016.N().S(`</h3></div> <table class="table"> `)
//line build/template/show.qtpl:136
		for _, j := range s.Jobs {
//line build/template/show.qtpl:136
			qw422016.N().S(` <tr> <td>`)
//line build/template/show.qtpl:138
			template.StreamRenderShortStatus(qw422016, j.Status)
//line build/template/show.qtpl:138
			qw422016.N().S(` <a href="`)
//line build/template/show.qtpl:138
			qw422016.E().S(j.Endpoint())
//line build/template/show.qtpl:138
			qw422016.N().S(`">`)
//line build/template/show.qtpl:138
			qw422016.E().S(j.Name)
//line build/template/show.qtpl:138
			qw422016.N().S(`</a></td> <td class="align-right"> `)
//line build/template/show.qtpl:140
			if !j.StartedAt.Valid || !j.FinishedAt.Valid {
//line build/template/show.qtpl:140
				qw422016.N().S(` <span class="muted">--</span> `)
//line build/template/show.qtpl:142
			} else {
//line build/template/show.qtpl:142
				qw422016.N().S(` `)
//line build/template/show.qtpl:143
				qw422016.E().V(j.FinishedAt.Time.Sub(j.StartedAt.Time))
//line build/template/show.qtpl:143
				qw422016.N().S(` `)
//line build/template/show.qtpl:144
			}
//line build/template/show.qtpl:144
			qw422016.N().S(` </td> </tr> `)
//line build/template/show.qtpl:147
		}
//line build/template/show.qtpl:147
		qw422016.N().S(` </table> </div> `)
//line build/template/show.qtpl:150
	}
//line build/template/show.qtpl:150
	qw422016.N().S(` </div> <div class="col-75 col-right"> `)
//line build/template/show.qtpl:153
	StreamRenderTrigger(qw422016, p.Build)
//line build/template/show.qtpl:153
	qw422016.N().S(` `)
//line build/template/show.qtpl:154
	if p.Section != nil {
//line build/template/show.qtpl:154
		qw422016.N().S(` `)
//line build/template/show.qtpl:155
		p.Section.StreamBody(qw422016)
//line build/template/show.qtpl:155
		qw422016.N().S(` `)
//line build/template/show.qtpl:156
	} else {
//line build/template/show.qtpl:156
		qw422016.N().S(` <div class="panel"> `)
//line build/template/show.qtpl:158
		if p.Build.Output.Valid {
//line build/template/show.qtpl:158
			qw422016.N().S(` <div class="panel-header"> <ul class="panel-actions"> <li> <a class="btn btn-primary" href="`)
//line build/template/show.qtpl:162
			qw422016.E().S(p.Build.Endpoint("output", "raw"))
//line build/template/show.qtpl:162
			qw422016.N().S(`"> `)
//line build/template/show.qtpl:163
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12.984 9h5.531l-5.531-5.484v5.484zM15.984 14.016v-2.016h-7.969v2.016h7.969zM15.984 18v-2.016h-7.969v2.016h7.969zM14.016 2.016l6 6v12c0 1.078-0.938 1.969-2.016 1.969h-12c-1.078 0-2.016-0.891-2.016-1.969l0.047-16.031c0-1.078 0.891-1.969 1.969-1.969h8.016z"></path>
</svg>
`)
//line build/template/show.qtpl:163
			qw422016.N().S(`<span>Raw</span> </a> </li> </ul> </div> `)
//line build/template/show.qtpl:168
			template.StreamRenderCode(qw422016, p.Build.Output.String)
//line build/template/show.qtpl:168
			qw422016.N().S(` `)
//line build/template/show.qtpl:169
		} else {
//line build/template/show.qtpl:169
			qw422016.N().S(` <div class="panel-message muted">No build output has been produced.</div> `)
//line build/template/show.qtpl:171
		}
//line build/template/show.qtpl:171
		qw422016.N().S(` </div> `)
//line build/template/show.qtpl:173
	}
//line build/template/show.qtpl:173
	qw422016.N().S(` </div> </div> `)
//line build/template/show.qtpl:176
}

//line build/template/show.qtpl:176
func (p *Show) WriteBody(qq422016 qtio422016.Writer) {
//line build/template/show.qtpl:176
	qw422016 := qt422016.AcquireWriter(qq422016)
//line build/template/show.qtpl:176
	p.StreamBody(qw422016)
//line build/template/show.qtpl:176
	qt422016.ReleaseWriter(qw422016)
//line build/template/show.qtpl:176
}

//line build/template/show.qtpl:176
func (p *Show) Body() string {
//line build/template/show.qtpl:176
	qb422016 := qt422016.AcquireByteBuffer()
//line build/template/show.qtpl:176
	p.WriteBody(qb422016)
//line build/template/show.qtpl:176
	qs422016 := string(qb422016.B)
//line build/template/show.qtpl:176
	qt422016.ReleaseByteBuffer(qb422016)
//line build/template/show.qtpl:176
	return qs422016
//line build/template/show.qtpl:176
}

//line build/template/show.qtpl:178
func (p *Show) StreamHeader(qw422016 *qt422016.Writer) {
//line build/template/show.qtpl:178
	qw422016.N().S(` <a href="/" class="back">`)
//line build/template/show.qtpl:179
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
//line build/template/show.qtpl:179
	qw422016.N().S(`</a> `)
//line build/template/show.qtpl:180
	if !p.Build.Namespace.IsZero() {
//line build/template/show.qtpl:180
		qw422016.N().S(` <a href="`)
//line build/template/show.qtpl:181
		qw422016.E().S(p.Build.Namespace.Endpoint())
//line build/template/show.qtpl:181
		qw422016.N().S(`">`)
//line build/template/show.qtpl:181
		qw422016.E().S(p.Build.Namespace.Name)
//line build/template/show.qtpl:181
		qw422016.N().S(`</a> / `)
//line build/template/show.qtpl:182
	}
//line build/template/show.qtpl:182
	qw422016.N().S(` Build #`)
//line build/template/show.qtpl:183
	qw422016.E().V(p.Build.Number)
//line build/template/show.qtpl:183
	qw422016.N().S(` `)
//line build/template/show.qtpl:184
}

//line build/template/show.qtpl:184
func (p *Show) WriteHeader(qq422016 qtio422016.Writer) {
//line build/template/show.qtpl:184
	qw422016 := qt422016.AcquireWriter(qq422016)
//line build/template/show.qtpl:184
	p.StreamHeader(qw422016)
//line build/template/show.qtpl:184
	qt422016.ReleaseWriter(qw422016)
//line build/template/show.qtpl:184
}

//line build/template/show.qtpl:184
func (p *Show) Header() string {
//line build/template/show.qtpl:184
	qb422016 := qt422016.AcquireByteBuffer()
//line build/template/show.qtpl:184
	p.WriteHeader(qb422016)
//line build/template/show.qtpl:184
	qs422016 := string(qb422016.B)
//line build/template/show.qtpl:184
	qt422016.ReleaseByteBuffer(qb422016)
//line build/template/show.qtpl:184
	return qs422016
//line build/template/show.qtpl:184
}

//line build/template/show.qtpl:186
func (p *Show) StreamActions(qw422016 *qt422016.Writer) {
//line build/template/show.qtpl:186
	qw422016.N().S(` `)
//line build/template/show.qtpl:187
	if p.User.ID == p.Build.UserID && p.Build.Status == runner.Running {
//line build/template/show.qtpl:187
		qw422016.N().S(` <li> <form method="POST" action="`)
//line build/template/show.qtpl:189
		qw422016.E().S(p.Build.Endpoint())
//line build/template/show.qtpl:189
		qw422016.N().S(`"> `)
//line build/template/show.qtpl:190
		qw422016.N().V(p.CSRF)
//line build/template/show.qtpl:190
		qw422016.N().S(` <input type="hidden" name="_method" value="DELETE"> <button type="submit" class="btn btn-danger">Kill</button> </form> </li> `)
//line build/template/show.qtpl:195
	}
//line build/template/show.qtpl:195
	qw422016.N().S(` `)
//line build/template/show.qtpl:196
}

//line build/template/show.qtpl:196
func (p *Show) WriteActions(qq422016 qtio422016.Writer) {
//line build/template/show.qtpl:196
	qw422016 := qt422016.AcquireWriter(qq422016)
//line build/template/show.qtpl:196
	p.StreamActions(qw422016)
//line build/template/show.qtpl:196
	qt422016.ReleaseWriter(qw422016)
//line build/template/show.qtpl:196
}

//line build/template/show.qtpl:196
func (p *Show) Actions() string {
//line build/template/show.qtpl:196
	qb422016 := qt422016.AcquireByteBuffer()
//line build/template/show.qtpl:196
	p.WriteActions(qb422016)
//line build/template/show.qtpl:196
	qs422016 := string(qb422016.B)
//line build/template/show.qtpl:196
	qt422016.ReleaseByteBuffer(qb422016)
//line build/template/show.qtpl:196
	return qs422016
//line build/template/show.qtpl:196
}

//line build/template/show.qtpl:199
func (p *Show) StreamNavigation(qw422016 *qt422016.Writer) {
//line build/template/show.qtpl:199
	qw422016.N().S(`<li><a href="`)
//line build/template/show.qtpl:201
	qw422016.E().S(p.Build.Endpoint())
//line build/template/show.qtpl:201
	qw422016.N().S(`" class="`)
//line build/template/show.qtpl:201
	qw422016.E().S(template.Active(p.Build.Endpoint() == p.URL.Path))
//line build/template/show.qtpl:201
	qw422016.N().S(`">`)
//line build/template/show.qtpl:202
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12 9c1.641 0 3 1.359 3 3s-1.359 3-3 3-3-1.359-3-3 1.359-3 3-3zM12 17.016c2.766 0 5.016-2.25 5.016-5.016s-2.25-5.016-5.016-5.016-5.016 2.25-5.016 5.016 2.25 5.016 5.016 5.016zM12 4.5c5.016 0 9.281 3.094 11.016 7.5-1.734 4.406-6 7.5-11.016 7.5s-9.281-3.094-11.016-7.5c1.734-4.406 6-7.5 11.016-7.5z"></path>
</svg>
`)
//line build/template/show.qtpl:202
	qw422016.N().S(`<span>Overview</span></a></li><li><a href="`)
//line build/template/show.qtpl:206
	qw422016.E().S(p.Build.Endpoint("manifest"))
//line build/template/show.qtpl:206
	qw422016.N().S(`" class="`)
//line build/template/show.qtpl:206
	qw422016.E().S(template.Active(p.Build.Endpoint("manifest") == p.URL.Path))
//line build/template/show.qtpl:206
	qw422016.N().S(`">`)
//line build/template/show.qtpl:207
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M6.984 12.984v-1.969h14.016v1.969h-14.016zM6.984 18.984v-1.969h14.016v1.969h-14.016zM6.984 5.016h14.016v1.969h-14.016v-1.969zM2.016 11.016v-1.031h3v0.938l-1.828 2.063h1.828v1.031h-3v-0.938l1.781-2.063h-1.781zM3 8.016v-3h-0.984v-1.031h1.969v4.031h-0.984zM2.016 17.016v-1.031h3v4.031h-3v-1.031h1.969v-0.469h-0.984v-1.031h0.984v-0.469h-1.969z"></path>
</svg>
`)
//line build/template/show.qtpl:207
	qw422016.N().S(`<span>Manifest</span></a></li><li><a href="`)
//line build/template/show.qtpl:211
	qw422016.E().S(p.Build.Endpoint("objects"))
//line build/template/show.qtpl:211
	qw422016.N().S(`" class="`)
//line build/template/show.qtpl:211
	qw422016.E().S(template.Active(p.Build.Endpoint("objects") == p.URL.Path))
//line build/template/show.qtpl:211
	qw422016.N().S(`">`)
//line build/template/show.qtpl:212
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M5.016 18h13.969v2.016h-13.969v-2.016zM9 15.984v-6h-3.984l6.984-6.984 6.984 6.984h-3.984v6h-6z"></path>
</svg>
`)
//line build/template/show.qtpl:212
	qw422016.N().S(`<span>Objects</span></a></li><li><a href="`)
//line build/template/show.qtpl:216
	qw422016.E().S(p.Build.Endpoint("artifacts"))
//line build/template/show.qtpl:216
	qw422016.N().S(`" class="`)
//line build/template/show.qtpl:216
	qw422016.E().S(template.Active(p.Build.Endpoint("artifacts") == p.URL.Path))
//line build/template/show.qtpl:216
	qw422016.N().S(`">`)
//line build/template/show.qtpl:217
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M5.016 18h13.969v2.016h-13.969v-2.016zM18.984 9l-6.984 6.984-6.984-6.984h3.984v-6h6v6h3.984z"></path>
</svg>
`)
//line build/template/show.qtpl:217
	qw422016.N().S(`<span>Artifacts</span></a></li><li><a href="`)
//line build/template/show.qtpl:221
	qw422016.E().S(p.Build.Endpoint("variables"))
//line build/template/show.qtpl:221
	qw422016.N().S(`" class="`)
//line build/template/show.qtpl:221
	qw422016.E().S(template.Active(p.Build.Endpoint("variables") == p.URL.Path))
//line build/template/show.qtpl:221
	qw422016.N().S(`">`)
//line build/template/show.qtpl:222
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M14.578 16.594l4.641-4.594-4.641-4.594 1.406-1.406 6 6-6 6zM9.422 16.594l-1.406 1.406-6-6 6-6 1.406 1.406-4.641 4.594z"></path>
</svg>
`)
//line build/template/show.qtpl:222
	qw422016.N().S(`<span>Variables</span></a></li><li><a href="`)
//line build/template/show.qtpl:226
	qw422016.E().S(p.Build.Endpoint("keys"))
//line build/template/show.qtpl:226
	qw422016.N().S(`" class="`)
//line build/template/show.qtpl:226
	qw422016.E().S(template.Active(p.Build.Endpoint("keys") == p.URL.Path))
//line build/template/show.qtpl:226
	qw422016.N().S(`">`)
//line build/template/show.qtpl:227
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M6.984 14.016c1.078 0 2.016-0.938 2.016-2.016s-0.938-2.016-2.016-2.016-1.969 0.938-1.969 2.016 0.891 2.016 1.969 2.016zM12.656 9.984h10.359v4.031h-2.016v3.984h-3.984v-3.984h-4.359c-0.797 2.344-3.047 3.984-5.672 3.984-3.328 0-6-2.672-6-6s2.672-6 6-6c2.625 0 4.875 1.641 5.672 3.984z"></path>
</svg>
`)
//line build/template/show.qtpl:227
	qw422016.N().S(`<span>Keys</span></a></li><li><a href="`)
//line build/template/show.qtpl:231
	qw422016.E().S(p.Build.Endpoint("tags"))
//line build/template/show.qtpl:231
	qw422016.N().S(`" class="`)
//line build/template/show.qtpl:231
	qw422016.E().S(template.Active(p.Build.Endpoint("tags") == p.URL.Path))
//line build/template/show.qtpl:231
	qw422016.N().S(`">`)
//line build/template/show.qtpl:232
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M17.625 5.859l4.359 6.141-4.359 6.141c-0.375 0.516-0.984 0.844-1.641 0.844h-10.969c-1.078 0-2.016-0.891-2.016-1.969v-10.031c0-1.078 0.938-1.969 2.016-1.969h10.969c0.656 0 1.266 0.328 1.641 0.844z"></path>
</svg>
`)
//line build/template/show.qtpl:232
	qw422016.N().S(`<span>Tags</span></a></li>`)
//line build/template/show.qtpl:235
}

//line build/template/show.qtpl:235
func (p *Show) WriteNavigation(qq422016 qtio422016.Writer) {
//line build/template/show.qtpl:235
	qw422016 := qt422016.AcquireWriter(qq422016)
//line build/template/show.qtpl:235
	p.StreamNavigation(qw422016)
//line build/template/show.qtpl:235
	qt422016.ReleaseWriter(qw422016)
//line build/template/show.qtpl:235
}

//line build/template/show.qtpl:235
func (p *Show) Navigation() string {
//line build/template/show.qtpl:235
	qb422016 := qt422016.AcquireByteBuffer()
//line build/template/show.qtpl:235
	p.WriteNavigation(qb422016)
//line build/template/show.qtpl:235
	qs422016 := string(qb422016.B)
//line build/template/show.qtpl:235
	qt422016.ReleaseByteBuffer(qb422016)
//line build/template/show.qtpl:235
	return qs422016
//line build/template/show.qtpl:235
}

//line build/template/show.qtpl:238
func (p *Job) StreamTitle(qw422016 *qt422016.Writer) {
//line build/template/show.qtpl:238
	qw422016.N().S(` `)
//line build/template/show.qtpl:239
	qw422016.E().S(p.Job.Name)
//line build/template/show.qtpl:239
	qw422016.N().S(` - Djinn `)
//line build/template/show.qtpl:240
}

//line build/template/show.qtpl:240
func (p *Job) WriteTitle(qq422016 qtio422016.Writer) {
//line build/template/show.qtpl:240
	qw422016 := qt422016.AcquireWriter(qq422016)
//line build/template/show.qtpl:240
	p.StreamTitle(qw422016)
//line build/template/show.qtpl:240
	qt422016.ReleaseWriter(qw422016)
//line build/template/show.qtpl:240
}

//line build/template/show.qtpl:240
func (p *Job) Title() string {
//line build/template/show.qtpl:240
	qb422016 := qt422016.AcquireByteBuffer()
//line build/template/show.qtpl:240
	p.WriteTitle(qb422016)
//line build/template/show.qtpl:240
	qs422016 := string(qb422016.B)
//line build/template/show.qtpl:240
	qt422016.ReleaseByteBuffer(qb422016)
//line build/template/show.qtpl:240
	return qs422016
//line build/template/show.qtpl:240
}

//line build/template/show.qtpl:242
func (p *Job) StreamBody(qw422016 *qt422016.Writer) {
//line build/template/show.qtpl:242
	qw422016.N().S(` <div class="overflow"> <div class="col-25 col-left"> <div class="panel"> <table class="table"> <tr> <td>Status:</td> <td class="align-right">`)
//line build/template/show.qtpl:249
	template.StreamRenderStatus(qw422016, p.Job.Status)
//line build/template/show.qtpl:249
	qw422016.N().S(`</td> </tr> <tr> <td>Started at:</td> <td class="align-right"> `)
//line build/template/show.qtpl:254
	if p.Job.StartedAt.Valid {
//line build/template/show.qtpl:254
		qw422016.N().S(` `)
//line build/template/show.qtpl:255
		qw422016.E().S(p.Job.StartedAt.Time.Format("2006-01-02T15:04:05"))
//line build/template/show.qtpl:255
		qw422016.N().S(` `)
//line build/template/show.qtpl:256
	} else {
//line build/template/show.qtpl:256
		qw422016.N().S(` <span class="muted">--</span> `)
//line build/template/show.qtpl:258
	}
//line build/template/show.qtpl:258
	qw422016.N().S(` </td> </tr> <tr> <td>Finished at:</td> <td class="align-right"> `)
//line build/template/show.qtpl:264
	if p.Job.FinishedAt.Valid {
//line build/template/show.qtpl:264
		qw422016.N().S(` `)
//line build/template/show.qtpl:265
		qw422016.E().S(p.Job.FinishedAt.Time.Format("2006-01-02T15:04:05"))
//line build/template/show.qtpl:265
		qw422016.N().S(` `)
//line build/template/show.qtpl:266
	} else {
//line build/template/show.qtpl:266
		qw422016.N().S(` <span class="muted">--</span> `)
//line build/template/show.qtpl:268
	}
//line build/template/show.qtpl:268
	qw422016.N().S(` </td> </tr> <tr> <td>Duration:</td> <td class="align-right"> `)
//line build/template/show.qtpl:274
	if !p.Job.FinishedAt.Valid || !p.Job.StartedAt.Valid {
//line build/template/show.qtpl:274
		qw422016.N().S(` <span class="muted">--</span> `)
//line build/template/show.qtpl:276
	} else {
//line build/template/show.qtpl:276
		qw422016.N().S(` `)
//line build/template/show.qtpl:277
		qw422016.E().V(p.Job.FinishedAt.Time.Sub(p.Job.StartedAt.Time))
//line build/template/show.qtpl:277
		qw422016.N().S(` `)
//line build/template/show.qtpl:278
	}
//line build/template/show.qtpl:278
	qw422016.N().S(` </td> </tr> </table> </div> </div> <div class="col-75 col-right"> `)
//line build/template/show.qtpl:285
	StreamRenderTrigger(qw422016, p.Job.Build)
//line build/template/show.qtpl:285
	qw422016.N().S(` <div class="panel"> `)
//line build/template/show.qtpl:287
	if p.Job.Output.Valid {
//line build/template/show.qtpl:287
		qw422016.N().S(` <div class="panel-header"> <h3>Output</h3> <ul class="panel-actions"> <li><a class="btn btn-primary" href="`)
//line build/template/show.qtpl:291
		qw422016.E().S(p.Job.Endpoint("output", "raw"))
//line build/template/show.qtpl:291
		qw422016.N().S(`">`)
//line build/template/show.qtpl:291
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12.984 9h5.531l-5.531-5.484v5.484zM15.984 14.016v-2.016h-7.969v2.016h7.969zM15.984 18v-2.016h-7.969v2.016h7.969zM14.016 2.016l6 6v12c0 1.078-0.938 1.969-2.016 1.969h-12c-1.078 0-2.016-0.891-2.016-1.969l0.047-16.031c0-1.078 0.891-1.969 1.969-1.969h8.016z"></path>
</svg>
`)
//line build/template/show.qtpl:291
		qw422016.N().S(`<span>Raw</span></a></li> </ul> </div> `)
//line build/template/show.qtpl:294
		template.StreamRenderCode(qw422016, p.Job.Output.String)
//line build/template/show.qtpl:294
		qw422016.N().S(` `)
//line build/template/show.qtpl:295
	} else {
//line build/template/show.qtpl:295
		qw422016.N().S(` <div class="panel-message muted">No job output has been produced.</div> `)
//line build/template/show.qtpl:297
	}
//line build/template/show.qtpl:297
	qw422016.N().S(` </div> <div class="panel"> <div class="panel-header"><h3>Artifacts</h3></div> `)
//line build/template/show.qtpl:301
	if len(p.Job.Artifacts) > 0 {
//line build/template/show.qtpl:301
		qw422016.N().S(` `)
//line build/template/show.qtpl:302
		StreamRenderArtifactTable(qw422016, p.Job.Artifacts, p.URL.Path, "", false)
//line build/template/show.qtpl:302
		qw422016.N().S(` `)
//line build/template/show.qtpl:303
	} else {
//line build/template/show.qtpl:303
		qw422016.N().S(` <div class="panel-message muted">No artifacts have been collected from this job.</div> `)
//line build/template/show.qtpl:305
	}
//line build/template/show.qtpl:305
	qw422016.N().S(` </div> </div> </div> `)
//line build/template/show.qtpl:309
}

//line build/template/show.qtpl:309
func (p *Job) WriteBody(qq422016 qtio422016.Writer) {
//line build/template/show.qtpl:309
	qw422016 := qt422016.AcquireWriter(qq422016)
//line build/template/show.qtpl:309
	p.StreamBody(qw422016)
//line build/template/show.qtpl:309
	qt422016.ReleaseWriter(qw422016)
//line build/template/show.qtpl:309
}

//line build/template/show.qtpl:309
func (p *Job) Body() string {
//line build/template/show.qtpl:309
	qb422016 := qt422016.AcquireByteBuffer()
//line build/template/show.qtpl:309
	p.WriteBody(qb422016)
//line build/template/show.qtpl:309
	qs422016 := string(qb422016.B)
//line build/template/show.qtpl:309
	qt422016.ReleaseByteBuffer(qb422016)
//line build/template/show.qtpl:309
	return qs422016
//line build/template/show.qtpl:309
}

//line build/template/show.qtpl:311
func (p *Job) StreamHeader(qw422016 *qt422016.Writer) {
//line build/template/show.qtpl:311
	qw422016.N().S(` <a href="`)
//line build/template/show.qtpl:312
	qw422016.E().S(p.Job.Build.Endpoint())
//line build/template/show.qtpl:312
	qw422016.N().S(`" class="back">`)
//line build/template/show.qtpl:312
	qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M20.016 11.016v1.969h-12.188l5.578 5.625-1.406 1.406-8.016-8.016 8.016-8.016 1.406 1.406-5.578 5.625h12.188z"></path>
</svg>
`)
//line build/template/show.qtpl:312
	qw422016.N().S(`</a> `)
//line build/template/show.qtpl:313
	if !p.Job.Build.Namespace.IsZero() {
//line build/template/show.qtpl:313
		qw422016.N().S(` <a href="`)
//line build/template/show.qtpl:314
		qw422016.E().S(p.Job.Build.Namespace.Endpoint())
//line build/template/show.qtpl:314
		qw422016.N().S(`">`)
//line build/template/show.qtpl:314
		qw422016.E().V(p.Job.Build.Namespace.Values()["name"])
//line build/template/show.qtpl:314
		qw422016.N().S(`</a> / `)
//line build/template/show.qtpl:315
	}
//line build/template/show.qtpl:315
	qw422016.N().S(` Build #`)
//line build/template/show.qtpl:316
	qw422016.E().V(p.Job.BuildID)
//line build/template/show.qtpl:316
	qw422016.N().S(` / `)
//line build/template/show.qtpl:316
	qw422016.E().S(p.Job.Stage.Name)
//line build/template/show.qtpl:316
	qw422016.N().S(` - `)
//line build/template/show.qtpl:316
	qw422016.E().S(p.Job.Name)
//line build/template/show.qtpl:316
	qw422016.N().S(` `)
//line build/template/show.qtpl:317
}

//line build/template/show.qtpl:317
func (p *Job) WriteHeader(qq422016 qtio422016.Writer) {
//line build/template/show.qtpl:317
	qw422016 := qt422016.AcquireWriter(qq422016)
//line build/template/show.qtpl:317
	p.StreamHeader(qw422016)
//line build/template/show.qtpl:317
	qt422016.ReleaseWriter(qw422016)
//line build/template/show.qtpl:317
}

//line build/template/show.qtpl:317
func (p *Job) Header() string {
//line build/template/show.qtpl:317
	qb422016 := qt422016.AcquireByteBuffer()
//line build/template/show.qtpl:317
	p.WriteHeader(qb422016)
//line build/template/show.qtpl:317
	qs422016 := string(qb422016.B)
//line build/template/show.qtpl:317
	qt422016.ReleaseByteBuffer(qb422016)
//line build/template/show.qtpl:317
	return qs422016
//line build/template/show.qtpl:317
}

//line build/template/show.qtpl:319
func (p *Job) StreamActions(qw422016 *qt422016.Writer) {
//line build/template/show.qtpl:319
}

//line build/template/show.qtpl:319
func (p *Job) WriteActions(qq422016 qtio422016.Writer) {
//line build/template/show.qtpl:319
	qw422016 := qt422016.AcquireWriter(qq422016)
//line build/template/show.qtpl:319
	p.StreamActions(qw422016)
//line build/template/show.qtpl:319
	qt422016.ReleaseWriter(qw422016)
//line build/template/show.qtpl:319
}

//line build/template/show.qtpl:319
func (p *Job) Actions() string {
//line build/template/show.qtpl:319
	qb422016 := qt422016.AcquireByteBuffer()
//line build/template/show.qtpl:319
	p.WriteActions(qb422016)
//line build/template/show.qtpl:319
	qs422016 := string(qb422016.B)
//line build/template/show.qtpl:319
	qt422016.ReleaseByteBuffer(qb422016)
//line build/template/show.qtpl:319
	return qs422016
//line build/template/show.qtpl:319
}

//line build/template/show.qtpl:320
func (p *Job) StreamNavigation(qw422016 *qt422016.Writer) {
//line build/template/show.qtpl:320
}

//line build/template/show.qtpl:320
func (p *Job) WriteNavigation(qq422016 qtio422016.Writer) {
//line build/template/show.qtpl:320
	qw422016 := qt422016.AcquireWriter(qq422016)
//line build/template/show.qtpl:320
	p.StreamNavigation(qw422016)
//line build/template/show.qtpl:320
	qt422016.ReleaseWriter(qw422016)
//line build/template/show.qtpl:320
}

//line build/template/show.qtpl:320
func (p *Job) Navigation() string {
//line build/template/show.qtpl:320
	qb422016 := qt422016.AcquireByteBuffer()
//line build/template/show.qtpl:320
	p.WriteNavigation(qb422016)
//line build/template/show.qtpl:320
	qs422016 := string(qb422016.B)
//line build/template/show.qtpl:320
	qt422016.ReleaseByteBuffer(qb422016)
//line build/template/show.qtpl:320
	return qs422016
//line build/template/show.qtpl:320
}
