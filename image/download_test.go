package image

import (
	"bytes"
	"encoding/gob"
	"net/url"
	"testing"
)

func Test_DownloadURL(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"http://example.com", "http://example.com"},
		{"https://admin:secret@example.com", "https://admin:xxxxx@example.com"},
		{"sftp://admin:secret@sftp.example.com", "sftp://admin:xxxxx@sftp.example.com"},
	}

	var buf bytes.Buffer

	for i, test := range tests {
		url, err := url.Parse(test.url)

		if err != nil {
			t.Fatalf("tests[%d] - %s\n", i, err)
		}

		downloadUrl := DownloadURL{
			URL: url,
		}

		val, _ := downloadUrl.Value()

		s, ok := val.(string)

		if !ok {
			t.Fatalf("tests[%d] - could not type assert DownloadURL.Value result to string\n", i)
		}

		if s != test.expected {
			t.Fatalf("tests[%d] - urls did not match, expected=%q, got=%q\n", i, test.expected, s)
		}

		if err := gob.NewEncoder(&buf).Encode(&downloadUrl); err != nil {
			t.Fatalf("tests[%d] - %s\n", i, err)
		}

		downloadUrl2 := DownloadURL{}

		if err := gob.NewDecoder(&buf).Decode(&downloadUrl2); err != nil {
			t.Fatalf("tests[%d] - %s\n", i, err)
		}

		if downloadUrl2.String() != test.url {
			t.Fatalf("tests[%d] - urls did not match after gob decode, expected=%q, got=%q\n", i, test.expected, s)
		}
	}
}
