package form

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"
)

var (
	width  = 200
	height = 100
)

func createImage(t *testing.T) image.Image {
	img := image.NewRGBA(image.Rectangle{
		image.Point{0, 0},
		image.Point{width, height},
	})

	rand.Seed(time.Now().UnixNano())

	for i := 0; i < width; i++ {
		for j := 0; j < height; j++ {
			r := uint8(rand.Intn(255))
			g := uint8(rand.Intn(255))
			b := uint8(rand.Intn(255))

			img.Set(i, j, color.RGBA{r, g, b, 0xFF})
		}
	}
	return img
}

func Test_FileValidate(t *testing.T) {
	tests := []struct{
		limit       int64
		disallowed  []string
		shouldError bool
	}{
		{0, []string{}, false},
		{0, []string{"image/png"}, true},
		{4096, []string{"image/png"}, true},
	}

	for i, test := range tests {
		pr, pw := io.Pipe()

		mw := multipart.NewWriter(pw)

		go func() {
			defer mw.Close()

			w, err := mw.CreateFormFile("file", "rand.png")

			if err != nil {
				t.Fatal(err)
			}

			if err := png.Encode(w, createImage(t)); err != nil {
				t.Fatal(err)
			}
		}()

		r := httptest.NewRequest("POST", "/", pr)
		r.Header.Add("Content-Type", mw.FormDataContentType())

		w := httptest.NewRecorder()

		f := File{
			Writer:     w,
			Request:    r,
			Limit:      test.limit,
			Disallowed: test.disallowed,
		}

		if err := f.Validate(); err != nil {
			if test.shouldError {
				continue
			}
			t.Fatal(err)
		}

		if test.shouldError {
			t.Errorf("test[%d] - should have failed\n", i)
		}
	}
}
