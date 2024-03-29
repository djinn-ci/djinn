// Code generated by qtc from "invite_index.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line template/invite_index.qtpl:2
package template

//line template/invite_index.qtpl:2
import (
	"djinn-ci.com/namespace"
	"djinn-ci.com/template/form"
)

//line template/invite_index.qtpl:8
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line template/invite_index.qtpl:8
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line template/invite_index.qtpl:9
type InviteIndex struct {
	*form.Form

	Namespace *namespace.Namespace
	Invites   []*namespace.Invite
}

//line template/invite_index.qtpl:18
func (p *InviteIndex) StreamTitle(qw422016 *qt422016.Writer) {
//line template/invite_index.qtpl:18
	qw422016.N().S(`Invites`)
//line template/invite_index.qtpl:18
}

//line template/invite_index.qtpl:18
func (p *InviteIndex) WriteTitle(qq422016 qtio422016.Writer) {
//line template/invite_index.qtpl:18
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/invite_index.qtpl:18
	p.StreamTitle(qw422016)
//line template/invite_index.qtpl:18
	qt422016.ReleaseWriter(qw422016)
//line template/invite_index.qtpl:18
}

//line template/invite_index.qtpl:18
func (p *InviteIndex) Title() string {
//line template/invite_index.qtpl:18
	qb422016 := qt422016.AcquireByteBuffer()
//line template/invite_index.qtpl:18
	p.WriteTitle(qb422016)
//line template/invite_index.qtpl:18
	qs422016 := string(qb422016.B)
//line template/invite_index.qtpl:18
	qt422016.ReleaseByteBuffer(qb422016)
//line template/invite_index.qtpl:18
	return qs422016
//line template/invite_index.qtpl:18
}

//line template/invite_index.qtpl:20
func (p *InviteIndex) StreamHeader(qw422016 *qt422016.Writer) {
//line template/invite_index.qtpl:20
	p.StreamTitle(qw422016)
//line template/invite_index.qtpl:20
}

//line template/invite_index.qtpl:20
func (p *InviteIndex) WriteHeader(qq422016 qtio422016.Writer) {
//line template/invite_index.qtpl:20
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/invite_index.qtpl:20
	p.StreamHeader(qw422016)
//line template/invite_index.qtpl:20
	qt422016.ReleaseWriter(qw422016)
//line template/invite_index.qtpl:20
}

//line template/invite_index.qtpl:20
func (p *InviteIndex) Header() string {
//line template/invite_index.qtpl:20
	qb422016 := qt422016.AcquireByteBuffer()
//line template/invite_index.qtpl:20
	p.WriteHeader(qb422016)
//line template/invite_index.qtpl:20
	qs422016 := string(qb422016.B)
//line template/invite_index.qtpl:20
	qt422016.ReleaseByteBuffer(qb422016)
//line template/invite_index.qtpl:20
	return qs422016
//line template/invite_index.qtpl:20
}

//line template/invite_index.qtpl:22
func (p *InviteIndex) StreamActions(qw422016 *qt422016.Writer) {
//line template/invite_index.qtpl:22
}

//line template/invite_index.qtpl:22
func (p *InviteIndex) WriteActions(qq422016 qtio422016.Writer) {
//line template/invite_index.qtpl:22
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/invite_index.qtpl:22
	p.StreamActions(qw422016)
//line template/invite_index.qtpl:22
	qt422016.ReleaseWriter(qw422016)
//line template/invite_index.qtpl:22
}

//line template/invite_index.qtpl:22
func (p *InviteIndex) Actions() string {
//line template/invite_index.qtpl:22
	qb422016 := qt422016.AcquireByteBuffer()
//line template/invite_index.qtpl:22
	p.WriteActions(qb422016)
//line template/invite_index.qtpl:22
	qs422016 := string(qb422016.B)
//line template/invite_index.qtpl:22
	qt422016.ReleaseByteBuffer(qb422016)
//line template/invite_index.qtpl:22
	return qs422016
//line template/invite_index.qtpl:22
}

//line template/invite_index.qtpl:23
func (p *InviteIndex) StreamNavigation(qw422016 *qt422016.Writer) {
//line template/invite_index.qtpl:23
}

//line template/invite_index.qtpl:23
func (p *InviteIndex) WriteNavigation(qq422016 qtio422016.Writer) {
//line template/invite_index.qtpl:23
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/invite_index.qtpl:23
	p.StreamNavigation(qw422016)
//line template/invite_index.qtpl:23
	qt422016.ReleaseWriter(qw422016)
//line template/invite_index.qtpl:23
}

//line template/invite_index.qtpl:23
func (p *InviteIndex) Navigation() string {
//line template/invite_index.qtpl:23
	qb422016 := qt422016.AcquireByteBuffer()
//line template/invite_index.qtpl:23
	p.WriteNavigation(qb422016)
//line template/invite_index.qtpl:23
	qs422016 := string(qb422016.B)
//line template/invite_index.qtpl:23
	qt422016.ReleaseByteBuffer(qb422016)
//line template/invite_index.qtpl:23
	return qs422016
//line template/invite_index.qtpl:23
}

//line template/invite_index.qtpl:24
func (p *InviteIndex) StreamFooter(qw422016 *qt422016.Writer) {
//line template/invite_index.qtpl:24
}

//line template/invite_index.qtpl:24
func (p *InviteIndex) WriteFooter(qq422016 qtio422016.Writer) {
//line template/invite_index.qtpl:24
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/invite_index.qtpl:24
	p.StreamFooter(qw422016)
//line template/invite_index.qtpl:24
	qt422016.ReleaseWriter(qw422016)
//line template/invite_index.qtpl:24
}

//line template/invite_index.qtpl:24
func (p *InviteIndex) Footer() string {
//line template/invite_index.qtpl:24
	qb422016 := qt422016.AcquireByteBuffer()
//line template/invite_index.qtpl:24
	p.WriteFooter(qb422016)
//line template/invite_index.qtpl:24
	qs422016 := string(qb422016.B)
//line template/invite_index.qtpl:24
	qt422016.ReleaseByteBuffer(qb422016)
//line template/invite_index.qtpl:24
	return qs422016
//line template/invite_index.qtpl:24
}

//line template/invite_index.qtpl:26
func (p *InviteIndex) streamrenderInviteForm(qw422016 *qt422016.Writer) {
//line template/invite_index.qtpl:26
	qw422016.N().S(` <form method="POST" action="`)
//line template/invite_index.qtpl:27
	qw422016.E().S(p.Namespace.Endpoint("invites"))
//line template/invite_index.qtpl:27
	qw422016.N().S(`"> `)
//line template/invite_index.qtpl:28
	qw422016.N().V(p.CSRF)
//line template/invite_index.qtpl:28
	qw422016.N().S(` <div class="form-field form-field-inline"> <input type="text" class="form-text" name="handle" placeholder="Invite user..." autocomplete="off"/> <button type="submit" class="btn btn-primary">Invite</button> `)
//line template/invite_index.qtpl:32
	p.StreamError(qw422016, "handle")
//line template/invite_index.qtpl:32
	qw422016.N().S(` </div> </form> `)
//line template/invite_index.qtpl:35
}

//line template/invite_index.qtpl:35
func (p *InviteIndex) writerenderInviteForm(qq422016 qtio422016.Writer) {
//line template/invite_index.qtpl:35
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/invite_index.qtpl:35
	p.streamrenderInviteForm(qw422016)
//line template/invite_index.qtpl:35
	qt422016.ReleaseWriter(qw422016)
//line template/invite_index.qtpl:35
}

//line template/invite_index.qtpl:35
func (p *InviteIndex) renderInviteForm() string {
//line template/invite_index.qtpl:35
	qb422016 := qt422016.AcquireByteBuffer()
//line template/invite_index.qtpl:35
	p.writerenderInviteForm(qb422016)
//line template/invite_index.qtpl:35
	qs422016 := string(qb422016.B)
//line template/invite_index.qtpl:35
	qt422016.ReleaseByteBuffer(qb422016)
//line template/invite_index.qtpl:35
	return qs422016
//line template/invite_index.qtpl:35
}

//line template/invite_index.qtpl:37
func (p *InviteIndex) streamrenderReceivedInvites(qw422016 *qt422016.Writer) {
//line template/invite_index.qtpl:37
	qw422016.N().S(` <table class="table"> <thead> <tr> <th>NAMESPACE</th> <th>INVITED BY</th> <th></th> </tr> </thead> <tbody> `)
//line template/invite_index.qtpl:47
	for _, i := range p.Invites {
//line template/invite_index.qtpl:47
		qw422016.N().S(` <tr> <td>`)
//line template/invite_index.qtpl:49
		qw422016.E().S(i.Namespace.Path)
//line template/invite_index.qtpl:49
		qw422016.N().S(`</td> <td>`)
//line template/invite_index.qtpl:50
		qw422016.E().S(i.Inviter.Username)
//line template/invite_index.qtpl:50
		qw422016.N().S(`</td> <td class="align-right"> <form method="POST" action="`)
//line template/invite_index.qtpl:52
		qw422016.E().S(i.Endpoint())
//line template/invite_index.qtpl:52
		qw422016.N().S(`"> `)
//line template/invite_index.qtpl:53
		qw422016.N().V(p.CSRF)
//line template/invite_index.qtpl:53
		qw422016.N().S(` <input type="hidden" name="_method" value="PATCH"/> <button type="submit" class="btn btn-primary">Accept</button> </form> <form method="POST" action="`)
//line template/invite_index.qtpl:57
		qw422016.E().S(i.Endpoint())
//line template/invite_index.qtpl:57
		qw422016.N().S(`"> `)
//line template/invite_index.qtpl:58
		qw422016.N().V(p.CSRF)
//line template/invite_index.qtpl:58
		qw422016.N().S(` <input type="hidden" name="_method" value="DELETE"/> <button type="submit" class="btn btn-danger">Reject</button> </form> </td> </tr> `)
//line template/invite_index.qtpl:64
	}
//line template/invite_index.qtpl:64
	qw422016.N().S(` </tbody> </table> `)
//line template/invite_index.qtpl:67
}

//line template/invite_index.qtpl:67
func (p *InviteIndex) writerenderReceivedInvites(qq422016 qtio422016.Writer) {
//line template/invite_index.qtpl:67
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/invite_index.qtpl:67
	p.streamrenderReceivedInvites(qw422016)
//line template/invite_index.qtpl:67
	qt422016.ReleaseWriter(qw422016)
//line template/invite_index.qtpl:67
}

//line template/invite_index.qtpl:67
func (p *InviteIndex) renderReceivedInvites() string {
//line template/invite_index.qtpl:67
	qb422016 := qt422016.AcquireByteBuffer()
//line template/invite_index.qtpl:67
	p.writerenderReceivedInvites(qb422016)
//line template/invite_index.qtpl:67
	qs422016 := string(qb422016.B)
//line template/invite_index.qtpl:67
	qt422016.ReleaseByteBuffer(qb422016)
//line template/invite_index.qtpl:67
	return qs422016
//line template/invite_index.qtpl:67
}

//line template/invite_index.qtpl:69
func (p *InviteIndex) streamrenderSentInvites(qw422016 *qt422016.Writer) {
//line template/invite_index.qtpl:69
	qw422016.N().S(` <table class="table"> <thead> <tr> <th>USER</th> <th></th> </tr> </thead> <tbody> `)
//line template/invite_index.qtpl:78
	for _, i := range p.Invites {
//line template/invite_index.qtpl:78
		qw422016.N().S(` <td>`)
//line template/invite_index.qtpl:79
		qw422016.E().S(i.Invitee.Username)
//line template/invite_index.qtpl:79
		qw422016.N().S(`</td> <td class="align-right"> <form method="POST" action="`)
//line template/invite_index.qtpl:81
		qw422016.E().S(i.Endpoint())
//line template/invite_index.qtpl:81
		qw422016.N().S(`"> `)
//line template/invite_index.qtpl:82
		qw422016.N().V(p.CSRF)
//line template/invite_index.qtpl:82
		qw422016.N().S(` <input type="hidden" name="_method" value="DELETE"/> <button type="submit" class="btn btn-danger">Revoke</button> </form> </td> `)
//line template/invite_index.qtpl:87
	}
//line template/invite_index.qtpl:87
	qw422016.N().S(` </tbody> </table> `)
//line template/invite_index.qtpl:90
}

//line template/invite_index.qtpl:90
func (p *InviteIndex) writerenderSentInvites(qq422016 qtio422016.Writer) {
//line template/invite_index.qtpl:90
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/invite_index.qtpl:90
	p.streamrenderSentInvites(qw422016)
//line template/invite_index.qtpl:90
	qt422016.ReleaseWriter(qw422016)
//line template/invite_index.qtpl:90
}

//line template/invite_index.qtpl:90
func (p *InviteIndex) renderSentInvites() string {
//line template/invite_index.qtpl:90
	qb422016 := qt422016.AcquireByteBuffer()
//line template/invite_index.qtpl:90
	p.writerenderSentInvites(qb422016)
//line template/invite_index.qtpl:90
	qs422016 := string(qb422016.B)
//line template/invite_index.qtpl:90
	qt422016.ReleaseByteBuffer(qb422016)
//line template/invite_index.qtpl:90
	return qs422016
//line template/invite_index.qtpl:90
}

//line template/invite_index.qtpl:92
func (p *InviteIndex) StreamBody(qw422016 *qt422016.Writer) {
//line template/invite_index.qtpl:92
	qw422016.N().S(` <div class="panel"> `)
//line template/invite_index.qtpl:94
	if p.Namespace != nil {
//line template/invite_index.qtpl:94
		qw422016.N().S(` <div class="panel-header panel-body">`)
//line template/invite_index.qtpl:95
		p.streamrenderInviteForm(qw422016)
//line template/invite_index.qtpl:95
		qw422016.N().S(`</div> `)
//line template/invite_index.qtpl:96
	}
//line template/invite_index.qtpl:96
	qw422016.N().S(` `)
//line template/invite_index.qtpl:97
	if len(p.Invites) == 0 {
//line template/invite_index.qtpl:97
		qw422016.N().S(` <div class="panel-message muted">No new namespace invites.</div> `)
//line template/invite_index.qtpl:99
	} else {
//line template/invite_index.qtpl:99
		qw422016.N().S(` `)
//line template/invite_index.qtpl:100
		if p.Namespace == nil {
//line template/invite_index.qtpl:100
			qw422016.N().S(` `)
//line template/invite_index.qtpl:101
			p.streamrenderReceivedInvites(qw422016)
//line template/invite_index.qtpl:101
			qw422016.N().S(` `)
//line template/invite_index.qtpl:102
		} else {
//line template/invite_index.qtpl:102
			qw422016.N().S(` `)
//line template/invite_index.qtpl:103
			p.streamrenderSentInvites(qw422016)
//line template/invite_index.qtpl:103
			qw422016.N().S(` `)
//line template/invite_index.qtpl:104
		}
//line template/invite_index.qtpl:104
		qw422016.N().S(` `)
//line template/invite_index.qtpl:105
	}
//line template/invite_index.qtpl:105
	qw422016.N().S(` </div> `)
//line template/invite_index.qtpl:107
}

//line template/invite_index.qtpl:107
func (p *InviteIndex) WriteBody(qq422016 qtio422016.Writer) {
//line template/invite_index.qtpl:107
	qw422016 := qt422016.AcquireWriter(qq422016)
//line template/invite_index.qtpl:107
	p.StreamBody(qw422016)
//line template/invite_index.qtpl:107
	qt422016.ReleaseWriter(qw422016)
//line template/invite_index.qtpl:107
}

//line template/invite_index.qtpl:107
func (p *InviteIndex) Body() string {
//line template/invite_index.qtpl:107
	qb422016 := qt422016.AcquireByteBuffer()
//line template/invite_index.qtpl:107
	p.WriteBody(qb422016)
//line template/invite_index.qtpl:107
	qs422016 := string(qb422016.B)
//line template/invite_index.qtpl:107
	qt422016.ReleaseByteBuffer(qb422016)
//line template/invite_index.qtpl:107
	return qs422016
//line template/invite_index.qtpl:107
}
