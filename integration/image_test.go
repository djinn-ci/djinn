// +build integration

package integration

import (
	"net/http"
	"testing"
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
}
