package http

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/url"
	"regexp"

	"djinn-ci.com/auth"
	"djinn-ci.com/database"
	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/namespace"

	"github.com/andrewpillar/webutil/v2"
)

type Form struct {
	Pool *database.Pool `schema:"-"`
	User *auth.User     `schema:"-"`

	Namespace   namespace.Path
	File        multipart.File
	Name        string
	DownloadURL image.DownloadURL `json:"download_url" schema:"download_url"`
}

func (f Form) Fields() map[string]string {
	return map[string]string{
		"namespace":    f.Namespace.String(),
		"name":         f.Name,
		"download_url": f.DownloadURL.String(),
	}
}

func fileExists(_ context.Context, val any) error {
	if val == nil {
		return webutil.ErrFieldRequired
	}
	return nil
}

func fileIsQCOW2(_ context.Context, val any) error {
	if val == nil {
		return webutil.ErrFieldRequired
	}

	f := val.(multipart.File)

	buf := make([]byte, len(image.QCOW2Sig))

	if _, err := f.Read(buf); err != nil {
		return err
	}

	f.Seek(0, io.SeekStart)

	if !bytes.Equal(buf, image.QCOW2Sig) {
		return errors.New("not a valid QCOW2 file format")
	}
	return nil
}

func validScheme(_ context.Context, val any) error {
	url := val.(*url.URL)

	schemes := map[string]struct{}{
		"http":  {},
		"https": {},
		"sftp":  {},
	}

	if _, ok := schemes[url.Scheme]; !ok {
		return errors.New("invalid url scheme")
	}
	return nil
}

func sftpHasPassword(_ context.Context, val any) error {
	url := val.(*url.URL)

	if url.Scheme != "sftp" {
		return nil
	}

	if _, ok := url.User.Password(); !ok {
		return errors.New("missing password from SFTP URL")
	}
	return nil
}

var reName = regexp.MustCompile("^[a-zA-Z0-9_\\-]+$")

func (f Form) Validate(ctx context.Context) error {
	var v webutil.Validator

	v.WrapError(
		webutil.IgnoreError("name", auth.ErrPermission),
		webutil.MapError(auth.ErrPermission, errors.New("permission denied")),
		webutil.WrapFieldError,
	)

	v.Add("namespace", f.Namespace, namespace.CanAccess(f.Pool, f.User))

	v.Add("name", f.Name, webutil.FieldRequired)
	v.Add("name", f.Name, webutil.FieldMatches(reName))
	v.Add("name", f.Name, namespace.ResourceUnique[*image.Image](image.NewStore(f.Pool), f.User, "name", f.Namespace))

	if f.DownloadURL.URL == nil {
		v.Add("file", f.File, fileExists)
		v.Add("file", f.File, fileIsQCOW2)
	} else {
		v.Add("download_url", f.DownloadURL.URL, validScheme)
		v.Add("download_url", f.DownloadURL.URL, sftpHasPassword)
	}

	errs := v.Validate(ctx)

	return errs.Err()
}
