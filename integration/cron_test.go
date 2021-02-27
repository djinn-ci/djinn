// +build integration

package integration

import (
	"net/http"
	"testing"
)

func Test_Cron(t *testing.T) {
	client := newClient(server)

	client.do(t, request{
		name:        "create cron namespace for user 'me'",
		method:      "POST",
		uri:         "/api/namespaces",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(map[string]string{"name": "cronspace", "visibility": "private"}),
		code:        http.StatusCreated,
	})

	client.do(t, request{
		name:        "create cron namespace for user 'you'",
		method:      "POST",
		uri:         "/api/namespaces",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(map[string]string{"name": "spacecron", "visibility": "private"}),
		code:        http.StatusCreated,
	})

	client.do(t, request{
		name:        "attempt to create cron unauthenticated",
		method:      "POST",
		uri:         "/api/cron",
		contentType: "application/json",
		code:        http.StatusNotFound,
	})

	client.do(t, request{
		name:        "attempt to create cron with no request body as 'me'",
		method:      "POST",
		uri:         "/api/cron",
		token:       myTok,
		contentType: "application/json",
		code:        http.StatusBadRequest,
	})

	client.do(t, request{
		name:        "attempt to create cron with invalid schedule as 'me'",
		method:      "POST",
		uri:         "/api/cron",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(map[string]string{"schedule": "foo"}),
		code:        http.StatusBadRequest,
		check:       checkFormErrors("schedule", "unknown schedule: foo"),
	})

	client.do(t, request{
		name:        "attempt to create cron in namespace spacecron@you as 'me'",
		method:      "POST",
		uri:         "/api/cron",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(map[string]string{"manifest": "namespace: spacecron@you"}),
		code:        http.StatusBadRequest,
	})

	manifest := "namespace: cronspace\ndriver:\n  type: qemu\n  image: centos/7"

	client.do(t, request{
		name:        "create daily cron as 'me' in namespace cronspace",
		method:      "POST",
		uri:         "/api/cron",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(map[string]string{"name": "Nightly", "manifest": manifest}),
		code:        http.StatusCreated,
	})

	client.do(t, request{
		name:        "create daily cron as 'me'",
		method:      "POST",
		uri:         "/api/cron",
		token:       myTok,
		contentType: "application/json",
		body:        jsonBody(map[string]string{"name": "Nightly", "manifest": manifest}),
		code:        http.StatusCreated,
	})
}
