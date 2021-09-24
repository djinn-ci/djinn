package integration

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime/debug"
	"testing"

	"djinn-ci.com/integration/djinn"
)

func apertureFlow(t *testing.T) {

}

var webhookSecret = "secret"

func webhookHandler(t *testing.T, m map[string]struct{}) func(http.ResponseWriter, *http.Request) {
	events := map[string]struct{}{
		"build.submitted": {},
		"invite.sent":     {},
		"invite.accepted": {},
		"invite.rejected": {},
		"namespaces":      {},
		"cron":            {},
		"images":          {},
		"objects":         {},
		"variables":       {},
		"ssh_keys":        {},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				w.WriteHeader(http.StatusInternalServerError)

				if err, ok := v.(error); ok {
					io.WriteString(w, err.Error()+"\n")
				}
				w.Write(debug.Stack())
				return
			}
		}()

		ev := r.Header.Get("X-Djinn-CI-Event")

		if _, ok := events[ev]; !ok {
			msg := "unknown webhook event " + ev

			t.Error(msg)

			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, msg)
			return
		}

		sig := r.Header.Get("X-Djinn-CI-Signature")

		if sig == "" {
			msg := "expected X-Djinn-CI-Signature in webhook request"

			t.Error(msg)

			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, msg)
			return
		}

		b, err := io.ReadAll(r.Body)

		if err != nil {
			t.Error(err)

			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, err.Error())
			return
		}

		actual := make([]byte, 32)

		hex.Decode(actual, []byte(sig[7:]))

		expected := hmac.New(sha256.New, []byte(webhookSecret))
		expected.Write(b)

		if !hmac.Equal(expected.Sum(nil), actual) {
			msg := "invalid request signature for webhook request"

			t.Error(msg)

			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, msg)
			return
		}

		m[ev] = struct{}{}
		w.WriteHeader(http.StatusNoContent)
	}
}

func blackMesaFlow(t *testing.T) {
	breen, _ := djinn.NewClientWithLogger(tokens.get("wallace.breen").Token, apiEndpoint, t)

	n, err := djinn.CreateNamespace(breen, djinn.NamespaceParams{
		Name:       "blackmesa",
		Visibility: djinn.Private,
	})

	if err != nil {
		t.Fatal(err)
	}

	m := make(map[string]struct{})

	srv := httptest.NewServer(http.HandlerFunc(webhookHandler(t, m)))
	defer srv.Close()

	t.Log("webhook server listening on", srv.URL)

	url, _ := url.Parse(srv.URL)

	_, port, _ := net.SplitHostPort(url.Host)

	url.Host = net.JoinHostPort("local.dev", port)

	wh, err := n.CreateWebhook(breen, djinn.WebhookParams{
		PayloadURL: url.String(),
		Secret:     webhookSecret,
		Active:     true,
		Events:     djinn.WebhookEvents,
	})

	if err != nil {
		t.Fatal(err)
	}

	if err := n.Update(breen, djinn.NamespaceParams{Description: "Black Mesa"}); err != nil {
		t.Fatal(err)
	}

	freeman, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	_, err = djinn.GetNamespace(freeman, n.User.Username, n.Path)

	if err == nil {
		t.Fatalf("expected djinn.GetNamespace(%s, %q, %q) to error, it did not\n", freeman.String(), n.User.Username, n.Path)
	}

	djinnerr, ok := err.(*djinn.Error)

	if !ok {
		t.Fatalf("unexpected error type, expected=%T, got=%T\n", &djinn.Error{}, err)
	}

	if djinnerr.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected status code, expected=%q, got=%q\n", http.StatusText(djinnerr.StatusCode), http.StatusText(http.StatusNotFound))
	}

	i, err := n.Invite(breen, "gordon.freeman@black-mesa.com")

	if err != nil {
		t.Fatal(err)
	}

	if err := i.Accept(breen); err == nil {
		t.Fatalf("expected i.Accept(%s) to error, it did not\n", breen.String())
	}

	if err := i.Accept(freeman); err != nil {
		t.Fatal(err)
	}

	v, err := djinn.CreateVariable(freeman, djinn.VariableParams{
			Namespace: "blackmesa@wallace.breen",
			Key:       "blackMesaFlow_Var",
			Value:     "freeman",
	})

	if err != nil {
		t.Fatal(err)
	}

	if v.NamespaceID.Int64 != n.ID {
		t.Fatalf("unexpected namespace id, expected=%d, got=%d\n", n.ID, v.NamespaceID.Int64)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)

	if err != nil {
		t.Fatal(err)
	}

	pem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	k, err := djinn.CreateKey(freeman, djinn.KeyParams{
		Namespace: "blackmesa@wallace.breen",
		Name:      "blackMesaFlow_Key",
		Key:       string(pem),
	})

	if err != nil {
		t.Fatal(err)
	}

	if k.NamespaceID.Int64 != n.ID {
		t.Fatalf("unexpected namespace id, expected=%d, got=%d\n", n.ID, k.NamespaceID.Int64)
	}

	_, err = djinn.CreateImage(breen, djinn.ImageParams{
		Namespace: "blackmesa@wallace.breen",
		Name:      "blackMesaFlow_Image",
		Image:     bytes.NewBuffer(qcow2number),
	})

	if err != nil {
		t.Fatal(err)
	}

	eli, _ := djinn.NewClientWithLogger(tokens.get("eli.vance").Token, apiEndpoint, t)

	var object bytes.Buffer

	blusqr(&object)

	_, err = djinn.CreateObject(eli, djinn.ObjectParams{
		Namespace: "blackmesa@wallace.breen",
		Name:      "blackMesaFlow_Object",
		Object:    &object,
	})

	if err == nil {
		t.Fatalf("expected djinn.CreateObject(%s) to error, it did not\n", eli.String())
	}

	djinnerr, ok = err.(*djinn.Error)

	if !ok {
		t.Fatalf("unexpected error type, expected=%T, got=%T\n", &djinn.Error{}, err)
	}

	if djinnerr.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected status, expected=%q, got=%q\n", http.StatusText(http.StatusBadRequest), http.StatusText(djinnerr.StatusCode))
	}

	i, err = n.Invite(freeman, "eli.vance@black-mesa.com")

	if err == nil {
		t.Fatalf("expected n.Invite(%s, %q) to error, it did not\n", freeman.String(), "eli.vance@black-mesa.com")
	}

	i, err = n.Invite(breen, "eli.vance@black-mesa.com")

	if err != nil {
		t.Fatal(err)
	}

	if err := i.Reject(eli); err != nil {
		t.Fatal(err)
	}

	blusqr(&object)

	_, err = djinn.CreateObject(freeman, djinn.ObjectParams{
		Namespace: "blackmesa@wallace.breen",
		Name:      "blackMesaFlow_Object",
		Object:    &object,
	})

	b, err := djinn.SubmitBuild(freeman, djinn.BuildParams{
		Manifest: djinn.Manifest{
			Namespace: "blackmesa@wallace.breen",
			Driver:    map[string]string{
				"type":      "docker",
				"image":     "golang",
				"workspace": "/go",
			},
		},
		Comment: "blackMesaFlow_Build",
		Tags:    []string{"flow_test"},
	})

	if err != nil {
		t.Fatal(err)
	}

	if b.NamespaceID.Int64 != n.ID {
		t.Fatalf("unexpected namespace id, expected=%d, got=%d\n", n.ID, b.NamespaceID.Int64)
	}

	_, err = djinn.CreateCron(breen, djinn.CronParams{
		Name:     "blackMesaFlow_Cron",
		Schedule: djinn.Daily,
		Manifest: djinn.Manifest{
			Namespace: "blackmesa@wallace.breen",
			Driver:    map[string]string{
				"type":      "docker",
				"image":     "golang",
				"workspace": "/go",
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	if err := n.DeleteCollaborator(breen, "gordon.freeman"); err != nil {
		t.Fatal(err)
	}

	_, err = b.Tag(freeman, "tag1")

	if err == nil {
		t.Fatalf("expected call to b.Tag(%s, %q) to error, it did not\n", freeman, "tag1")
	}

	djinnerr, ok = err.(*djinn.Error)

	if !ok {
		t.Fatalf("unexpected error type, expected=%T, got=%T\n", &djinn.Error{}, err)
	}

	if djinnerr.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected status, expected=%q, got=%q\n", http.StatusText(http.StatusNotFound), http.StatusText(djinnerr.StatusCode))
	}

	for _, ev := range wh.Events {
		if _, ok := m[ev]; !ok {
			t.Errorf("webhook event %q was not received", ev)
		}
	}
}

func Test_UsageFlow(t *testing.T) {
	blackMesaFlow(t)
	apertureFlow(t)
}
