// Package mail provides a simple SMTP client for sending plain-text emails.
package mail

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net"
	"net/smtp"
	"strings"

	"github.com/andrewpillar/djinn/errors"
)

type Mail struct {
	From    string   // From is the address we're sending the mail from.
	To      []string // To is the list of addresses to send the mail to.
	Subject string   // Subject is the subject line of the mail.
	Body    string   // Body is the body of the mail.
}

// ClientConfig specifies how the client connection to the SMTP server should
// be configured.
type ClientConfig struct {
	// CA is the path to the PEM encoded root CAs. If empty then TLS will not
	// be attempted upon connection to the SMTP server.
	CA string

	// Addr is the full address (host and port) of the SMTP server to connect
	// to.
	Addr string

	// Username and Password are the credentials to use the plain
	// authentication against the SMTP server. If none are provided then no
	// authentication attempts are made.
	Username string
	Password string
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

// NewClient will use the given ClientConfig to return an *smtp.Client.
// Depending on the fields present in the given config will determine whether
// authentication or TLS connectivity is made.
func NewClient(cfg ClientConfig) (*smtp.Client, error) {
	var (
		auth   smtp.Auth
		tlscfg *tls.Config
	)

	host, _, err := net.SplitHostPort(cfg.Addr)

	if err != nil {
		return nil, errors.Err(err)
	}

	if cfg.Username != "" && cfg.Password != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, host)
	}

	if cfg.CA != "" {
		b, err := ioutil.ReadFile(cfg.CA)

		if err != nil {
			return nil, errors.Err(err)
		}

		pool := x509.NewCertPool()

		if !pool.AppendCertsFromPEM(b) {
			return nil, errors.New("failed to append certificates from PEM, please check if valid")
		}

		tlscfg = &tls.Config{
			ServerName: host,
			RootCAs:    pool,
		}
	}

	cli, err := smtp.Dial(cfg.Addr)

	if err != nil {
		return nil, errors.Err(err)
	}

	if tlscfg != nil {
		if err := cli.StartTLS(tlscfg); err != nil {
			return nil, errors.Err(err)
		}
	}

	if auth != nil {
		if err := cli.Auth(auth); err != nil {
			return nil, errors.Err(err)
		}
	}
	return cli, nil
}

// String returns the string representation of the current mail. This is
// typically what's written to the SMTP server once the DATA command has been
// issued.
func (m Mail) String() string {
	var buf strings.Builder

	writeField(&buf, "From", m.From)
	writeField(&buf, "To", strings.Join(m.To, "; "))
	writeField(&buf, "Subject", m.Subject)

	buf.WriteString("\r\n")
	buf.WriteString(m.Body)

	return buf.String()
}

// Send builds up the current Mail into something that can be sent to the given
// smtp.Client. If any errors occur when adding a recipient via RCPT, then an
// attempt to send the mail will still be done, and the ErrRcpts type will be
// returned.
func (m Mail) Send(cli *smtp.Client) error {
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
