// +build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func Test_CollaboratorCRUD(t *testing.T) {
	client := newClient(server)

	myNamespace := map[string]interface{}{
		"name":       "conclave",
		"visibility": "private",
	}

	invite := map[string]interface{}{
		"handle": "you",
	}

	yourVariable := map[string]interface{}{
		"namespace": "conclave@me",
		"key":       "TOKEN",
		"value":     "1a2b3c4d",
	}

	yourNamespace := map[string]interface{}{
		"parent": "conclave",
		"name":   "sistine",
	}

	client.do(t, request{
		name:        "create conclave@me namespace",
		method:      "POST",
		uri:         "/api/namespaces",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(myNamespace),
		code:        http.StatusCreated,
	})

	client.do(t, request{
		name:        "attempt to create variable in conclave@me namespace as 'you'",
		method:      "POST",
		uri:         "/api/namespaces",
		token:       yourTok,
		contentType: "application/json",
		body:        jsonBody(yourVariable),
		code:        http.StatusUnprocessableEntity,
	})

	client.do(t, request{
		name:        "invite 'me' to conclave@me namespace",
		method:      "POST",
		uri:         "/api/n/me/conclave/-/invites",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(map[string]interface{}{"handle": "me"}),
		code:        http.StatusBadRequest,
	})

	inviteResp := client.do(t, request{
		name:        "invite 'you' to conclave@me namespace",
		method:      "POST",
		uri:         "/api/n/me/conclave/-/invites",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(invite),
		code:        http.StatusCreated,
	})
	defer inviteResp.Body.Close()

	i := struct {
		URL string
	}{}

	if err := json.NewDecoder(inviteResp.Body).Decode(&i); err != nil {
		t.Fatalf("unexpected json.Decode error: %s\n", err)
	}

	url, err := url.Parse(i.URL)

	if err != nil {
		t.Fatalf("unexpected url.Parse error: %s\n", err)
	}

	client.do(t, request{
		name:        "accept invite to conclave@me namespace as 'me'",
		method:      "PATCH",
		uri:         url.Path,
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusNotFound,
	})

	client.do(t, request{
		name:        "accept invite to conclave@me namespace as 'you'",
		method:      "PATCH",
		uri:         url.Path,
		token:       yourTok,
		contentType: "application/json",
		code:        http.StatusOK,
	})

	variableResp1 := client.do(t, request{
		name:        "attempt to create variable in conclave@me namespace as 'you'",
		method:      "POST",
		uri:         "/api/variables",
		token:       yourTok,
		contentType: "application/json",
		body:        jsonBody(yourVariable),
		code:        http.StatusCreated,
	})
	defer variableResp1.Body.Close()

	v := struct {
		UserID int64  `json:"user_id"`
		URL    string
	}{}

	if err := json.NewDecoder(variableResp1.Body).Decode(&v); err != nil {
		t.Fatalf("unexpected json.Decode error: %s\n", err)
	}

	reqs := []request{
		{
			name:        "attempt to create namespace with parent 'conclave' as 'you'",
			method:      "POST",
			uri:         "/api/namespaces",
			token:       yourTok,
			contentType: "application/json",
			body:        jsonBody(yourNamespace),
			code:        http.StatusUnprocessableEntity,
		},
		{
			name:        "attempt to remove 'me' as collaborator from namespace",
			method:      "DELETE",
			uri:         "/api/n/me/conclave/-/collaborators/me",
			token:       myTok,
			contentType: "application/json",
			code:        http.StatusBadRequest,
		},
		{
			name:        "attempt to remove 'you' as collaborator from namespace as 'you'",
			method:      "DELETE",
			uri:         "/api/n/me/conclave/-/collaborators/you",
			token:       yourTok,
			contentType: "application/json",
			code:        http.StatusNotFound,
		},
		{
			name:        "remove 'you' as collaborator from namespace as 'me'",
			method:      "DELETE",
			uri:         "/api/n/me/conclave/-/collaborators/you",
			token:       myTok,
			contentType: "application/json",
			code:        http.StatusNoContent,
		},
		{
			name:        "attempt to view conclave@me namespace as 'you'",
			method:      "GET",
			uri:         "/api/n/me/conclave",
			token:       yourTok,
			contentType: "application/json",
			code:        http.StatusNotFound,
		},
		{
			name:        "attempt to view namespace variable as 'you'",
			method:      "GET",
			uri:         v.URL,
			token:       yourTok,
			contentType: "application/json",
			code:        http.StatusNotFound,
		},
	}

	for _, req := range reqs {
		client.do(t, req)
	}

	variableResp2 := client.do(t, request{
		name:        "check variable user_id",
		method:      "GET",
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusOK,
	})
	defer variableResp2.Body.Close()

	originalUser := v.UserID

	if err := json.NewDecoder(variableResp2.Body).Decode(&v); err != nil {
		t.Fatalf("unexpected json.Decode error: %s\n", err)
	}

	if originalUser == v.UserID {
		t.Fatalf("expected variable user_id to change after collaborator deletion")
	}
}
