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
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/log"
	"djinn-ci.com/queue"

	"github.com/andrewpillar/fs"
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

	MimeTypeQEMU = "application/x-qemu-disk"

	ErrInvalidScheme = errors.New("image: invalid url scheme")
)

func NewDownloadJob(pool *database.Pool, i *Image, url DownloadURL) (queue.Job, error) {
	q := query.Insert(
		downloadTable,
		query.Columns("image_id", "source", "created_at"),
		query.Values(i.ID, url, time.Now()),
		query.Returning("id"),
	)

	var id int64

	if err := pool.QueryRow(context.Background(), q.Build(), q.Args()...).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}
	return &DownloadJob{
		ID:      id,
		ImageID: i.ID,
		Source:  url,
	}, nil
}

func DownloadJobInit(pool *database.Pool, q queue.Queue, log *log.Logger, fs fs.FS) queue.InitFunc {
	return func(j queue.Job) {
		if d, ok := j.(*DownloadJob); ok {
			d.store = &Store{
				Store: NewStore(pool),
				FS:    fs,
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

	ctx := context.Background()

	if _, err := d.store.Exec(ctx, q.Build(), q.Args()...); err != nil {
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

	i, _, err := d.store.Get(ctx, query.Where("id", "=", query.Arg(d.ImageID)))

	if err != nil {
		d.log.Error.Println(errors.Err(err))
		return errors.Err(err)
	}

	if r != nil {
		store, err := i.filestore(d.store.FS)

		if err != nil {
			d.log.Error.Println(errors.Err(err))
			return errors.Err(err)
		}

		f, err := fs.ReadFile(i.Hash, r)

		if err != nil {
			return errors.Err(err)
		}

		defer fs.Cleanup(f)

		if _, err := store.Put(f); err != nil {
			downloaderr.String = errors.Cause(err).Error()
			downloaderr.Valid = true
		}
	}

	q = query.Update(
		downloadTable,
		query.Set("error", query.Arg(downloaderr)),
		query.Set("finished_at", query.Arg(time.Now())),
		query.Where("id", "=", query.Arg(d.ID)),
	)

	if _, err := d.store.Exec(ctx, q.Build(), q.Args()...); err != nil {
		d.log.Error.Println(errors.Err(err))
		return errors.Err(err)
	}

	d.log.Debug.Println("dispatching 'downloaded' event")

	d.queue.Produce(ctx, &Event{
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
	return nil
}

func (u *DownloadURL) UnmarshalJSON(b []byte) error {
	var s string

	if err := json.Unmarshal(b, &s); err != nil {
		return errors.Err(err)
	}

	if err := u.UnmarshalText([]byte(s)); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (u DownloadURL) Value() (driver.Value, error) { return strings.ToLower(u.Redacted()), nil }

type Download struct {
	ID         int64
	ImageID    int64
	Source     DownloadURL
	Error      database.Null[string]
	CreatedAt  time.Time
	StartedAt  database.Null[time.Time]
	FinishedAt database.Null[time.Time]
}

var _ database.Model = (*Download)(nil)

func (d *Download) Primary() (string, any) { return "id", d.ID }

func (d *Download) Scan(r *database.Row) error {
	valtab := map[string]any{
		"id":          &d.ID,
		"image_id":    &d.ImageID,
		"source":      &d.Source,
		"error":       &d.Error,
		"created_at":  &d.CreatedAt,
		"started_at":  &d.StartedAt,
		"finished_at": &d.FinishedAt,
	}

	if err := database.Scan(r, valtab); err != nil {
		return errors.Err(err)
	}
	return nil
}

func (d *Download) Params() database.Params {
	return database.Params{
		"id":          database.ImmutableParam(d.ID),
		"image_id":    database.CreateOnlyParam(d.ImageID),
		"source":      database.CreateOnlyParam(d.Source),
		"error":       database.UpdateOnlyParam(d.Error),
		"created_at":  database.CreateOnlyParam(d.CreatedAt),
		"started_at":  database.UpdateOnlyParam(d.StartedAt),
		"finished_at": database.UpdateOnlyParam(d.FinishedAt),
	}
}

func (*Download) Bind(database.Model) {}

func (d *Download) MarshalJSON() ([]byte, error) {
	if d == nil {
		return []byte("null"), nil
	}

	b, err := json.Marshal(map[string]any{
		"source":      d.Source.String(),
		"error":       d.Error,
		"created_at":  d.CreatedAt,
		"started_at":  d.StartedAt,
		"finished_at": d.FinishedAt,
	})

	if err != nil {
		return nil, errors.Err(err)
	}
	return b, nil
}

func (*Download) Endpoint(...string) string { return "" }

const downloadTable = "image_downloads"

func NewDownloadStore(pool *database.Pool) *database.Store[*Download] {
	return database.NewStore[*Download](pool, downloadTable, func() *Download {
		return &Download{}
	})
}
