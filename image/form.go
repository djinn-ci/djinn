package image

import (
	"bytes"
	"compress/bzip2"
	"io"
	"io/ioutil"
	"regexp"

	"djinn-ci.com/errors"
	"djinn-ci.com/namespace"

	"github.com/andrewpillar/query"
	"github.com/andrewpillar/webutil"
)

// Form is the type that represents input data for uploading a new driver image.
type Form struct {
	namespace.Resource
	*webutil.File

	Images      *Store      `schema:"-"`
	Name        string      `schema:"name" json:"name"`
	DownloadURL DownloadURL `schema:"download_url" json:"download_url"`
}

var (
	_ webutil.Form = (*Form)(nil)

	rename = regexp.MustCompile("^[a-zA-Z0-9_\\-]+$")

	qcow = []byte{0x51, 0x46, 0x49, 0xFB} // QCOW magic number
)

// Fields returns a map containing the namespace, and name fields from the
// original Form.
func (f *Form) Fields() map[string]string {
	return map[string]string{
		"namespace":    f.Namespace,
		"name":         f.Name,
		"download_url": f.DownloadURL.String(),
	}
}

var downloadschemes = map[string]struct{}{
	"http":  {},
	"https": {},
	"sftp":  {},
}

// Validate checks to see if there is a name for the image, and if that name
// is valid. This will also check the contents of the uploaded file to make
// sure it is a valid QCOW2 image file. It does this by checking the first
// four bytes of the file match the ASCII magic number for QCOW2.
func (f *Form) Validate() error {
	errs := webutil.NewErrors()

	if err := f.Resource.Resolve(f.Images); err != nil {
		return errors.Err(err)
	}

	if f.Name == "" {
		errs.Put("name", webutil.ErrFieldRequired("Name"))
	}

	if !rename.Match([]byte(f.Name)) {
		errs.Put("name", webutil.ErrField("Name", errors.New("can only contain letters, numbers, and dashes")))
	}

	opts := []query.Option{
		query.Where("name", "=", query.Arg(f.Name)),
	}

	if f.Images.Namespace.IsZero() {
		opts = append(opts, query.Where("namespace_id", "IS", query.Lit("NULL")))
	}

	i, err := f.Images.Get(opts...)

	if err != nil {
		return errors.Err(err)
	}

	if !i.IsZero() {
		errs.Put("name", webutil.ErrFieldExists("Name"))
	}

	if f.DownloadURL.URL == nil {
		if err := f.File.Validate(); err != nil {
			if ferrs, ok := err.(*webutil.Errors); ok {
				errs.Merge(ferrs)
				return errs.Err()
			}
			return errors.Err(err)
		}

		if err := f.extractIfBzip(); err != nil {
			return errors.Err(err)
		}

		buf := make([]byte, 4, 4)

		if _, err := f.File.Read(buf); err != nil {
			return errors.Err(err)
		}

		if !bytes.Equal(buf, qcow) {
			errs.Put("file", webutil.ErrField("File", errors.New("not a valid QCOW file format")))
		}

		f.File.Seek(0, io.SeekStart)
		return errs.Err()
	}

	if _, ok := downloadschemes[f.DownloadURL.Scheme]; !ok {
		errs.Put("download_url", errors.New("invalid url scheme, must be http, https, or sftp"))
	}

	if f.DownloadURL.Scheme == "sftp" {
		if _, ok := f.DownloadURL.User.Password(); !ok {
			errs.Put("download_url", errors.New("sftp url must use password for authentication"))
		}
	}
	return errs.Err()
}

// extractIfBzip will check to see if the current uploaded file has been bzip2
// compressed. If it has been, then it will be decompressed, and the underlying
// multipart file will be replaced with the newly decrompressed file.
func (f *Form) extractIfBzip() error {
	// bzip2 magic number
	bzh := []byte("BZh")

	buf := make([]byte, 3, 3)

	if _, err := f.File.Read(buf); err != nil {
		return errors.Err(err)
	}

	f.Seek(0, io.SeekStart)

	// Not bzip, do nothing.
	if !bytes.Equal(buf, bzh) {
		return nil
	}

	tmp, err := ioutil.TempFile("", "bzip-raw")

	if err != nil {
		return errors.Err(err)
	}

	bzip2 := bzip2.NewReader(f.File)

	if _, err := io.Copy(tmp, bzip2); err != nil {
		return errors.Err(err)
	}

	f.Close()

	if err := f.Remove(); err != nil {
		return errors.Err(err)
	}

	tmp.Seek(0, io.SeekStart)
	f.File.File = tmp
	return nil
}
