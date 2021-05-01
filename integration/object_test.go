package integration

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"djinn-ci.com/object"

	"github.com/andrewpillar/query"
)

func Test_Object(t *testing.T) {
	client := newClient(server)

	f1 := openFile(t, "data")
	defer f1.Close()

	f2 := openFile(t, "data")
	defer f2.Close()

	reqs := []request{
		{
			name:        "attempt to create nil object",
			method:      "POST",
			contentType: "application/octet-stream",
			uri:         "/api/objects",
			token:       myTok,
			code:        http.StatusBadRequest,
		},
		{
			name:        "attempt to create nil object with name",
			method:      "POST",
			contentType: "application/octet-stream",
			uri:         "/api/objects?name=data",
			token:       myTok,
			code:        http.StatusBadRequest,
		},
		{
			name:        "create object in namespace objectspace",
			method:      "POST",
			contentType: "application/octet-stream",
			uri:         "/api/objects?name=data&namespace=objectspace",
			token:       myTok,
			body:        f1,
			code:        http.StatusCreated,
		},
		{
			name:        "create object in namespace objectspace with duplicate name",
			method:      "POST",
			contentType: "application/octet-stream",
			uri:         "/api/objects?name=data&namespace=objectspace",
			token:       myTok,
			code:        http.StatusBadRequest,
			check:       checkFormErrors("name", "Name already exists"),
		},
		{
			name:        "create object with no namespace",
			method:      "POST",
			contentType: "application/octet-stream",
			uri:         "/api/objects?name=data",
			token:       myTok,
			body:        f2,
			code:        http.StatusCreated,
		},
	}

	var o0 struct {
		ID  int64
		URL string
	}

	for i, req := range reqs {
		resp := client.do(t, req)

		if i == len(reqs)-1 {
			if err := json.NewDecoder(resp.Body).Decode(&o0); err != nil {
				t.Fatal(err)
			}
		}
	}

	o, err := object.NewStore(db).Get(query.Where("id", "=", query.Arg(o0.ID)))

	if err != nil {
		t.Fatal(err)
	}

	part, err := objectStore.Partition(o.UserID)

	if err != nil {
		t.Fatal(err)
	}

	if _, err := part.Stat(o.Hash); err != nil {
		t.Fatalf("failed to stat image file %s: %s\n", o.Hash, err)
	}

	client.do(t, request{
		name:        "search for object 'data'",
		method:      "GET",
		uri:         "/api/objects?search=data",
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusOK,
		check:       checkResponseJSONLen(2),
	})

	client.do(t, request{
		name:        "search for object 'data' as 'you'",
		method:      "GET",
		uri:         "/api/objects?search=data",
		token:       yourTok,
		contentType: "application/json",
		code:        http.StatusOK,
		check:       checkResponseJSONLen(0),
	})

	url, _ := url.Parse(o0.URL)

	client.do(t, request{
		name:   "delete object 'data'",
		method: "DELETE",
		uri:    url.Path,
		token:  myTok,
		code:   http.StatusNoContent,
	})

	if _, err := objectStore.Stat(o.Hash); err == nil {
		t.Fatalf("expected objectStore.Stat(%q) to fail, it did not\n", o.Hash)
	}
}
