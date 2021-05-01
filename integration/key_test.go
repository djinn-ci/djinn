package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/url"
	"testing"
)

func Test_Key(t *testing.T) {
	client := newClient(server)

	key, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		t.Fatalf("unexpected rsa.GenerateKey error: %s\n", err)
	}

	b := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	keyBody := map[string]interface{}{
		"namespace": "keyspace",
		"name":      "id_rsa",
		"key":       "AAAAABBBBBCCCCC",
	}

	client.do(t, request{
		name:        "attempt to create ssh key with no body",
		method:      "POST",
		uri:         "/api/keys",
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusBadRequest,
	})

	client.do(t, request{
		name:        "attempt to create ssh key with invalid key",
		method:      "POST",
		uri:         "/api/keys",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(keyBody),
		code:        http.StatusBadRequest,
	})

	keyBody["key"] = string(b)

	client.do(t, request{
		name:        "create ssh key in namespace keyspace",
		method:      "POST",
		uri:         "/api/keys",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(keyBody),
		code:        http.StatusCreated,
	})

	delete(keyBody, "namespace")

	createResp := client.do(t, request{
		name:        "create ssh key",
		method:      "POST",
		uri:         "/api/keys",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(keyBody),
		code:        http.StatusCreated,
	})
	defer createResp.Body.Close()

	k := struct {
		URL string
	}{}

	if err := json.NewDecoder(createResp.Body).Decode(&k); err != nil {
		t.Fatalf("unexpected json.Decode error: %s\n", err)
	}

	url, err := url.Parse(k.URL)

	if err != nil {
		t.Fatalf("unexpected url.Parse error: %s\n", err)
	}

	keyBody["key"] = "foo"
	keyBody["config"] = "UserKnownHostsFile /dev/null"

	client.do(t, request{
		name:        "update ssh key",
		method:      "PATCH",
		uri:         url.Path,
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(keyBody),
		code:        http.StatusOK,
	})
}
