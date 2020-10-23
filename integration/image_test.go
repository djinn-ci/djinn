// +build integration

package integration

import (
	"net/http"
	"testing"
)

func Test_Image(t *testing.T) {
	client := newClient(server)

	f := openFile(t, "image")
	defer f.Close()

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
			name:        "create image",
			method:      "POST",
			uri:         "/api/images?name=djinn-dev",
			token:       myTok,
			contentType: "application/octet-stream",
			body:        f,
			code:        http.StatusCreated,
		},
	}

	for _, req := range reqs {
		client.do(t, req)
	}

	client.do(t, request{
		name:        "search for image 'djinn'",
		method:      "GET",
		uri:         "/api/images?search=djinn-dev",
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusOK,
		check:       checkResponseJSONLen(1),
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
}
