// Package mail provides a simple SMTP client for sending plain-text emails.
package mail

import (
	"crypto/tls"
	"net"
	"net/smtp"
	"strings"

	"djinn-ci.com/errors"
	"djinn-ci.com/queue"
)

type Client struct {
	*smtp.Client

	Addr      string
	Auth      smtp.Auth
	TLSConfig *tls.Config
}

type Mail struct {
	cli *Client

	From    string   // From is the address we're sending the mail from.
	To      []string // To is the list of addresses to send the mail to.
	Subject string   // Subject is the subject line of the mail.
	Body    string   // Body is the body of the mail.
}

// ErrRcpts is a list of any errors that occur when a RCPT command is send to
// the SMTP server. This will store each error message against the
// corresponding recipient that caused the error.
type ErrRcpts map[string]string

func writeField(buf *strings.Builder, field, val string) {
	buf.WriteString(field)
	buf.WriteString(": ")
	buf.WriteString(val)
	buf.WriteString("\r\n")
}

func (c *Client) Dial() error {
	conn, err := net.Dial("tcp", c.Addr)

	if err != nil {
		return errors.Err(err)
	}

	host, _, _ := net.SplitHostPort(c.Addr)

	cli, err := smtp.NewClient(conn, host)

	if err != nil {
		return errors.Err(err)
	}

	if c.TLSConfig != nil {
		if err := cli.StartTLS(c.TLSConfig); err != nil {
			return errors.Err(err)
		}
	}

	if c.Auth != nil {
		if err := cli.Auth(c.Auth); err != nil {
			return errors.Err(err)
		}
	}

	c.Client = cli
	return nil
}

// String returns the string representation of the current mail. This is
// typically what's written to the SMTP server once the DATA command has been
// issued.
func (m *Mail) String() string {
	var buf strings.Builder

	writeField(&buf, "From", m.From)
	writeField(&buf, "To", strings.Join(m.To, "; "))
	writeField(&buf, "Subject", m.Subject)

	buf.WriteString("\r\n")
	buf.WriteString(m.Body)

	return buf.String()
}

func InitJob(cli *Client) queue.InitFunc {
	return func(j queue.Job) {
		if m, ok := j.(*Mail); ok {
			m.cli = cli
		}
	}
}

func (m *Mail) Name() string {
	return "email"
}

func (m *Mail) Perform() error {
	return m.Send(m.cli)
}

// Send builds up the current Mail into something that can be sent to the given
// smtp.Client. If any errors occur when adding a recipient via RCPT, then an
// attempt to send the mail will still be done, and the ErrRcpts type will be
// returned.
func (m *Mail) Send(cli *Client) error {
	if err := cli.Reset(); err != nil {
		// Failure could be due to a broken pipe, so attempt to redial.
		if err := cli.Dial(); err != nil {
			return errors.Err(err)
		}
	}

	if err := cli.Mail(m.From); err != nil {
		return errors.Err(err)
	}

	rcpterrs := ErrRcpts(make(map[string]string))

	for _, rcpt := range m.To {
		if err := cli.Rcpt(rcpt); err != nil {
			rcpterrs[rcpt] = err.Error()
		}
	}

	w, err := cli.Data()

	if err != nil {
		return errors.Err(err)
	}

	defer w.Close()

	if _, err := w.Write([]byte(m.String())); err != nil {
		return errors.Err(err)
	}
	return rcpterrs.err()
}

func (e *ErrRcpts) err() error {
	if len((*e)) > 0 {
		return e
	}
	return nil
}

// Error returns a formatted string of the recipients that couldn't received
// the email alongside ther original error.
func (e *ErrRcpts) Error() string {
	var buf strings.Builder

	i := 0

	for k, v := range *e {
		buf.WriteString(k + ": " + v)

		if i != len((*e))-1 {
			buf.WriteRune('\n')
		}
		i++
	}
	return buf.String()
}
