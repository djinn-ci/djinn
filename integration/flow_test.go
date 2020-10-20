// +build integration

package integration

//import (
//	"bytes"
//	"crypto/rand"
//	"crypto/rsa"
//	"crypto/x509"
//	"encoding/json"
//	"encoding/pem"
//	"net/http"
//	"net/url"
//	"testing"
//
//	"github.com/andrewpillar/djinn/namespace"
//	"github.com/andrewpillar/djinn/oauth2"
//)

//func Test_ObjectFlow(t *testing.T) {
//	f := OpenFile(t, "data")
//	defer f.Close()
//
//	f1 := NewFlow()
//
//	f1.Add(ApiPost(t, "/api/objects", myTok, nil), 400, nil)
//	f1.Add(ApiPost(t, "/api/objects?name=file", myTok, f), 201, func(t *testing.T, r *http.Request, b []byte) {
//		o := struct {
//			URL string
//		}{}
//
//		if err := json.Unmarshal(b, &o); err != nil {
//			t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
//		}
//
//		url, _ := url.Parse(o.URL)
//
//		f2 := NewFlow()
//
//		f2.Add(ApiGet(t, "/api/objects?search=fil", myTok), 200, checkJSONResponseSizeApprox(1))
//		f2.Add(ApiGet(t, url.Path, myTok), 200, nil)
//		f2.Add(ApiGet(t, url.Path, yourTok), 404, nil)
//		f2.Add(ApiDelete(t, url.Path, myTok), 204, nil)
//		f2.Add(ApiGet(t, "/api/objects?search=fil", myTok), 200, checkJSONResponseSize(0))
//		f2.Do(t, server.Client())
//	})
//
//	f1.Do(t, server.Client())
//}
//
//func Test_CronFlow(t *testing.T) {
//	f1 := NewFlow()
//
//	f1.Add(ApiPost(t, "/api/namespaces", myTok, JSON(t, map[string]interface{}{
//		"name":       "cronspace",
//		"visibility": "private",
//	})), 201, nil)
//
//	f1.Add(ApiPost(t, "/api/namespaces", yourTok, JSON(t, map[string]interface{}{
//		"name":       "spacecron",
//		"visibility": "private",
//	})), 201, nil)
//
//	f1.Add(ApiPost(t, "/api/cron", nil, nil), 404, nil)
//	f1.Add(ApiPost(t, "/api/cron", myTok, nil), 400, nil)
//	f1.Add(ApiPost(t, "/api/cron", myTok, JSON(t, map[string]interface{}{
//		"namespace": "spacecron@you",
//		"name":      "Nightly",
//		"schedule":  "daily",
//		"manifest":  `driver:
//  type: qemu
//  image: centos/7`,
//	})), 422, nil)
//	f1.Add(ApiPost(t, "/api/cron", myTok, JSON(t, map[string]interface{}{
//		"name": "Nightly",
//		"schedule": "foo",
//	})), 400, nil)
//	f1.Add(ApiPost(t, "/api/cron", myTok, JSON(t, map[string]interface{}{
//		"name":     "Nightly",
//		"schedule": "daily",
//		"manifest": `driver:
//  type: qemu
//  image: centos/7`,
//	})), 201, func(t *testing.T, r *http.Request, b []byte) {
//		c := struct {
//			URL string
//		}{}
//
//		if err := json.Unmarshal(b, &c); err != nil {
//			t.Fatalf("%q %q unexpected Unmarshal error: %s: %q\n", r.Method, r.URL, err, string(b))
//		}
//
//		url, _ := url.Parse(c.URL)
//
//		f2 := NewFlow()
//		f2.Add(ApiPatch(t, url.Path, myTok, JSON(t, map[string]interface{}{
//			"name": "Daily Build",
//		})), 200, nil)
//		f2.Add(ApiDelete(t, url.Path, myTok), 204, nil)
//		f2.Do(t, server.Client())
//	})
//	f1.Add(ApiPost(t, "/api/cron", myTok, JSON(t, map[string]interface{}{
//		"namespace": "cronspace",
//		"name":      "Nightly",
//		"schedule":  "daily",
//		"manifest":  `driver:
//  type: qemu
//  image: centos/7`,
//	})), 201, nil)
//
//	f1.Do(t, server.Client())
//}
