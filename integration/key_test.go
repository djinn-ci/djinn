package integration

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"djinn-ci.com/integration/djinn"
)

func Test_KeyCreate(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	key, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		t.Fatal(err)
	}

	b := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	k, err := djinn.CreateKey(cli, djinn.KeyParams{
		Name: "Test_KeyCreate",
		Key:  string(b),
	})

	if err != nil {
		t.Fatal(err)
	}

	if err := k.Get(cli); err != nil {
		t.Fatal(err)
	}
}

func Test_KeyUpdate(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	key, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		t.Fatal(err)
	}

	b := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	k, err := djinn.CreateKey(cli, djinn.KeyParams{
		Name: "Test_KeyUpdate",
		Key:  string(b),
	})

	if err != nil {
		t.Fatal(err)
	}

	config := "UserKnownHostsFile /dev/null"

	if err := k.Update(cli, djinn.KeyParams{Config: config}); err != nil {
		t.Fatal(err)
	}

	if k.Config != config {
		t.Fatalf("unexpected key config after updated, expected=%q, got=%q\n", config, k.Config)
	}
}

func Test_KeyDelete(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	key, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		t.Fatal(err)
	}

	b := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	k, err := djinn.CreateKey(cli, djinn.KeyParams{
		Name: "Test_KeyDelete",
		Key:  string(b),
	})

	if err != nil {
		t.Fatal(err)
	}

	if err := k.Delete(cli); err != nil {
		t.Fatal(err)
	}
}
