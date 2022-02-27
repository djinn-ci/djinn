package http

import (
	"bytes"
	"io"
	"regexp"

	"djinn-ci.com/errors"
	"djinn-ci.com/image"
	"djinn-ci.com/namespace"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

type Form struct {
	File        *webutil.File
	Namespace   namespace.Path
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

type Validator struct {
	UserID int64
	Images *image.Store
	File   *webutil.FileValidator
	Form   *Form
}

var rename = regexp.MustCompile("^[a-zA-Z0-9_\\-]+$")

func (v Validator) Validate(errs webutil.ValidationErrors) {
	if v.Form.Name == "" {
		errs.Add("name", webutil.ErrFieldRequired("Name"))
	}

	if !rename.Match([]byte(v.Form.Name)) {
		errs.Add("name", errors.New("Name can only contain letters, numbers, and dashes"))
	}

	opts := []query.Option{
		query.Where("user_id", "=", query.Arg(v.UserID)),
		query.Where("name", "=", query.Arg(v.Form.Name)),
	}

	if v.Form.Namespace.Valid {
		_, n, err := v.Form.Namespace.ResolveOrCreate(v.Images.Pool, v.UserID)

		if err != nil {
			if perr, ok := err.(*namespace.PathError); ok {
				errs.Add("namespace", perr)
				return
			}
			errs.Add("fatal", err)
			return
		}

		if err := n.IsCollaborator(v.Images.Pool, v.UserID); err != nil {
			if errors.Is(err, namespace.ErrPermission) {
				errs.Add("namespace", err)
				return
			}
			errs.Add("fatal", err)
			return
		}
		opts[0] = query.Where("namespace_id", "=", query.Arg(n.ID))
	}

	_, ok, err := v.Images.Get(opts...)

	if err != nil {
		errs.Add("fatal", err)
		return
	}

	if ok {
		errs.Add("name", webutil.ErrFieldExists("Name"))
	}

	if v.Form.DownloadURL.URL == nil {
		v.File.Validate(errs)

		// Nil file indicates no file sent in the request, this error would be
		// set in the previous Validate call, so return early.
		if !v.File.HasFile() {
			return
		}

		buf := make([]byte, 4)

		if _, err := v.File.Read(buf); err != nil {
			errs.Add("fatal", err)
			return
		}

		if !bytes.Equal(buf, image.QCOW2Sig) {
			errs.Add("file", errors.New("File is not a valid QCOW2 file format"))
		}

		v.File.Seek(0, io.SeekStart)
		return
	}

	if v.Form.DownloadURL.Scheme == "sftp" {
		if _, ok := v.Form.DownloadURL.User.Password(); !ok {
			errs.Add("download_url", errors.New("Missing password from SFTP URL"))
		}
	}
}
