package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"djinn-ci.com/integration/djinn"
)

func jsonencode(m map[string]interface{}) *bytes.Buffer {
	var buf bytes.Buffer

	json.NewEncoder(&buf).Encode(m)
	return &buf
}

// Test_UnmarshalErrors makes sure that JSON payloads with invalid data types
// received a 400 response from the API instead of a 500.
func Test_UnmarshalErrors(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	tests := []struct {
		endpoint string
		body     *bytes.Buffer
		errors   []string
	}{
		{
			"/cron",
			jsonencode(map[string]interface{}{
				"name":     1,
				"schedule": -1,
				"manifest": djinn.Manifest{
					Driver: map[string]string{
						"type":      "docker",
						"image":     "golang",
						"workspace": "/go",
					},
				},
			}),
			[]string{"name", "schedule"},
		},
	}

	errs := make(map[string][]string)

	for i, test := range tests {
		func() {
			req, err := http.NewRequest("POST", apiEndpoint+test.endpoint, test.body)

			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err := cli.Do(req)

			if err != nil {
				t.Fatal(err)
			}

			defer resp.Body.Close()

			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("tests[%d] - unexpected status, expected=%q, got=%q\n", i, http.StatusText(http.StatusBadRequest), resp.Status)
			}

			if err := json.NewDecoder(resp.Body).Decode(&errs); err != nil {
				t.Fatalf("tests[%d] - failed to decode error json %s\n", i, err)
			}

			if len(test.errors) != len(errs) {
				t.Fatalf("tests[%d] - unexpected error count, expected=%d, got=%d\n", i, len(test.errors), len(errs))
			}
		}()
	}
}
