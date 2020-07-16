// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/andrewpillar/thrall/namespace"
)

func Test_ManifestNamespaceValidation(t *testing.T) {
	flow := NewFlow()

	namespace := map[string]interface{}{
		"name":       "city17",
		"visibility": "private",
	}

	flow.Add(ApiPost(t, "/api/namespaces", yourTok, JSON(t, namespace)), 201, nil)

	tests := []struct{
		name         string
		expectedCode int
	}{
		{
			"black-mesa",
			400,
		},
		{
			"city17@you",
			422,
		},
		{
			"blackmesa",
			201,
		},
	}

	for _, test := range tests {
		build := map[string]interface{}{
			"manifest": "namespace: " + test.name + "\ndriver:\n  type: qemu\n  image: centos/7",
		}

		flow.Add(ApiPost(t, "/api/builds", myTok, JSON(t, build)), test.expectedCode, nil)
	}
	flow.Do(t, server.Client())
}

func Test_AnonymousBuildCreateFlow(t *testing.T) {
	flow := NewFlow()

	build := map[string]interface{}{
		"manifest": string(ReadFile(t, "anonymous.yml")),
		"tags":     []string{"anon", "golang"},
	}

	flow.Add(ApiPost(t, "/api/builds", myTok, JSON(t, build)), 201, func(t *testing.T, r *http.Request, b []byte) {
		resp := struct{
			ObjectsURL   string `json:"objects_url"`
			VariablesURL string `json:"variables_url"`
			JobsURL      string `json:"jobs_url"`
			ArtifactsURL string `json:"artifacts_url"`
			TagsURL      string `json:"tags_url"`
		}{}

		if err := json.Unmarshal(b, &resp); err != nil {
			t.Errorf("%q %q - unexpected Unmarshal error: %s\n", r.Method, r.URL, err)
		}

		u, _ := url.Parse(resp.ObjectsURL)
		objectsUrl := u.Path

		u, _ = url.Parse(resp.VariablesURL)
		variablesUrl := u.Path

		u, _ = url.Parse(resp.JobsURL)
		jobsUrl := u.Path

		u, _ = url.Parse(resp.ArtifactsURL)
		artifactsUrl := u.Path

		u, _ = url.Parse(resp.TagsURL)
		tagsUrl := u.Path

		subflow := NewFlow()

		subflow.Add(ApiGet(t, objectsUrl, myTok), 200, checkJSONResponseSize(0))
		subflow.Add(ApiGet(t, variablesUrl, myTok), 200, checkJSONResponseSize(1))
		subflow.Add(ApiGet(t, jobsUrl, myTok), 200, checkJSONResponseSize(2))
		subflow.Add(ApiGet(t, artifactsUrl, myTok), 200, checkJSONResponseSize(0))
		subflow.Add(ApiGet(t, tagsUrl, myTok), 200, checkJSONResponseSize(1))
		subflow.Do(t, server.Client())
	})

	flow.Add(ApiGet(t, "/api/n/me/djinn/-/builds?search=anon", myTok), 200, checkJSONResponseSize(0))
	flow.Add(ApiGet(t, "/api/builds?search=golang", myTok), 200, checkJSONResponseSize(1))
}

func Test_NamespaceBuildCreateFlow(t *testing.T) {
	flow := NewFlow()

	build := map[string]interface{}{
		"manifest": string(ReadFile(t, "build.yml")),
		"tags":     []string{"centos/7"},
	}

	variable := map[string]interface{}{
		"namespace": "djinn",
		"key":       "EDITOR",
		"value":     "ed",
	}

	f := OpenFile(t, "data")
	defer f.Close()

	flow.Add(ApiPost(t, "/api/objects?name=data&namespace=djinn", myTok, f), 201, nil)
	flow.Add(ApiPost(t, "/api/variables", myTok, JSON(t, variable)), 201, nil)

	flow.Add(ApiGet(t, "/api/n/me/djinn", myTok), 200, nil)
	flow.Add(ApiGet(t, "/api/n/me/djinn/-/objects", myTok), 200, checkJSONResponseSize(1))
	flow.Add(ApiGet(t, "/api/n/me/djinn/-/objects?search=da", myTok), 200, checkJSONResponseSize(1))
	flow.Add(ApiGet(t, "/api/n/me/djinn/-/variables", myTok), 200, checkJSONResponseSize(1))
	flow.Add(ApiGet(t, "/api/n/me/djinn/-/variables?search=EDI", myTok), 200, checkJSONResponseSize(1))

	flow.Add(ApiGet(t, "/api/builds?search=centos/7", myTok), 200, checkJSONResponseSize(0))

	flow.Add(ApiPost(t, "/api/builds", myTok, JSON(t, build)), 201, func(t *testing.T, r *http.Request, b []byte) {
		resp := struct{
			ObjectsURL   string `json:"objects_url"`
			VariablesURL string `json:"variables_url"`
			JobsURL      string `json:"jobs_url"`
			ArtifactsURL string `json:"artifacts_url"`
			TagsURL      string `json:"tags_url"`
		}{}

		if err := json.Unmarshal(b, &resp); err != nil {
			t.Errorf("%q %q - unexpected Unmarshal error: %s\n", r.Method, r.URL, err)
		}

		u, _ := url.Parse(resp.ObjectsURL)
		objectsUrl := u.Path

		u, _ = url.Parse(resp.VariablesURL)
		variablesUrl := u.Path

		u, _ = url.Parse(resp.JobsURL)
		jobsUrl := u.Path

		u, _ = url.Parse(resp.ArtifactsURL)
		artifactsUrl := u.Path

		u, _ = url.Parse(resp.TagsURL)
		tagsUrl := u.Path

		subflow := NewFlow()

		subflow.Add(ApiGet(t, objectsUrl, myTok), 200, checkJSONResponseSize(1))
		subflow.Add(ApiGet(t, variablesUrl, myTok), 200, checkJSONResponseSize(2))
		subflow.Add(ApiGet(t, jobsUrl, myTok), 200, checkJSONResponseSize(2))
		subflow.Add(ApiGet(t, artifactsUrl, myTok), 200, checkJSONResponseSize(1))
		subflow.Add(ApiPost(t, tagsUrl, myTok, bytes.NewBufferString(`["another-tag"]`)), 201, nil)
		subflow.Add(ApiGet(t, tagsUrl, myTok), 200, checkJSONResponseSize(2))
		subflow.Do(t, server.Client())
	})

	flow.Add(ApiGet(t, "/api/builds?search=centos/7", myTok), 200, checkJSONResponseSize(1))
	flow.Add(ApiGet(t, "/api/builds?status=queued", myTok), 200, checkJSONResponseSizeApprox(1))

	flow.Add(ApiGet(t, "/api/n/me/djinn/-/builds", myTok), 200, checkJSONResponseSize(1))

	flow.Do(t, server.Client())
}

func Test_NamespaceFlow(t *testing.T) {
	flow := NewFlow()

	parent := map[string]interface{}{
		"name":       "fremen",
		"visibility": "private",
	}

	child := map[string]interface{}{
		"parent":     "fremen",
		"name":       "chani",
		"visibility": "public",
	}

	grandchild := map[string]interface{}{
		"parent": "fremen/chani",
		"name":   "letoii",
	}

	flow.Add(ApiPost(t, "/api/namespaces", myTok, JSON(t, map[string]interface{}{})), 400, nil)
	flow.Add(ApiPost(t, "/api/namespaces", myTok, JSON(t, parent)), 201, nil)
	flow.Add(ApiPost(t, "/api/namespaces", myTok, JSON(t, child)), 201, func(t *testing.T, r *http.Request, b []byte) {
		n := struct{
			Visibility namespace.Visibility
			Parent     struct{
				Name string
			}
		}{}

		if err := json.Unmarshal(b, &n); err != nil {
			t.Fatalf("%q %q unexpected Unmarshal error: %s\n", r.Method, r.URL, err)
		}

		if n.Visibility != namespace.Private {
			t.Errorf("exepected namespace visibility %d. got=%d\n", namespace.Private, n.Visibility)
		}

		if n.Parent.Name != "fremen" {
			t.Errorf("expected namespace parent name %q. got=%q\n", "fremen", n.Parent.Name)
		}
	})
	flow.Add(ApiPost(t, "/api/namespaces", myTok, JSON(t, grandchild)), 201, nil)
	flow.Add(ApiDelete(t, "/api/n/me/fremen/chani", myTok), 204, nil)
	flow.Add(ApiGet(t, "/api/n/me/fremen/chani/leto", myTok), 404, nil)
	flow.Add(ApiGet(t, "/api/n/me/fremen/chani", myTok), 404, nil)

	flow.Do(t, server.Client())
}

func Test_CollaboratorFlow(t *testing.T) {
	f1 := NewFlow()

	myNamespace := map[string]interface{}{
		"name":       "conclave",
		"visibility": "private",
	}

	invite := map[string]interface{}{
		"handle": "you",
	}

	variable := map[string]interface{}{
		"namespace": "conclave@me",
		"key":       "TOKEN",
		"value":     "1a2b3c4d",
	}

	f1.Add(ApiPost(t, "/api/namespaces", myTok, JSON(t, myNamespace)), 201, nil)
	f1.Add(ApiGet(t, "/api/n/me/conclave", yourTok), 404, nil)
	f1.Add(ApiPost(t, "/api/n/me/conclave/-/invites", myTok, JSON(t, map[string]interface{}{"handle": "me"})), 400, nil)
	f1.Add(ApiPost(t, "/api/n/me/conclave/-/invites", myTok, JSON(t, invite)), 201, func(t *testing.T, r *http.Request, b []byte) {
		i := struct{
			URL string
		}{}

		if err := json.Unmarshal(b, &i); err != nil {
			t.Fatalf("%q %q unexpected Unmarshal error: %s\n", r.Method, r.URL, err)
		}

		url, _ := url.Parse(i.URL)

		f2 := NewFlow()

		f2.Add(ApiGet(t, "/api/invites", yourTok), 200, checkJSONResponseSize(1))
		f2.Add(ApiPatch(t, url.Path, myTok, nil), 404, nil)
		f2.Add(ApiPatch(t, url.Path, yourTok, nil), 200 , nil)
		f2.Add(ApiGet(t, "/api/n/me/conclave", yourTok), 200, nil)
		f2.Add(ApiPost(t, "/api/variables", yourTok, JSON(t, variable)), 201, func(t *testing.T, r *http.Request, b []byte) {
			v := struct{
				URL string
			}{}

			if err := json.Unmarshal(b, &v); err != nil {
				t.Fatalf("%q %q unexpected Unmarshal error: %s\n", r.Method, r.URL, err)
			}

			url, _ := url.Parse(v.URL)

			f3 := NewFlow()

			f3.Add(ApiGet(t, "/api/n/me/conclave/-/variables?search=TOKEN", yourTok), 200, checkJSONResponseSize(1))
			f3.Add(ApiDelete(t, url.Path, myTok), 204, nil)

			f3.Do(t, server.Client())
		})

		f2.Do(t, server.Client())
	})
	f1.Add(ApiDelete(t, "/api/n/me/conclave/-/collaborators/you", yourTok), 404, nil)
	f1.Add(ApiDelete(t, "/api/n/me/conclave/-/collaborators/you", myTok), 204, nil)
	f1.Add(ApiGet(t, "/api/n/me/conclave", yourTok), 404, nil)
	f1.Add(ApiPost(t, "/api/variables", yourTok, JSON(t, variable)), 422, nil)

	f1.Do(t, server.Client())
}

func Test_ImageFlow(t *testing.T) {
	f1 := NewFlow()

	f := OpenFile(t, "image")
	defer f.Close()

	f1.Add(ApiPost(t, "/api/images", myTok, nil), 400, nil)
	f1.Add(ApiPost(t, "/api/images?name=go-dev", myTok, nil), 400, nil)
	f1.Add(ApiPost(t, "/api/images?name=go-dev", myTok, f), 201, nil)
	f1.Add(ApiGet(t, "/api/images?search=go", myTok), 200, checkJSONResponseSizeApprox(1))
	f1.Add(ApiGet(t, "/api/images?search=go", yourTok), 200, checkJSONResponseSize(0))

	f1.Do(t, server.Client())
}

func Test_VariableFlow(t *testing.T) {
	f1 := NewFlow()

	variable := map[string]interface{}{

	}

	f1.Add(ApiPost(t, "/api/variables", myTok, JSON(t, variable)), 201, nil)

	f1.Do(t, server.Client())
}
