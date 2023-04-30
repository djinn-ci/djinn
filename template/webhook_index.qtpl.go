// Code generated by qtc from "webhook_index.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/webhook_index.qtpl:2
package template

//line template/webhook_index.qtpl:2
import (
	"djinn-ci.com/namespace"
	"djinn-ci.com/template/form"
)

//line template/webhook_index.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/webhook_index.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/webhook_index.qtpl:9
type WebhookIndex struct {
	*Page

	Namespace *namespace.Namespace
	Webhooks  []*namespace.Webhook
}

//line template/webhook_index.qtpl:18
func (p *WebhookIndex) StreamTitle(qw422016 *qt422016.Writer) {
//line template/webhook_index.qtpl:18
	qw422016.N().S(`Webhooks`)
//line template/webhook_index.qtpl:18
}

//line template/webhook_index.qtpl:18
func (p *WebhookIndex) WriteTitle(qq422016 qtio422016.Writer) {
//line template/webhook_index.qtpl:18
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/webhook_index.qtpl:18
	p.StreamTitle(qw422016)
//line template/webhook_index.qtpl:18
	qt422016.ReleaseWriter(qw422016)
//line template/webhook_index.qtpl:18
}

//line template/webhook_index.qtpl:18
func (p *WebhookIndex) Title() string {
//line template/webhook_index.qtpl:18
	qb422016 := qt422016.AcquireByteBuffer()
//line template/webhook_index.qtpl:18
	p.WriteTitle(qb422016)
//line template/webhook_index.qtpl:18
	qs422016 := string(qb422016.B)
//line template/webhook_index.qtpl:18
	qt422016.ReleaseByteBuffer(qb422016)
//line template/webhook_index.qtpl:18
	return qs422016
//line template/webhook_index.qtpl:18
}

//line template/webhook_index.qtpl:20
func (p *WebhookIndex) StreamHeader(qw422016 *qt422016.Writer) {
//line template/webhook_index.qtpl:20
}

//line template/webhook_index.qtpl:20
func (p *WebhookIndex) WriteHeader(qq422016 qtio422016.Writer) {
//line template/webhook_index.qtpl:20
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/webhook_index.qtpl:20
	p.StreamHeader(qw422016)
//line template/webhook_index.qtpl:20
	qt422016.ReleaseWriter(qw422016)
//line template/webhook_index.qtpl:20
}

//line template/webhook_index.qtpl:20
func (p *WebhookIndex) Header() string {
//line template/webhook_index.qtpl:20
	qb422016 := qt422016.AcquireByteBuffer()
//line template/webhook_index.qtpl:20
	p.WriteHeader(qb422016)
//line template/webhook_index.qtpl:20
	qs422016 := string(qb422016.B)
//line template/webhook_index.qtpl:20
	qt422016.ReleaseByteBuffer(qb422016)
//line template/webhook_index.qtpl:20
	return qs422016
//line template/webhook_index.qtpl:20
}

//line template/webhook_index.qtpl:21
func (p *WebhookIndex) StreamActions(qw422016 *qt422016.Writer) {
//line template/webhook_index.qtpl:21
}

//line template/webhook_index.qtpl:21
func (p *WebhookIndex) WriteActions(qq422016 qtio422016.Writer) {
//line template/webhook_index.qtpl:21
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/webhook_index.qtpl:21
	p.StreamActions(qw422016)
//line template/webhook_index.qtpl:21
	qt422016.ReleaseWriter(qw422016)
//line template/webhook_index.qtpl:21
}

//line template/webhook_index.qtpl:21
func (p *WebhookIndex) Actions() string {
//line template/webhook_index.qtpl:21
	qb422016 := qt422016.AcquireByteBuffer()
//line template/webhook_index.qtpl:21
	p.WriteActions(qb422016)
//line template/webhook_index.qtpl:21
	qs422016 := string(qb422016.B)
//line template/webhook_index.qtpl:21
	qt422016.ReleaseByteBuffer(qb422016)
//line template/webhook_index.qtpl:21
	return qs422016
//line template/webhook_index.qtpl:21
}

//line template/webhook_index.qtpl:22
func (p *WebhookIndex) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/webhook_index.qtpl:22
}

//line template/webhook_index.qtpl:22
func (p *WebhookIndex) WriteNavigation(qq422016 qtio422016.Writer) {
//line template/webhook_index.qtpl:22
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/webhook_index.qtpl:22
	p.StreamNavigation(qw422016)
//line template/webhook_index.qtpl:22
	qt422016.ReleaseWriter(qw422016)
//line template/webhook_index.qtpl:22
}

//line template/webhook_index.qtpl:22
func (p *WebhookIndex) Navigation() string {
//line template/webhook_index.qtpl:22
	qb422016 := qt422016.AcquireByteBuffer()
//line template/webhook_index.qtpl:22
	p.WriteNavigation(qb422016)
//line template/webhook_index.qtpl:22
	qs422016 := string(qb422016.B)
//line template/webhook_index.qtpl:22
	qt422016.ReleaseByteBuffer(qb422016)
//line template/webhook_index.qtpl:22
	return qs422016
//line template/webhook_index.qtpl:22
}

//line template/webhook_index.qtpl:23
func (p *WebhookIndex) StreamFooter(qw422016 *qt422016.Writer) {
//line template/webhook_index.qtpl:23
}

//line template/webhook_index.qtpl:23
func (p *WebhookIndex) WriteFooter(qq422016 qtio422016.Writer) {
//line template/webhook_index.qtpl:23
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/webhook_index.qtpl:23
	p.StreamFooter(qw422016)
//line template/webhook_index.qtpl:23
	qt422016.ReleaseWriter(qw422016)
//line template/webhook_index.qtpl:23
}

//line template/webhook_index.qtpl:23
func (p *WebhookIndex) Footer() string {
//line template/webhook_index.qtpl:23
	qb422016 := qt422016.AcquireByteBuffer()
//line template/webhook_index.qtpl:23
	p.WriteFooter(qb422016)
//line template/webhook_index.qtpl:23
	qs422016 := string(qb422016.B)
//line template/webhook_index.qtpl:23
	qt422016.ReleaseByteBuffer(qb422016)
//line template/webhook_index.qtpl:23
	return qs422016
//line template/webhook_index.qtpl:23
}

//line template/webhook_index.qtpl:25
func (p *WebhookIndex) streamrenderWebhookItem(qw422016 *qt422016.Writer, w *namespace.Webhook) {
//line template/webhook_index.qtpl:25
	qw422016.N().S(` <tr> `)
//line template/webhook_index.qtpl:27
	if !w.Active {
//line template/webhook_index.qtpl:27
		qw422016.N().S(` <td class="hook-status hook-status-none" title="Disabled">`)
//line template/webhook_index.qtpl:28
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M12 20.016c4.406 0 8.016-3.609 8.016-8.016 0-1.781-0.656-3.516-1.734-4.922l-11.203 11.203c1.406 1.078 3.141 1.734 4.922 1.734zM3.984 12c0 1.781 0.656 3.516 1.734 4.922l11.203-11.203c-1.406-1.078-3.141-1.734-4.922-1.734-4.406 0-8.016 3.609-8.016 8.016zM12 2.016c5.484 0 9.984 4.5 9.984 9.984s-4.5 9.984-9.984 9.984-9.984-4.5-9.984-9.984 4.5-9.984 9.984-9.984z"></path>
</svg>
`)
//line template/webhook_index.qtpl:28
		qw422016.N().S(`</td> `)
//line template/webhook_index.qtpl:29
	}
//line template/webhook_index.qtpl:29
	qw422016.N().S(` `)
//line template/webhook_index.qtpl:30
	if w.LastDelivery == nil {
//line template/webhook_index.qtpl:30
		qw422016.N().S(` <td class="hook-status hook-status-none">`)
//line template/webhook_index.qtpl:31
		qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M18.984 6.422l-5.578 5.578 5.578 5.578-1.406 1.406-5.578-5.578-5.578 5.578-1.406-1.406 5.578-5.578-5.578-5.578 1.406-1.406 5.578 5.578 5.578-5.578z"></path>
</svg>
`)
//line template/webhook_index.qtpl:31
		qw422016.N().S(`</td> `)
//line template/webhook_index.qtpl:32
	} else {
//line template/webhook_index.qtpl:32
		qw422016.N().S(` `)
//line template/webhook_index.qtpl:33
		if w.LastDelivery.ResponseCode.Elem >= 200 && w.LastDelivery.ResponseCode.Elem < 300 {
//line template/webhook_index.qtpl:33
			qw422016.N().S(` <td class="hook-status hook-status-ok">`)
//line template/webhook_index.qtpl:34
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M9 16.172l10.594-10.594 1.406 1.406-12 12-5.578-5.578 1.406-1.406z"></path>
</svg>
`)
//line template/webhook_index.qtpl:34
			qw422016.N().S(`</td> `)
//line template/webhook_index.qtpl:35
		} else {
//line template/webhook_index.qtpl:35
			qw422016.N().S(` <td class="hook-status hook-status-err">`)
//line template/webhook_index.qtpl:36
			qw422016.N().S(`<!-- Generated by IcoMoon.io -->
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">
<path d="M18.984 6.422l-5.578 5.578 5.578 5.578-1.406 1.406-5.578-5.578-5.578 5.578-1.406-1.406 5.578-5.578-5.578-5.578 1.406-1.406 5.578 5.578 5.578-5.578z"></path>
</svg>
`)
//line template/webhook_index.qtpl:36
			qw422016.N().S(`</td> `)
//line template/webhook_index.qtpl:37
		}
//line template/webhook_index.qtpl:37
		qw422016.N().S(` `)
//line template/webhook_index.qtpl:38
	}
//line template/webhook_index.qtpl:38
	qw422016.N().S(` <td>`)
//line template/webhook_index.qtpl:39
	qw422016.E().S(w.Author.Username)
//line template/webhook_index.qtpl:39
	qw422016.N().S(`</td> <td><a href="`)
//line template/webhook_index.qtpl:40
	qw422016.E().S(w.Endpoint())
//line template/webhook_index.qtpl:40
	qw422016.N().S(`">`)
//line template/webhook_index.qtpl:40
	qw422016.E().S(w.PayloadURL.String())
//line template/webhook_index.qtpl:40
	qw422016.N().S(`</a></td> <td class="align-right"> <form method="POST" action="`)
//line template/webhook_index.qtpl:42
	qw422016.E().S(w.Endpoint())
//line template/webhook_index.qtpl:42
	qw422016.N().S(`"> `)
//line template/webhook_index.qtpl:43
	form.StreamMethod(qw422016, "DELETE")
//line template/webhook_index.qtpl:43
	qw422016.N().S(` `)
//line template/webhook_index.qtpl:44
	qw422016.N().V(p.CSRF)
//line template/webhook_index.qtpl:44
	qw422016.N().S(` <button type="submit" class="btn btn-danger">Delete</button> </form> </td> </tr> `)
//line template/webhook_index.qtpl:49
}

//line template/webhook_index.qtpl:49
func (p *WebhookIndex) writerenderWebhookItem(qq422016 qtio422016.Writer, w *namespace.Webhook) {
//line template/webhook_index.qtpl:49
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/webhook_index.qtpl:49
	p.streamrenderWebhookItem(qw422016, w)
//line template/webhook_index.qtpl:49
	qt422016.ReleaseWriter(qw422016)
//line template/webhook_index.qtpl:49
}

//line template/webhook_index.qtpl:49
func (p *WebhookIndex) renderWebhookItem(w *namespace.Webhook) string {
//line template/webhook_index.qtpl:49
	qb422016 := qt422016.AcquireByteBuffer()
//line template/webhook_index.qtpl:49
	p.writerenderWebhookItem(qb422016, w)
//line template/webhook_index.qtpl:49
	qs422016 := string(qb422016.B)
//line template/webhook_index.qtpl:49
	qt422016.ReleaseByteBuffer(qb422016)
//line template/webhook_index.qtpl:49
	return qs422016
//line template/webhook_index.qtpl:49
}

//line template/webhook_index.qtpl:51
func (p *WebhookIndex) StreamBody(qw422016 *qt422016.Writer) {
//line template/webhook_index.qtpl:51
	qw422016.N().S(` <div class="panel"> <div class="panel-header panel-body"> <a class="btn btn-primary right" href="`)
//line template/webhook_index.qtpl:54
	qw422016.E().S(p.Namespace.Endpoint("webhooks", "create"))
//line template/webhook_index.qtpl:54
	qw422016.N().S(`"> Create webhook </a> </div> `)
//line template/webhook_index.qtpl:58
	if len(p.Webhooks) == 0 {
//line template/webhook_index.qtpl:58
		qw422016.N().S(` <div class="panel-message muted">Add a webhook to notify external services of events.</div> `)
//line template/webhook_index.qtpl:60
	} else {
//line template/webhook_index.qtpl:60
		qw422016.N().S(` <table class="table"> <tbody> `)
//line template/webhook_index.qtpl:63
		for _, w := range p.Webhooks {
//line template/webhook_index.qtpl:63
			qw422016.N().S(` `)
//line template/webhook_index.qtpl:64
			p.streamrenderWebhookItem(qw422016, w)
//line template/webhook_index.qtpl:64
			qw422016.N().S(` `)
//line template/webhook_index.qtpl:65
		}
//line template/webhook_index.qtpl:65
		qw422016.N().S(` </tbody> </table> `)
//line template/webhook_index.qtpl:68
	}
//line template/webhook_index.qtpl:68
	qw422016.N().S(` </div> `)
//line template/webhook_index.qtpl:70
}

//line template/webhook_index.qtpl:70
func (p *WebhookIndex) WriteBody(qq422016 qtio422016.Writer) {
//line template/webhook_index.qtpl:70
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/webhook_index.qtpl:70
	p.StreamBody(qw422016)
//line template/webhook_index.qtpl:70
	qt422016.ReleaseWriter(qw422016)
//line template/webhook_index.qtpl:70
}

//line template/webhook_index.qtpl:70
func (p *WebhookIndex) Body() string {
//line template/webhook_index.qtpl:70
	qb422016 := qt422016.AcquireByteBuffer()
//line template/webhook_index.qtpl:70
	p.WriteBody(qb422016)
//line template/webhook_index.qtpl:70
	qs422016 := string(qb422016.B)
//line template/webhook_index.qtpl:70
	qt422016.ReleaseByteBuffer(qb422016)
//line template/webhook_index.qtpl:70
	return qs422016
//line template/webhook_index.qtpl:70
}
