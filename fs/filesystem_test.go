package fs

import (
	"crypto/rand"
	"os"
	"strconv"
	"testing"

	"djinn-ci.com/errors"
)

func randBytes(t *testing.T) []byte {
	b := make([]byte, 16)

	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return b
}

func Test_Filesystem(t *testing.T) {
	tmpdir := os.TempDir()

	tests := []struct {
		dir   string
		limit int64
	}{
		{tmpdir, 5},
		{tmpdir, 32},
		{tmpdir, 0},
	}

	for i, test := range tests {
		fs := NewFilesystemWithLimit(test.dir, test.limit)

		func(i int) {
			name := strconv.FormatInt(int64(i+1), 10) + ".record"

			r, err := fs.Create(name)

			if err != nil {
				t.Errorf("tests[%d] - unexpected Create error: %s\n", i, errors.Cause(err))
				return
			}

			_, err = fs.Create(name)

			if err == nil {
				t.Errorf("tests[%d] - expected subsequent Create to error, it did not\n", i)
				return
			}

			if err != ErrRecordExists {
				t.Errorf("tests[%d] - expected ErrRecordExists, got=%T\n", i, errors.Cause(err))
				return
			}

			defer func() {
				if err := fs.Remove(name); err != nil {
					t.Errorf("tests[%d] - unexpected Remove error: %s\n", i, errors.Cause(err))
				}
			}()

			b := randBytes(t)

			n, err := r.Write(b)

			if len(b) > n {
				cause := errors.Cause(err)

				if cause != ErrWriteLimit {
					t.Errorf("tests[%d] - expected ErrWriteLimit, got=%T %s\n", i, cause, cause)
					return
				}

				if int64(n) != test.limit {
					t.Errorf("tests[%d] - expected %d bytes written, got=%d\n", i, test.limit, n)
				}
			} else {
				if err != nil {
					t.Errorf("tests[%d] - unexpected Write error: %s\n", i, errors.Cause(err))
					return
				}

				if n != len(b) {
					t.Errorf("tests[%d] - expected %d bytes written, got=%d\n", i, len(b), n)
				}
			}

			if err := r.Close(); err != nil {
				t.Errorf("tests[%d] - unexpected Close error: %s\n", i, err)
			}

			if _, err := r.Write([]byte{}); err != nil {
				if err != ErrRecordClosed {
					t.Errorf("tests[%d] - expected ErrRecordClosedd from Write, got=%T\n", i, errors.Cause(err))
				}
			}

			if _, err := r.Read([]byte{}); err != nil {
				if err != ErrRecordClosed {
					t.Errorf("tests[%d] - expected ErrRecordClosedd from Read, got=%T\n", i, errors.Cause(err))
				}
			}

			if _, err := r.Seek(0, 0); err != nil {
				if err != ErrRecordClosed {
					t.Errorf("tests[%d] - expected ErrRecordClosedd from Seek, got=%T\n", i, errors.Cause(err))
				}
			}
		}(i)
	}
}
