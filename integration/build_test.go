package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"djinn-ci.com/namespace"
)

func Test_ManifestNamespaceValidation(t *testing.T) {
	client := newClient(server)

	manifest := "namespace: %s\ndriver:\n  type: qemu\n  image: centos/7"

	reqs := []request{
		{
			name:        "invalid namespace name",
			method:      "POST",
			uri:         "/api/builds",
			token:       myTok,
			contentType: "application/json",
			body:        jsonBody(map[string]interface{}{"manifest": fmt.Sprintf(manifest, "black-mesa")}),
			code:        http.StatusBadRequest,
			check:       checkFormErrors("manifest", namespace.ErrName.Error()),
		},
		{
			name:        "incorrect namespace owner",
			method:      "POST",
			uri:         "/api/builds",
			token:       myTok,
			contentType: "application/json",
			body:        jsonBody(map[string]interface{}{"manifest": fmt.Sprintf(manifest, "city17@you")}),
			code:        http.StatusBadRequest,
			check:       checkFormErrors("namespace", "Could not find namespace"),
		},
		{
			name:        "valid namespace name - me",
			method:      "POST",
			uri:         "/api/builds",
			token:       myTok,
			contentType: "application/json",
			body:        jsonBody(map[string]interface{}{"manifest": fmt.Sprintf(manifest, "blackmesa")}),
			code:        http.StatusCreated,
		},
		{
			name:        "valid namespace name - you",
			method:      "POST",
			uri:         "/api/builds",
			token:       yourTok,
			contentType: "application/json",
			body:        jsonBody(map[string]interface{}{"manifest": fmt.Sprintf(manifest, "blackmesa")}),
			code:        http.StatusCreated,
		},
	}

	for _, req := range reqs {
		client.do(t, req)
	}
}

func Test_AnonymousBuild(t *testing.T) {
	client := newClient(server)

	body := map[string]interface{}{
		"manifest": string(readFile(t, "anonymous.yml")),
		"tags":     []string{"anon", "go"},
	}

	createResp := client.do(t, request{
		name:        "create anonymous build",
		method:      "POST",
		uri:         "/api/builds",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(body),
		code:        http.StatusCreated,
	})
	defer createResp.Body.Close()

	build := struct {
		URL          string
		ObjectsURL   string `json:"objects_url"`
		VariablesURL string `json:"variables_url"`
		JobsURL      string `json:"jobs_url"`
		ArtifactsURL string `json:"artifacts_url"`
		TagsURL      string `json:"tags_url"`
	}{}

	if err := json.NewDecoder(createResp.Body).Decode(&build); err != nil {
		t.Fatalf("unexpected json.Decode error: %s\n", err)
	}

	for _, rawurl := range []string{
		build.ObjectsURL,
		build.VariablesURL,
		build.JobsURL,
		build.ArtifactsURL,
		build.TagsURL,
	} {
		url, err := url.Parse(rawurl)

		if err != nil {
			t.Fatalf("unexpected url.Parse error: %s\n", err)
		}

		client.do(t, request{
			name:        "GET " + url.Path,
			method:      "GET",
			uri:         url.Path,
			token:       myTok,
			contentType: "application/json",
			code:        http.StatusOK,
		})
	}

	reqs := []request{
		{
			name:        "search builds for 'anon' tag",
			method:      "GET",
			uri:         "/api/builds?search=anon",
			token:       myTok,
			contentType: "application/json",
			code:        http.StatusOK,
			check:       checkResponseJSONLen(1),
		},
		{
			name:        "search builds for 'go' tag",
			method:      "GET",
			uri:         "/api/builds?search=go",
			token:       myTok,
			contentType: "application/json",
			code:        http.StatusOK,
			check:       checkResponseJSONLen(1),
		},
	}

	for _, req := range reqs {
		client.do(t, req)
	}
}

func Test_NamespaceBuild(t *testing.T) {
	client := newClient(server)

	buildBody := map[string]interface{}{
		"manifest": string(readFile(t, "build.yml")),
		"tags":     []string{"centos/7"},
	}

	client.do(t, request{
		name:        "search builds for 'centos/7' tag",
		method:      "GET",
		uri:         "/api/builds?search=centos/7",
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusOK,
		check:       checkResponseJSONLen(0),
	})

	createResp := client.do(t, request{
		name:        "create namespace build",
		method:      "POST",
		uri:         "/api/builds",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(buildBody),
		code:        http.StatusCreated,
	})
	defer createResp.Body.Close()

	build := struct {
		ObjectsURL   string `json:"objects_url"`
		VariablesURL string `json:"variables_url"`
		JobsURL      string `json:"jobs_url"`
		ArtifactsURL string `json:"artifacts_url"`
		TagsURL      string `json:"tags_url"`
	}{}

	if err := json.NewDecoder(createResp.Body).Decode(&build); err != nil {
		t.Fatalf("unexpected json.Decode error: %s\n", err)
	}

	for _, rawurl := range []string{
		build.ObjectsURL,
		build.VariablesURL,
		build.JobsURL,
		build.ArtifactsURL,
		build.TagsURL,
	} {
		url, err := url.Parse(rawurl)

		if err != nil {
			t.Fatalf("unexpected url.Parse error: %s\n", err)
		}

		client.do(t, request{
			name:        "GET " + url.Path,
			method:      "GET",
			uri:         url.Path,
			token:       myTok,
			contentType: "application/json",
			code:        http.StatusOK,
		})
	}

	client.do(t, request{
		name:        "search namespaces for 'djinn'",
		method:      "GET",
		uri:         "/api/namespaces?search=djinn",
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusOK,
		check:       checkResponseJSONLen(1),
	})

	client.do(t, request{
		name:        "search builds for 'centos/7' tag",
		method:      "GET",
		uri:         "/api/builds?search=centos/7",
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusOK,
		check:       checkResponseJSONLen(1),
	})
}
