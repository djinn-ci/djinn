package integration

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
//	"net/http/httptest"
//	"net/url"
	"sync"
	"testing"

//	"djinn-ci.com/integration/djinn"
)

var webhookSecret = "secret"

func webhookHandler(t *testing.T, m *sync.Map) func(http.ResponseWriter, *http.Request) {
	events := map[string]struct{}{
		"build.submitted": {},
		"invite.sent":     {},
		"invite.accepted": {},
		"cron":            {},
		"images":          {},
		"objects":         {},
		"variables":       {},
		"ssh_keys":        {},
	}

	return func(w http.ResponseWriter, r *http.Request) {
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

		actual := make([]byte, 20)

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

		m.Store(ev, struct{}{})
		w.WriteHeader(http.StatusNoContent)
	}
}

func Test_Webhooks(t *testing.T) {
//	var m sync.Map
//
//	srv := httptest.NewServer(http.HandlerFunc(webhookHandler(t, &m)))
//	defer srv.Close()
//
//	cli, _ := djinn.NewClient(tokens.get("gordon.freeman").Token, apiEndpoint)
//
//	n, err := djinn.CreateNamespace(cli, djinn.NamespaceParams{
//		Name:       "TestNamespaceWebhooks",
//		Visibility: djinn.Private,
//	})
//
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	url, _ := url.Parse(srv.URL)
//	url.Host = "local.dev"
//
//	_, err = n.CreateWebhook(cli, djinn.WebhookParams{
//		PayloadURL: url.String(),
//		Secret:     webhookSecret,
//		Active:     true,
//		Events: []string{
//			"build.submitted",
//			"invite.sent",
//			"invite.accepted",
//			"namespaces",
//			"cron",
//			"images",
//			"objects",
//			"variables",
//			"ssh_keys",
//		},
//	})
//
//	if err != nil {
//		t.Fatal(err)
//	}
}
