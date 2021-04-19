// +build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"

	"djinn-ci.com/image"

	"github.com/andrewpillar/query"
)

func Test_Image(t *testing.T) {
	client := newClient(server)

	f1 := openFile(t, "image")
	defer f1.Close()

	f2 := openFile(t, "image")
	defer f2.Close()

	reqs := []request{
		{
			name:        "attempt to create nil image",
			method:      "POST",
			uri:         "/api/images",
			token:       myTok,
			contentType: "application/octet-stream",
			code:        http.StatusBadRequest,
		},
		{
			name:        "attempt to create nil image with name",
			method:      "POST",
			uri:         "/api/images?name=djinn-dev",
			token:       myTok,
			contentType: "application/octet-stream",
			code:        http.StatusBadRequest,
		},
		{
			name:        "create image in namespace imagespace",
			method:      "POST",
			uri:         "/api/images?name=djinn-dev&namespace=imagespace",
			token:       myTok,
			contentType: "application/octet-stream",
			body:        f1,
			code:        http.StatusCreated,
		},
		{
			name:        "create image in namespace imagespace with duplicate name",
			method:      "POST",
			uri:         "/api/images?name=djinn-dev&namespace=imagespace",
			token:       myTok,
			contentType: "application/octet-stream",
			code:        http.StatusBadRequest,
			check:       checkFormErrors("name", "Name already exists"),
		},
		{
			name:        "create image with no namespace",
			method:      "POST",
			uri:         "/api/images?name=djinn-dev",
			token:       myTok,
			contentType: "application/octet-stream",
			body:        f2,
			code:        http.StatusCreated,
		},
	}

	var i0 struct {
		ID  int64
		URL string
	}

	for i, req := range reqs {
		resp := client.do(t, req)

		if i == len(reqs)-1 {
			if err := json.NewDecoder(resp.Body).Decode(&i0); err != nil {
				t.Fatal(err)
			}
		}
	}

	i, err := image.NewStore(db).Get(query.Where("id", "=", query.Arg(i0.ID)))

	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(i.Driver.String(), i.Hash)

	if _, err := imageStore.Stat(path); err != nil {
		t.Fatalf("failed to stat image file %s: %s\n", path, err)
	}

	client.do(t, request{
		name:        "search for image 'djinn'",
		method:      "GET",
		uri:         "/api/images?search=djinn-dev",
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusOK,
		check:       checkResponseJSONLen(2),
	})

	client.do(t, request{
		name:        "search for image 'djinn' as 'you'",
		method:      "GET",
		uri:         "/api/images?search=djinn-dev",
		token:       yourTok,
		contentType: "application/json",
		code:        http.StatusOK,
		check:       checkResponseJSONLen(0),
	})

	url, _ := url.Parse(i0.URL)

	client.do(t, request{
		name:   "delete image 'djinn'",
		method: "DELETE",
		uri:    url.Path,
		token:  myTok,
		code:   http.StatusNoContent,
	})

	if _, err := imageStore.Stat(path); err == nil {
		t.Fatalf("expected imageStore.Stat(%q) to fail, it did not\n", path)
	}
}
