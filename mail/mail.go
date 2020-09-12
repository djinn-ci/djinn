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
	From    string
	To      []string
	Subject string
	Body    string
}

type ClientConfig struct {
	CA       string
	Addr     string
	Username string
	Password string
}

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

	for k, v := range (*e) {
		buf.WriteString(k + ": " + v)

		if i != len((*e)) - 1 {
			buf.WriteRune('\n')
		}
		i++
	}
	return buf.String()
}
