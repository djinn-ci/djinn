// +build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"djinn-ci.com/namespace"
)

func Test_NamespaceCRUD(t *testing.T) {
	client := newClient(server)

	parentBody := map[string]interface{}{
		"name":       "fremen",
		"visibility": "private",
	}

	childBody := map[string]interface{}{
		"parent":     "fremen",
		"name":       "chani",
		"visibility": "public",
	}

	grandchildBody := map[string]interface{}{
		"parent": "fremen/chani",
		"name":   "letoii",
	}

	client.do(t, request{
		name:        "attempt to create grandchild namespace fremen/chani/letoii",
		method:      "POST",
		uri:         "/api/namespaces",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(grandchildBody),
		code:        http.StatusBadRequest,
	})

	parentResp := client.do(t, request{
		name:        "create parent namespace fremen",
		method:      "POST",
		uri:         "/api/namespaces",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(parentBody),
		code:        http.StatusCreated,
	})
	defer parentResp.Body.Close()

	parent := struct {
		URL string
	}{}

	if err := json.NewDecoder(parentResp.Body).Decode(&parent); err != nil {
		t.Fatalf("unexpected json.Decode error: %s\n", err)
	}

	parenturl, err := url.Parse(parent.URL)

	if err != nil {
		t.Fatalf("unexpected url.Parse error: %s\n", err)
	}

	childResp := client.do(t, request{
		name:        "create child namespace fremen/chani",
		method:      "POST",
		uri:         "/api/namespaces",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(childBody),
		code:        http.StatusCreated,
	})
	defer childResp.Body.Close()

	child := struct {
		URL string
	}{}

	if err := json.NewDecoder(childResp.Body).Decode(&child); err != nil {
		t.Fatalf("unexpected json.Decode error: %s\n", err)
	}

	createGrandchildResp := client.do(t, request{
		name:        "create grandchild namespace fremen/chani/letoii",
		method:      "POST",
		uri:         "/api/namespaces",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(grandchildBody),
		code:        http.StatusCreated,
	})
	defer createGrandchildResp.Body.Close()

	grandchild := struct {
		Visibility namespace.Visibility
		URL        string
		Parent     struct {
			Name string
		}
	}{}

	if err := json.NewDecoder(createGrandchildResp.Body).Decode(&grandchild); err != nil {
		t.Fatalf("unexpected json.Decode error: %s\n", err)
	}

	if grandchild.Visibility != namespace.Private {
		t.Fatalf("unexpected namespace visibility, expected=%q, got=%q\n", namespace.Private, grandchild.Visibility)
	}

	if grandchild.Parent.Name != "chani" {
		t.Fatalf("unexpected namespace name, expected=%q, got=%q\n", "fremen", grandchild.Parent.Name)
	}

	grandchildurl, err := url.Parse(grandchild.URL)

	if err != nil {
		t.Fatalf("unexpected url.Parse error: %s\n", err)
	}

	updatedGrandchild := map[string]interface{}{
		"visibility": "internal",
	}

	updateGrandchildResp := client.do(t, request{
		name:        "attempt to set grandchild namespace visibility to 'internal'",
		method:      "PATCH",
		uri:         grandchildurl.Path,
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(updatedGrandchild),
		code:        http.StatusOK,
	})
	defer updateGrandchildResp.Body.Close()

	grandchildUpdated := struct {
		Visibility namespace.Visibility
	}{}

	if err := json.NewDecoder(updateGrandchildResp.Body).Decode(&grandchildUpdated); err != nil {
		t.Fatalf("unexpected json.Decode error: %s\n", err)
	}

	updatedParent := map[string]interface{}{
		"visibility": "internal",
	}

	updateParentResp := client.do(t, request{
		name:        "attempt to set parent namespace visibility to 'internal'",
		method:      "PATCH",
		uri:         parenturl.Path,
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(updatedParent),
		code:        http.StatusOK,
	})
	defer updateParentResp.Body.Close()

	if err := json.NewDecoder(updateParentResp.Body).Decode(&grandchild); err != nil {
		t.Fatalf("unexpected json.Decode error: %s\n", err)
	}

	grandchildResp := client.do(t, request{
		name:        "get grandchild namespace to recheck visibility",
		method:      "GET",
		uri:         grandchildurl.Path,
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusOK,
	})
	defer grandchildResp.Body.Close()

	if err := json.NewDecoder(grandchildResp.Body).Decode(&grandchild); err != nil {
		t.Fatalf("unexpected json.Decode error: %s\n", err)
	}

	if grandchild.Visibility != namespace.Internal {
		t.Fatalf("unexpected namespace visibility, expected=%q, got=%q\n", namespace.Internal, grandchild.Visibility)
	}

	childurl, err := url.Parse(child.URL)

	if err != nil {
		t.Fatalf("unexpected url.Parse error: %s\n", err)
	}

	client.do(t, request{
		name:        "deleted the fremen/chani namespace",
		method:      "DELETE",
		uri:         childurl.Path,
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusNoContent,
	})

	client.do(t, request{
		name:        "attempt to view the fremen/chani namespace",
		method:      "GET",
		uri:         childurl.Path,
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusNotFound,
	})

	client.do(t, request{
		name:        "attempt to view the fremen/chani namespace",
		method:      "GET",
		uri:         grandchildurl.Path,
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusNotFound,
	})
}
