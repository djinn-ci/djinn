// +build integration

package integration

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/url"
	"testing"

	"github.com/andrewpillar/djinn/namespace"
)

func Test_ManifestNamespaceValidation(t *testing.T) {
	flow := NewFlow()

	namespace := map[string]interface{}{
		"name":       "city17",
		"visibility": "private",
	}

	flow.Add(ApiPost(t, "/api/namespaces", yourTok, JSON(t, namespace)), 201, nil)

	tests := []struct {
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
		resp := struct {
			URL          string
			ObjectsURL   string `json:"objects_url"`
			VariablesURL string `json:"variables_url"`
			JobsURL      string `json:"jobs_url"`
			ArtifactsURL string `json:"artifacts_url"`
			TagsURL      string `json:"tags_url"`
		}{}

		if err := json.Unmarshal(b, &resp); err != nil {
			t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
		}

		u, _ := url.Parse(resp.URL)
		buildUrl := u.Path

		u, _ = url.Parse(resp.ObjectsURL)
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

		subflow.Add(ApiGet(t, buildUrl, myTok), 200, nil)
		subflow.Add(ApiGet(t, objectsUrl, myTok), 200, checkJSONResponseSize(0))
		subflow.Add(ApiGet(t, variablesUrl, myTok), 200, checkJSONResponseSize(1))
		subflow.Add(ApiGet(t, jobsUrl, myTok), 200, checkJSONResponseSize(2))
		subflow.Add(ApiGet(t, artifactsUrl, myTok), 200, checkJSONResponseSize(0))
		subflow.Add(ApiGet(t, tagsUrl, myTok), 200, checkJSONResponseSize(2))
		subflow.Do(t, server.Client())
	})

	flow.Add(ApiGet(t, "/api/builds?search=anon", myTok), 200, checkJSONResponseSize(1))
	flow.Add(ApiGet(t, "/api/builds?search=golang", myTok), 200, checkJSONResponseSize(1))
	flow.Do(t, server.Client())
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
		resp := struct {
			ObjectsURL   string `json:"objects_url"`
			VariablesURL string `json:"variables_url"`
			JobsURL      string `json:"jobs_url"`
			ArtifactsURL string `json:"artifacts_url"`
			TagsURL      string `json:"tags_url"`
		}{}

		if err := json.Unmarshal(b, &resp); err != nil {
			t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
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
	f1 := NewFlow()

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

	parentUrl := ""

	f1.Add(ApiPost(t, "/api/namespaces", myTok, JSON(t, map[string]interface{}{})), 400, nil)
	f1.Add(ApiPost(t, "/api/namespaces", myTok, JSON(t, map[string]interface{}{"visibility": "foo"})), 400, nil)
	f1.Add(ApiPost(t, "/api/namespaces", myTok, JSON(t, parent)), 201, func(t *testing.T, r *http.Request, b []byte) {
		n := struct {
			URL string
		}{}

		if err := json.Unmarshal(b, &n); err != nil {
			t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
		}

		url, _ := url.Parse(n.URL)
		parentUrl = url.Path
	})
	f1.Add(ApiPost(t, "/api/namespaces", myTok, JSON(t, child)), 201, func(t *testing.T, r *http.Request, b []byte) {
		n := struct {
			Visibility namespace.Visibility
			URL        string
			Parent     struct {
				Name string
			}
		}{}

		if err := json.Unmarshal(b, &n); err != nil {
			t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
		}

		if n.Visibility != namespace.Private {
			t.Errorf("exepected namespace visibility %q. got=%q\n", namespace.Private.String(), n.Visibility.String())
		}

		if n.Parent.Name != "fremen" {
			t.Errorf("expected namespace parent name %q. got=%q\n", "fremen", n.Parent.Name)
		}
	})
	f1.Add(ApiPost(t, "/api/namespaces", myTok, JSON(t, grandchild)), 201, nil)

	f1.Do(t, server.Client())

	updatedParent := map[string]interface{}{
		"visibility": "internal",
	}

	f1.Add(ApiPatch(t, parentUrl, myTok, JSON(t, updatedParent)), 200, func(t *testing.T, r *http.Request, b []byte) {
		n := struct {
			NamespacesURL string `json:"namespaces_url"`
		}{}

		if err := json.Unmarshal(b, &n); err != nil {
			t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
		}

		f2 := NewFlow()

		url, _ := url.Parse(n.NamespacesURL)

		f2.Add(ApiGet(t, url.Path, myTok), 200, func(t *testing.T, r *http.Request, b []byte) {
			nn := []struct {
				URL string
			}{}

			if err := json.Unmarshal(b, &nn); err != nil {
				t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
			}

			f3 := NewFlow()

			checkVisibility := func(t *testing.T, r *http.Request, b []byte) {
				n := struct {
					Visibility namespace.Visibility
				}{}

				if err := json.Unmarshal(b, &n); err != nil {
					t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
				}

				if n.Visibility != namespace.Internal {
					t.Errorf("expected namespace visibility %q. got=%q\n", namespace.Internal.String(), n.Visibility.String())
				}
			}

			for _, n := range nn {
				url, _ := url.Parse(n.URL)

				f3.Add(ApiGet(t, url.Path, myTok), 200, checkVisibility)
			}

			f3.Do(t, server.Client())
		})

		f2.Do(t, server.Client())
	})

	f1.Add(ApiDelete(t, "/api/n/me/fremen/chani", myTok), 204, nil)
	f1.Add(ApiGet(t, "/api/n/me/fremen/chani/leto", myTok), 404, nil)
	f1.Add(ApiGet(t, "/api/n/me/fremen/chani", myTok), 404, nil)
	f1.Do(t, server.Client())
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

	yourNamespace := map[string]interface{}{
		"parent": "conclave",
		"name":   "sistine",
	}

	f1.Add(ApiPost(t, "/api/namespaces", myTok, JSON(t, myNamespace)), 201, nil)
	f1.Add(ApiGet(t, "/api/n/me/conclave", yourTok), 404, nil)
	f1.Add(ApiPost(t, "/api/n/me/conclave/-/invites", myTok, JSON(t, map[string]interface{}{"handle": "me"})), 400, nil)
	f1.Add(ApiPost(t, "/api/n/me/conclave/-/invites", myTok, JSON(t, invite)), 201, func(t *testing.T, r *http.Request, b []byte) {
		i := struct {
			URL string
		}{}

		if err := json.Unmarshal(b, &i); err != nil {
			t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
		}

		url, _ := url.Parse(i.URL)

		f2 := NewFlow()

		f2.Add(ApiGet(t, "/api/invites", yourTok), 200, checkJSONResponseSize(1))
		f2.Add(ApiPatch(t, url.Path, myTok, nil), 404, nil)
		f2.Add(ApiPatch(t, url.Path, yourTok, nil), 200, nil)
		f2.Add(ApiGet(t, "/api/n/me/conclave", yourTok), 200, nil)
		f2.Add(ApiPost(t, "/api/variables", yourTok, JSON(t, variable)), 201, func(t *testing.T, r *http.Request, b []byte) {
			v := struct {
				URL string
			}{}

			if err := json.Unmarshal(b, &v); err != nil {
				t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
			}

			url, _ := url.Parse(v.URL)

			f3 := NewFlow()

			f3.Add(ApiGet(t, "/api/n/me/conclave/-/variables?search=TOKEN", yourTok), 200, checkJSONResponseSize(1))
			f3.Add(ApiDelete(t, url.Path, myTok), 204, nil)

			f3.Do(t, server.Client())
		})
		f2.Add(ApiPost(t, "/api/namespaces", yourTok, JSON(t, yourNamespace)), 422, nil)

		f2.Do(t, server.Client())
	})
	f1.Add(ApiDelete(t, "/api/n/me/conclave/-/collaborators/you", yourTok), 404, nil)
	f1.Add(ApiDelete(t, "/api/n/me/conclave/-/collaborators/you", myTok), 204, nil)
	f1.Add(ApiGet(t, "/api/n/me/conclave", yourTok), 404, nil)
	f1.Add(ApiPost(t, "/api/namespaces", yourTok, JSON(t, yourNamespace)), 422, nil)
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
		"key":   "PGADDR",
		"value": "host=localhost port=5432 dbname=djinn user=djinn password=secret sslmode=disable",
	}

	f1.Add(ApiPost(t, "/api/variables", myTok, JSON(t, map[string]interface{}{})), 400, nil)
	f1.Add(ApiPost(t, "/api/variables", myTok, JSON(t, variable)), 201, nil)
	f1.Add(ApiPost(t, "/api/variables", myTok, JSON(t, variable)), 400, nil)

	variable["namespace"] = "database"

	f1.Add(ApiPost(t, "/api/variables", myTok, JSON(t, variable)), 201, nil)
	f1.Add(ApiGet(t, "/api/variables?search=PG", myTok), 200, checkJSONResponseSizeApprox(2))

	f1.Do(t, server.Client())
}

func Test_KeyFlow(t *testing.T) {
	rsakey, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		t.Fatal(err)
	}

	b := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(rsakey),
	})

	key := map[string]interface{}{
		"name": "id_rsa",
		"key":  "AAAAABBBBBCCCCC",
	}

	f1 := NewFlow()

	f1.Add(ApiPost(t, "/api/keys", myTok, JSON(t, map[string]interface{}{})), 400, nil)
	f1.Add(ApiPost(t, "/api/keys", myTok, JSON(t, key)), 400, nil)

	key["key"] = string(b)

	f1.Add(ApiPost(t, "/api/keys", myTok, JSON(t, key)), 201, func(t *testing.T, r *http.Request, b []byte) {
		k := struct {
			URL string
		}{}

		if err := json.Unmarshal(b, &k); err != nil {
			t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
		}

		key := map[string]interface{}{
			"key":    "AAAABBBBCCCC",
			"config": "UserKnownHostsFile /dev/null",
		}

		url, _ := url.Parse(k.URL)

		f2 := NewFlow()

		f2.Add(ApiPatch(t, url.Path, myTok, JSON(t, key)), 200, nil)
		f2.Do(t, server.Client())
	})

	f1.Do(t, server.Client())
}

func Test_ObjectFlow(t *testing.T) {
	f := OpenFile(t, "data")
	defer f.Close()

	f1 := NewFlow()

	f1.Add(ApiPost(t, "/api/objects", myTok, nil), 400, nil)
	f1.Add(ApiPost(t, "/api/objects?name=file", myTok, f), 201, func(t *testing.T, r *http.Request, b []byte) {
		o := struct {
			URL string
		}{}

		if err := json.Unmarshal(b, &o); err != nil {
			t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
		}

		url, _ := url.Parse(o.URL)

		f2 := NewFlow()

		f2.Add(ApiGet(t, "/api/objects?search=fil", myTok), 200, checkJSONResponseSizeApprox(1))
		f2.Add(ApiGet(t, url.Path, myTok), 200, nil)
		f2.Add(ApiGet(t, url.Path, yourTok), 404, nil)
		f2.Add(ApiDelete(t, url.Path, myTok), 204, nil)
		f2.Add(ApiGet(t, "/api/objects?search=fil", myTok), 200, checkJSONResponseSize(0))
		f2.Do(t, server.Client())
	})

	f1.Do(t, server.Client())
}

func Test_CronFlow(t *testing.T) {
	f1 := NewFlow()

	f1.Add(ApiPost(t, "/api/cron", nil, nil), 404, nil)
	f1.Add(ApiPost(t, "/api/cron", myTok, nil), 400, nil)
	f1.Add(ApiPost(t, "/api/cron", myTok, JSON(t, map[string]interface{}{
		"name": "Nightly",
		"schedule": "foo",
	})), 400, nil)
	f1.Add(ApiPost(t, "/api/cron", myTok, JSON(t, map[string]interface{}{
		"name":     "Nightly",
		"schedule": "daily",
		"manifest": `driver:
  type: qemu
  image: centos/7`,
	})), 201, func(t *testing.T, r *http.Request, b []byte) {
		c := struct {
			URL string
		}{}

		if err := json.Unmarshal(b, &c); err != nil {
			t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
		}

		url, _ := url.Parse(c.URL)

		f2 := NewFlow()
		f2.Add(ApiPatch(t, url.Path, myTok, JSON(t, map[string]interface{}{
			"name": "Daily Build",
		})), 200, nil)
		f2.Add(ApiDelete(t, url.Path, myTok), 204, nil)
		f2.Do(t, server.Client())
	})
	f1.Add(ApiPost(t, "/api/cron", myTok, JSON(t, map[string]interface{}{
		"namespace": "cronspace",
		"name":      "Nightly",
		"schedule":  "daily",
		"manifest":  `driver:
  type: qemu
  image: centos/7`,
	})), 201, nil)

	f1.Do(t, server.Client())
}
