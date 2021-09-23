package integration

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"testing"

	"djinn-ci.com/integration/djinn"
)

// blusqr encodes a PNG of a blue square into the given writer.
func blusqr(w io.Writer) {
	rect := image.NewRGBA(image.Rect(0, 0, 100, 100))
	blue := color.RGBA{
		R: 127,
		G: 213,
		B: 234,
		A: 1,
	}

	draw.Draw(rect, rect.Bounds(), &image.Uniform{C: blue}, image.ZP, draw.Src)

	png.Encode(w, rect)
}

func Test_ObjectValidation(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	tests := []struct {
		params djinn.ObjectParams
		errors []string
	}{
		{
			djinn.ObjectParams{},
			[]string{"name", "file"},
		},
		{
			djinn.ObjectParams{
				Name:   "Test_ObjectValidation",
				Object: bytes.NewBuffer(make([]byte, 5*(1<<20))),
			},
			[]string{"file"},
		},
	}

	for i, test := range tests {
		_, err := djinn.CreateObject(cli, test.params)

		if err == nil {
			t.Fatalf("tests[%d] - expected error, got nil\n", i)
		}

		djinnerr, ok := err.(*djinn.Error)

		if !ok {
			t.Fatalf("tests[%d] - unexpected error type, expected=%T, got=%T (%q)\n", i, djinn.Error{}, err, err)
		}

		if len(djinnerr.Params) != len(test.errors) {
			t.Fatalf("tests[%d] - unexpected error count, expected=%d, got=%d\nerrs = %v", i, len(test.errors), len(djinnerr.Params), djinnerr.Params)
		}

		for _, err := range test.errors {
			if _, ok := djinnerr.Params[err]; !ok {
				t.Fatalf("tests[%d] - could not find field %s in field errors\n", i, err)
			}
		}
	}
}

func Test_ObjectCreate(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	var data bytes.Buffer

	blusqr(&data)

	origmd5 := md5.New()
	tr := io.TeeReader(&data, origmd5)

	o, err := djinn.CreateObject(cli, djinn.ObjectParams{
		Name:   "Test_ObjectCreate",
		Object: tr,
	})

	if err != nil {
		t.Fatal(err)
	}

	rc, err := o.Data(cli)

	if err != nil {
		t.Fatal(err)
	}

	defer rc.Close()

	md5 := md5.New()

	if _, err := io.Copy(md5, rc); err != nil {
		t.Fatal(err)
	}

	if orig := origmd5.Sum(nil); !bytes.Equal(orig, md5.Sum(nil)) {
		t.Fatalf("downloaded object content does not match, expected=%q, got=%q", hex.EncodeToString(orig), hex.EncodeToString(md5.Sum(nil)))
	}
}

func Test_ObjectDelete(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	var data bytes.Buffer

	blusqr(&data)

	origmd5 := md5.New()
	tr := io.TeeReader(&data, origmd5)

	o, err := djinn.CreateObject(cli, djinn.ObjectParams{
		Name:   "Test_ObjectDelete",
		Object: tr,
	})

	if err != nil {
		t.Fatal(err)
	}

	if err := o.Delete(cli); err != nil {
		t.Fatal(err)
	}
}
