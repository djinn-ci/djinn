package integration

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"testing"

	"djinn-ci.com/auth"
	"djinn-ci.com/env"
	"djinn-ci.com/integration/djinn"
)

func Test_UserPerms(t *testing.T) {
	freeman, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, env.DJINN_API_SERVER, t)

	n, err := djinn.CreateNamespace(freeman, djinn.NamespaceParams{
		Name:       "blueshift",
		Visibility: djinn.Public,
	})

	if err != nil {
		t.Fatal(err)
	}

	v, err := djinn.CreateVariable(freeman, djinn.VariableParams{
		Namespace: n.Path,
		Key:       "blue",
		Value:     "shift",
	})

	if err != nil {
		t.Fatal(err)
	}

	var data bytes.Buffer
	blusqr(&data)

	o, err := djinn.CreateObject(freeman, djinn.ObjectParams{
		Namespace: n.Path,
		Name:      "blueshift",
		Object:    &data,
	})

	if err != nil {
		t.Fatal(err)
	}

	i, err := djinn.CreateImage(freeman, djinn.ImageParams{
		Namespace: n.Path,
		Name:      "blueshift",
		Image:     bytes.NewReader(qcow2number),
	})

	if err != nil {
		t.Fatal(err)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		t.Fatal(err)
	}

	b := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	k, err := djinn.CreateKey(freeman, djinn.KeyParams{
		Namespace: n.Path,
		Name:      "blueshift",
		Key:       string(b),
	})

	if err != nil {
		t.Fatal(err)
	}

	breen, _ := djinn.NewClientWithLogger(tokens.get("wallace.breen").Token, env.DJINN_API_SERVER, t)

	deletes := map[string]func(*djinn.Client) error{
		"delete variable": v.Delete,
		"delete object":   o.Delete,
		"delete image":    i.Delete,
		"delete key":      k.Delete,
	}

	for name, fn := range deletes {
		err := fn(breen)

		if err == nil {
			t.Fatalf("expectd %q to error, it did not\n", name)
		}

		derr, ok := err.(*djinn.Error)

		if !ok {
			t.Fatalf("unexpected error, expected=%T, got=%T(%q)\n", &djinn.Error{}, err, err)
		}

		if derr.StatusCode != http.StatusNotFound {
			t.Fatalf("unexpected status code, expected=%q, got=%q\n", http.StatusText(derr.StatusCode), http.StatusText(http.StatusNotFound))
		}
	}

	gets := map[string]func(*djinn.Client) error{
		"get variable": v.Get,
		"get object":   o.Get,
		"get image":    i.Get,
		"get key":      k.Get,
	}

	for name, fn := range gets {
		if err := fn(freeman); err != nil {
			t.Fatalf("unexpected error when attempting %q - %s\n", name, err)
		}
	}

	_, err = djinn.CreateVariable(breen, djinn.VariableParams{
		Namespace: n.Path + "@" + n.User.Username,
		Key:       "blue",
		Value:     "shift",
	})

	if err == nil {
		t.Fatal("expected attempted variable creation to fail, it did not")
	}

	derr, ok := err.(*djinn.Error)

	if !ok {
		t.Fatalf("unexpected error, expected=%T, got=%T(%q)\n", &djinn.Error{}, err, err)
	}

	if len(derr.Params) > 1 {
		t.Fatalf("too many errors, expected=%d, got=%d\n", 1, len(derr.Params))
	}

	msg, ok := derr.Params["namespace"]

	if !ok {
		t.Fatalf("could not find key %q in errors map\n", "namespace")
	}

	if msg[0] != "Namespace permission denied" {
		t.Fatalf("unexpected namespace error, expected=%q, got=%q\n", auth.ErrPermission, msg[0])
	}
}
