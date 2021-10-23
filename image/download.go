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
	"path/filepath"
	"strings"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/log"
	"djinn-ci.com/queue"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"

	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"
)

type DownloadJob struct {
	db    *sqlx.DB    `gob:"-" msgpack:"-"`
	log   *log.Logger `gob:"-" msgpack:"-"`
	queue queue.Queue `gob:"-" msgpack:"-"`
	store fs.Store    `gob:"-" msgpack:"-"`

	ID      int64
	ImageID int64
	Source  DownloadURL
}

type Download struct {
	ID         int64          `db:"id"`
	ImageID    int64          `db:"image_id"`
	Source     DownloadURL    `db:"source"`
	Error      sql.NullString `db:"error"`
	CreatedAt  time.Time      `db:"created_at"`
	StartedAt  sql.NullTime   `db:"started_at"`
	FinishedAt sql.NullTime   `db:"finished_at"`

	User  *user.User `db:"-"`
	Image *Image     `db:"-"`
}

type DownloadURL struct {
	*url.URL
}

type DownloadStore struct {
	database.Store

	User  *user.User
	Image *Image
}

var (
	_ database.Binder = (*DownloadStore)(nil)

	_ sql.Scanner   = (*DownloadURL)(nil)
	_ driver.Valuer = (*DownloadURL)(nil)

	_ queue.Job = (*DownloadJob)(nil)

	downloadTable   = "image_downloads"
	downloadSchemes = map[string]struct{}{
		"http":  {},
		"https": {},
		"sftp":  {},
	}

	MimeTypeQEMU = "application/x-qemu-disk"

	ErrInvalidScheme = errors.New("invalid url scheme")
)

// NewDownloadJob creates the underlying Download for the given Image and
// returns it as a queue.Job for background processing.
func NewDownloadJob(db *sqlx.DB, i *Image, url DownloadURL) (queue.Job, error) {
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

func NewDownloadStore(db *sqlx.DB, mm ...database.Model) *DownloadStore {
	s := &DownloadStore{
		Store: database.Store{DB: db},
	}
	s.Bind(mm...)
	return s
}

func (s *DownloadStore) Bind(mm ...database.Model) {
	for _, m := range mm {
		switch v := m.(type) {
		case *user.User:
			s.User = v
		case *Image:
			s.Image = v
		}
	}
}

func (s *DownloadStore) Get(opts ...query.Option) (*Download, error) {
	d := &Download{
		User:  s.User,
		Image: s.Image,
	}

	opts = append([]query.Option{
		database.Where(s.Image, "image_id"),
	}, opts...)

	if s.User != nil {
		opts = append([]query.Option{
			query.Where("image_id", "IN", query.Select(
				query.Columns("id"),
				query.From(table),
				query.Where("user_id", "=", query.Arg(s.User.ID)),
			)),
		}, opts...)
	}

	err := s.Store.Get(d, downloadTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}
	return d, errors.Err(err)
}

func (s *DownloadStore) All(opts ...query.Option) ([]*Download, error) {
	dd := make([]*Download, 0)

	opts = append([]query.Option{
		database.Where(s.Image, "image_id"),
	}, opts...)

	if s.User != nil {
		opts = append([]query.Option{
			query.Where("image_id", "IN", query.Select(
				query.Columns("id"),
				query.From(table),
				query.Where("user_id", "=", query.Arg(s.User.ID)),
			)),
		}, opts...)
	}

	err := s.Store.All(&dd, downloadTable, opts...)

	if err == sql.ErrNoRows {
		err = nil
	}

	for _, d := range dd {
		d.User = s.User
		d.Image = s.Image
	}
	return dd, errors.Err(err)
}

// DownloadJobInit returns a callback for initializing a download job with the
// given database connection and file store.
func DownloadJobInit(db *sqlx.DB, q queue.Queue, log *log.Logger, store fs.Store) queue.InitFunc {
	return func(j queue.Job) {
		if d, ok := j.(*DownloadJob); ok {
			d.db = db
			d.queue = q
			d.log = log
			d.store = store
		}
	}
}

// Name implements the queue.Job interface.
func (d *DownloadJob) Name() string { return "download_job" }

// Perform implements the queue.Job interface. This will attempt to download
// the image from the remote URL. This will fail if the URL does not have a
// valid scheme such as, http, https, or sftp.
func (d *DownloadJob) Perform() error {
	now := time.Now()

	q := query.Update(
		downloadTable,
		query.Set("started_at", query.Arg(now)),
		query.Where("id", "=", query.Arg(d.ID)),
	)

	d.log.Debug.Println("image download started at", now)

	if _, err := d.db.Exec(q.Build(), q.Args()...); err != nil {
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

		if !bytes.Equal(buf, qcow) {
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

	i, err := NewStore(d.db).Get(query.Where("id", "=", query.Arg(d.ImageID)))

	if err != nil {
		d.log.Error.Println(errors.Err(err))
		return errors.Err(err)
	}

	if r != nil {
		dst, err := d.store.Create(filepath.Join(i.Driver.String(), i.Hash))

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

	if _, err := d.db.Exec(q.Build(), q.Args()...); err != nil {
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

func (u *DownloadURL) Scan(v interface{}) error {
	if v == nil {
		return nil
	}

	b, err := database.Scan(v)

	if err != nil {
		return errors.Err(err)
	}

	u.URL, err = url.Parse(string(b))
	return errors.Err(err)
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
	return nil
}

func (u *DownloadURL) Validate() error {
	if _, ok := downloadSchemes[u.Scheme]; ok {
		return ErrInvalidScheme
	}
	return nil
}

func (u DownloadURL) Value() (driver.Value, error) { return strings.ToLower(u.Redacted()), nil }
