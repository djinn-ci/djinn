package integration

import (
	"bytes"
	"encoding/gob"
	"io"
	//	"net"
	"net/http"
	"net/http/httptest"
	//	"net/url"
	"os"
	"testing"
	"time"

	"djinn-ci.com/database"
	"djinn-ci.com/image"
	"djinn-ci.com/integration/djinn"
	"djinn-ci.com/log"
	"djinn-ci.com/queue"

	"github.com/mcmathja/curlyq"

	"github.com/vmihailenco/msgpack/v4"
)

func Test_ImageValidation(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	tests := []struct {
		params djinn.ImageParams
		errors []string
	}{
		{
			djinn.ImageParams{},
			[]string{"file", "name"},
		},
		{
			djinn.ImageParams{
				Name: "!!!!!",
			},
			[]string{"file", "name"},
		},
		{
			djinn.ImageParams{
				Name:        "Test_ImageValidation",
				DownloadURL: "ssh://root:secret@example.com",
			},
			[]string{"download_url"},
		},
		{
			djinn.ImageParams{
				Name:  "Test_ImageValidation",
				Image: bytes.NewBuffer([]byte{0x0, 0x0, 0x0, 0x0, 0x0}),
			},
			[]string{"file"},
		},
	}

	for i, test := range tests {
		_, err := djinn.CreateImage(cli, test.params)

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

var qcow2number = []byte{0x51, 0x46, 0x49, 0xFB}

func Test_ImageCreate(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	data := bytes.NewBuffer(qcow2number)

	var expected bytes.Buffer

	tr := io.TeeReader(data, &expected)

	i, err := djinn.CreateImage(cli, djinn.ImageParams{
		Name:  "Test_ImageCreate",
		Image: tr,
	})

	if err != nil {
		t.Fatal(err)
	}

	rc, err := i.Data(cli)

	if err != nil {
		t.Fatal(err)
	}

	defer rc.Close()

	var buf bytes.Buffer

	if _, err := io.Copy(&buf, rc); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(expected.Bytes(), buf.Bytes()) {
		t.Fatalf("downloaded image content does not match, expected=%s, got=%s\n", expected.Bytes(), buf.Bytes())
	}
}

var jobQueue = "jobs:data"

func Test_ImageCreateDownload(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", image.MimeTypeQEMU)
		http.ServeContent(w, r, "Test_ImageCreateDownload", time.Now(), bytes.NewReader(qcow2number))
	}))
	defer srv.Close()

	t.Log("started test server on", srv.URL, "to serve image downloads")

	i, err := djinn.CreateImage(cli, djinn.ImageParams{
		Name:        "Test_ImageCreateDownload",
		DownloadURL: srv.URL,
	})

	if err != nil {
		t.Fatal(err)
	}

	m, err := redis.HGetAll(jobQueue).Result()

	if err != nil {
		t.Fatal(err)
	}

	var (
		qjob  curlyq.Job
		job   queue.Job
		found bool
	)

	gob.Register(&image.DownloadJob{})

	for _, v := range m {
		if err := msgpack.Unmarshal([]byte(v), &qjob); err != nil {
			t.Fatal(err)
		}

		if err := gob.NewDecoder(bytes.NewBuffer(qjob.Data)).Decode(&job); err != nil {
			t.Fatal(err)
		}

		downloadJob, ok := job.(*image.DownloadJob)

		if !ok {
			t.Fatalf("unexpected job type, expected=%T, got=%T\n", &image.DownloadJob{}, job)
		}

		if downloadJob.ImageID == i.ID {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("could not find download job for image %d in queue %s\n", i.ID, jobQueue)
	}

	store, err := imagestore.Partition(i.UserID)

	if err != nil {
		t.Fatal(err)
	}

	log := log.New(os.Stdout)
	log.SetLevel("DEBUG")

	db2, err := database.Connect("host=localhost port=5432 dbname=djinn user=djinn_consumer password=secret sslmode=disable")

	if err != nil {
		t.Fatal(err)
	}

	defer db2.Close()

	// Directly call the init function since we are not processing it on the
	// queue where this would have otherwise been invoked.
	image.DownloadJobInit(db2, log, store)(job)

	if err := job.Perform(); err != nil {
		t.Fatal(err)
	}

	if err := i.Get(cli); err != nil {
		t.Fatal(err)
	}

	if i.Download == nil {
		t.Fatalf("expected image to have a download")
	}

	if i.Download.Error.Valid {
		t.Fatalf("unexpected image download error %q\n", i.Download.Error.String)
	}
}

func Test_ImageDelete(t *testing.T) {
	cli, _ := djinn.NewClient(tokens.get("gordon.freeman").Token, apiEndpoint)

	data := bytes.NewBuffer([]byte{0x51, 0x46, 0x49, 0xFB})

	i, err := djinn.CreateImage(cli, djinn.ImageParams{
		Name:  "Test_ImageDelete",
		Image: data,
	})

	if err != nil {
		t.Fatal(err)
	}

	if err := i.Delete(cli); err != nil {
		t.Fatal(err)
	}
}
