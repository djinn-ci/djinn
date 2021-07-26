package image

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"database/sql/driver"
	"net/http"
	"net/url"
	"strings"
	"time"

	"djinn-ci.com/errors"
	"djinn-ci.com/fs"
	"djinn-ci.com/database"
	"djinn-ci.com/queue"
	"djinn-ci.com/user"

	"github.com/andrewpillar/query"

	"github.com/jmoiron/sqlx"
)

type downloadJob struct {
	db    *sqlx.DB
	store fs.Store

	image *Image
	url   DownloadURL
	id    int64
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

	_ queue.Job = (*downloadJob)(nil)

	downloadTable   = "image_downloads"
	downloadSchemes = map[string]struct{}{
		"http":  {},
		"https": {},
		"sftp":  {},
	}

	qemuMimeType = "application/x-qemu-disk"

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

	if err := db.QueryRow(q.Build(), q.Args()).Scan(&id); err != nil {
		return nil, errors.Err(err)
	}
	return &downloadJob{
		image: i,
		url:   url,
		id:    id,
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
func DownloadJobInit(db *sqlx.DB, store fs.Store) queue.InitFunc {
	return func(j queue.Job) {
		if d, ok := j.(*downloadJob); ok {
			d.db = db
			d.store = store
		}
	}
}

// Name implements the queue.Job interface.
func (d *downloadJob) Name() string { return "download_job" }

// Perform implements the queue.Job interface. This will attempt to download
// the image from the remote URL. This will fail if the URL does not have a
// valid scheme such as, http, https, or sftp.
func (d *downloadJob) Perform() error {
	q := query.Update(
		downloadTable,
		query.Set("started_at", query.Arg(time.Now())),
		query.Where("id", "=", query.Arg(d.id)),
	)

	if _, err := d.db.Exec(q.Build(), q.Args()...); err != nil {
		return errors.Err(err)
	}

	var (
		tlscfg      *tls.Config
		downloaderr sql.NullString
	)

	switch d.url.Scheme {
	case "https":
		pool, err := x509.SystemCertPool()

		if err != nil {
			return errors.Err(err)
		}

		tlscfg = &tls.Config{
			ServerName: d.url.Host,
			RootCAs:    pool,
		}
		fallthrough
	case "http":
		cli := &http.Client{
			Timeout:   time.Minute*10,
			Transport: &http.Transport{
				TLSClientConfig: tlscfg,
			},
		}

		req := &http.Request{
			Method: "GET",
			URL:    d.url.URL,
		}

		resp, err := cli.Do(req)

		if err != nil {
			downloaderr.String = err.Error()
			downloaderr.Valid = true
			break
		}

		defer resp.Body.Close()

		if resp.Header.Get("Content-Type") != qemuMimeType {
			downloaderr.String = "unexpected Content-Type, expected "+qemuMimeType
			downloaderr.Valid = true
			break
		}


		break
	case "sftp":
		break
	default:
		return ErrInvalidScheme
	}

	q = query.Update(
		downloadTable,
		query.Set("error", query.Arg(downloaderr)),
		query.Set("finished_at", query.Arg(time.Now())),
	)

	_, err := d.db.Exec(q.Build(), q.Args()...)
	return errors.Err(err)
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
