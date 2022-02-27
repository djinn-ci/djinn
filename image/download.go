package image

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"
	"encoding/gob"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/log"
	"djinn-ci.com/queue"

	"github.com/andrewpillar/query"

	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"
)

type DownloadJob struct {
	log   *log.Logger `msgpack:"-"`
	queue queue.Queue `msgpack:"-"`
	store *Store      `msgpack:"-"`

	ID      int64
	ImageID int64
	Source  DownloadURL
}

var (
	_ queue.Job = (*DownloadJob)(nil)

	downloadSchemes = map[string]struct{}{
		"http":  {},
		"https": {},
		"sftp":  {},
	}

	MimeTypeQEMU = "application/x-qemu-disk"

	ErrInvalidScheme   = errors.New("invalid url scheme")
	ErrMissingPassword = errors.New("missing password in url")
)

func NewDownloadJob(db database.Pool, i *Image, url DownloadURL) (queue.Job, error) {
	q := query.Insert(
		downloadTable,
		query.Columns("image_id", "source", "created_at"),
		query.Values(i.ID, url, time.Now()),
		query.Returning("id"),
	)

	var id int64

	if err := db.QueryRow(q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}
	return &DownloadJob{
		ID:      id,
		ImageID: i.ID,
		Source:  url,
	}, nil
}

func DownloadJobInit(db database.Pool, q queue.Queue, log *log.Logger, store fs.Store) queue.InitFunc {
	return func(j queue.Job) {
		if d, ok := j.(*DownloadJob); ok {
			d.store = &Store{
				Pool:  db,
				Store: store,
			}
			d.queue = q
			d.log = log
		}
	}
}

func (d *DownloadJob) Name() string { return "download_job" }

var QCOW2Sig = []byte{0x51, 0x46, 0x49, 0xFB}

func (d *DownloadJob) Perform() error {
	now := time.Now()

	q := query.Update(
		downloadTable,
		query.Set("started_at", query.Arg(now)),
		query.Where("id", "=", query.Arg(d.ID)),
	)

	d.log.Debug.Println("image download started at", now)

	if _, err := d.store.Exec(q.Build(), q.Args()...); err != nil {
		d.log.Error.Println(errors.Err(err))
		return errors.Err(err)
	}

	var (
		tlscfg      *tls.Config
		downloaderr sql.NullString
	)

	var r io.Reader

	d.log.Debug.Println("downloading image from", d.Source)

	switch d.Source.Scheme {
	case "https":
		pool, err := x509.SystemCertPool()

		if err != nil {
			d.log.Error.Println(errors.Err(err))
			return errors.Err(err)
		}

		tlscfg = &tls.Config{
			ServerName: d.Source.Host,
			RootCAs:    pool,
		}
		fallthrough
	case "http":
		cli := &http.Client{
			Timeout: time.Minute * 10,
			Transport: &http.Transport{
				TLSClientConfig: tlscfg,
			},
		}

		req := &http.Request{
			Method: "GET",
			URL:    d.Source.URL,
		}

		if d.Source.User != nil {
			auth := []byte(d.Source.User.String())

			req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString(auth))
		}

		resp, err := cli.Do(req)

		if err != nil {
			downloaderr.String = err.Error()
			downloaderr.Valid = true
			break
		}

		defer resp.Body.Close()

		if resp.Header.Get("Content-Type") != MimeTypeQEMU {
			downloaderr.String = "unexpected Content-Type, expected " + MimeTypeQEMU
			downloaderr.Valid = true
			break
		}
		r = resp.Body
	case "sftp":
		timeout := time.Minute

		if dur := d.Source.Query().Get("timeout"); dur != "" {
			dur, err := time.ParseDuration(dur)

			if err == nil {
				timeout = dur
			}
		}

		password, _ := d.Source.User.Password()

		cfg := &ssh.ClientConfig{
			User: d.Source.User.Username(),
			Auth: []ssh.AuthMethod{
				ssh.Password(password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         timeout,
		}

		sshcli, err := ssh.Dial("tcp", d.Source.Host, cfg)

		if err != nil {
			downloaderr.String = err.Error()
			downloaderr.Valid = true
			break
		}

		defer sshcli.Close()

		cli, err := sftp.NewClient(sshcli)

		if err != nil {
			downloaderr.String = err.Error()
			downloaderr.Valid = true
			break
		}

		defer cli.Close()

		f, err := cli.Open(d.Source.Path)

		if err != nil {
			downloaderr.String = err.Error()
			downloaderr.Valid = true
			break
		}

		defer f.Close()

		buf := make([]byte, 0, 4)

		if _, err := f.Read(buf); err != nil {
			downloaderr.String = err.Error()
			downloaderr.Valid = true
			break
		}

		if !bytes.Equal(buf, QCOW2Sig) {
			downloaderr.String = "not a valid qcow2 file"
			downloaderr.Valid = true
			break
		}

		if _, err := f.Seek(0, io.SeekStart); err != nil {
			downloaderr.String = err.Error()
			downloaderr.Valid = true
			break
		}
		r = f
	default:
		return ErrInvalidScheme
	}

	i, _, err := d.store.Get(query.Where("id", "=", query.Arg(d.ImageID)))

	if err != nil {
		d.log.Error.Println(errors.Err(err))
		return errors.Err(err)
	}

	if r != nil {
		store, err := d.store.Partition(i.UserID)

		if err != nil {
			d.log.Error.Println(errors.Err(err))
			return errors.Err(err)
		}

		dst, err := store.Create(i.path())

		if err != nil {
			downloaderr.String = errors.Cause(err).Error()
			downloaderr.Valid = true
			goto update
		}

		if _, err := io.Copy(dst, r); err != nil {
			downloaderr.String = errors.Cause(err).Error()
			downloaderr.Valid = true
		}
	}

update:
	q = query.Update(
		downloadTable,
		query.Set("error", query.Arg(downloaderr)),
		query.Set("finished_at", query.Arg(time.Now())),
		query.Where("id", "=", query.Arg(d.ID)),
	)

	if _, err := d.store.Exec(q.Build(), q.Args()...); err != nil {
		d.log.Error.Println(errors.Err(err))
		return errors.Err(err)
	}

	d.log.Debug.Println("dispatching 'downloaded' event")

	d.queue.Produce(context.Background(), &Event{
		Image:  i,
		Action: "downloaded",
	})
	return nil
}

type DownloadURL struct {
	*url.URL
}

var (
	_ sql.Scanner   = (*DownloadURL)(nil)
	_ driver.Valuer = (*DownloadURL)(nil)
)

func (u *DownloadURL) GobEncode() ([]byte, error) {
	var buf bytes.Buffer

	if err := gob.NewEncoder(&buf).Encode(u.URL); err != nil {
		return nil, errors.Err(err)
	}
	return buf.Bytes(), nil
}

func (u *DownloadURL) GobDecode(p []byte) error {
	var url url.URL

	if err := gob.NewDecoder(bytes.NewReader(p)).Decode(&url); err != nil {
		return errors.Err(err)
	}

	u.URL = &url
	return nil
}

func (u *DownloadURL) String() string {
	if u.URL != nil {
		return u.URL.String()
	}
	return ""
}

func (u *DownloadURL) Scan(val interface{}) error {
	if val == nil {
		return nil
	}

	v, err := driver.String.ConvertValue(val)

	if err != nil {
		return errors.Err(err)
	}

	s, ok := v.(string)

	if !ok {
		return errors.New("image: could not type assert DownloadURL to string")
	}

	u.URL, err = url.Parse(s)

	if err != nil {
		return errors.Err(err)
	}
	return nil
}

func (u *DownloadURL) UnmarshalText(b []byte) error {
	var err error

	if len(b) == 0 {
		return nil
	}

	u.URL, err = url.Parse(string(b))

	if err != nil {
		return errors.Err(err)
	}

	if _, ok := downloadSchemes[u.Scheme]; !ok {
		return ErrInvalidScheme
	}

	if u.Scheme == "sftp" {
		if _, ok := u.User.Password(); !ok {
			return ErrMissingPassword
		}
	}
	return nil
}

func (u *DownloadURL) Validate() error {
	if _, ok := downloadSchemes[u.Scheme]; ok {
		return ErrInvalidScheme
	}
	return nil
}

func (u DownloadURL) Value() (driver.Value, error) { return strings.ToLower(u.Redacted()), nil }

type Download struct {
	ID         int64
	ImageID    int64
	Source     DownloadURL
	Error      sql.NullString
	CreatedAt  time.Time
	StartedAt  sql.NullTime
	FinishedAt sql.NullTime
}

var _ database.Model = (*Download)(nil)

func (d *Download) Dest() []interface{} {
	return []interface{}{
		&d.ID,
		&d.ImageID,
		&d.Source,
		&d.Error,
		&d.CreatedAt,
		&d.StartedAt,
		&d.FinishedAt,
	}
}

func (*Download) Bind(_ database.Model) {}

func (*Download) JSON(_ string) map[string]interface{} { return nil }

func (*Download) Endpoint(_ ...string) string { return "" }

func (d *Download) Values() map[string]interface{} {
	return map[string]interface{}{
		"id":          d.ID,
		"image_id":    d.ImageID,
		"source":      d.Source,
		"error":       d.Error,
		"created_at":  d.CreatedAt,
		"started_at":  d.StartedAt,
		"finished_at": d.FinishedAt,
	}
}

type DownloadStore struct {
	database.Pool
}

var (
	_ database.Loader = (*DownloadStore)(nil)

	downloadTable = "image_downloads"
)

func (s DownloadStore) Get(opts ...query.Option) (*Download, bool, error) {
	var d Download

	ok, err := s.Pool.Get(downloadTable, &d, opts...)

	if err != nil {
		return nil, false, errors.Err(err)
	}

	if !ok {
		return nil, false, nil
	}
	return &d, ok, nil
}

func (s DownloadStore) All(opts ...query.Option) ([]*Download, error) {
	dd := make([]*Download, 0)

	new := func() database.Model {
		d := &Download{}
		dd = append(dd, d)
		return d
	}

	if err := s.Pool.All(downloadTable, new, opts...); err != nil {
		return nil, errors.Err(err)
	}
	return dd, nil
}

func (s DownloadStore) Load(fk, pk string, mm ...database.Model) error {
	vals := database.Values(fk, mm)

	dd, err := s.All(query.Where(pk, "IN", database.List(vals...)))

	if err != nil {
		return errors.Err(err)
	}

	for _, d := range dd {
		for _, m := range mm {
			m.Bind(d)
		}
	}
	return nil
}
