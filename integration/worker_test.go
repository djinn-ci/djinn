package integration

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"djinn-ci.com/env"
	"djinn-ci.com/integration/djinn"
)

func Test_WorkerOSDriver(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, env.DJINN_API_SERVER, t)

	b, err := djinn.SubmitBuild(cli, djinn.BuildParams{
		Manifest: djinn.Manifest{
			Driver: map[string]string{
				"type": "os",
			},
			Stages: []string{"os-release"},
			Jobs: []djinn.ManifestJob{
				{
					Stage:    "os-release",
					Commands: []string{"cat /etc/os-release"},
				},
			},
		},
		Comment: "Test_BuildOSDriver",
		Tags:    []string{"os"},
	})

	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(time.Second):
				if err := b.Get(cli); err != nil {
					t.Fatal(err)
				}

				if b.Status != djinn.Queued && b.Status != djinn.Running {
					done <- struct{}{}
					return
				}
			}
		}
	}()

	select {
	case <-time.After(time.Second * 5):
		done <- struct{}{}
	case <-done:
	}

	if b.Status != djinn.Passed {
		t.Fatalf("unexpected status, expected=%q, got=%q\n", djinn.Passed, b.Status)
	}
}

func Test_WorkerBinaryOutput(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, env.DJINN_API_SERVER, t)

	b, err := djinn.SubmitBuild(cli, djinn.BuildParams{
		Manifest: djinn.Manifest{
			Namespace: "",
			Driver: map[string]string{
				"type": "os",
			},
			Stages: []string{"/dev/urandom"},
			Jobs: []djinn.ManifestJob{
				{
					Stage: "/dev/urandom",
					Commands: []string{
						`printf \n\0\0\0\0\0\0\0\0\n`,
						"head -c 1024 /dev/urandom",
						`printf \n\0\0\0\0\0\0\0\0\n`,
					},
				},
			},
		},
		Comment: "Test_BuildBinaryOutput",
	})

	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(time.Second):
				if err := b.Get(cli); err != nil {
					t.Fatal(err)
				}

				if b.Status != djinn.Queued && b.Status != djinn.Running {
					done <- struct{}{}
					return
				}
			}
		}
	}()

	select {
	case <-time.After(time.Second * 5):
		done <- struct{}{}
	case <-done:
	}

	if b.Status != djinn.Passed {
		t.Fatalf("unexpected status, expected=%q, got=%q\n", djinn.Passed, b.Status)
	}
}

func Test_WorkerKillBuild(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, env.DJINN_API_SERVER, t)

	b, err := djinn.SubmitBuild(cli, djinn.BuildParams{
		Manifest: djinn.Manifest{
			Driver: map[string]string{
				"type": "os",
			},
			Stages: []string{"sleep"},
			Jobs: []djinn.ManifestJob{
				{
					Stage:    "sleep",
					Commands: []string{"sleep 5"},
				},
			},
		},
		Comment: "Test_WorkerKillBuild",
		Tags:    []string{"kill"},
	})

	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(time.Second):
				if err := b.Get(cli); err != nil {
					t.Fatal(err)
				}

				if b.Status == djinn.Running {
					done <- struct{}{}
					return
				}
			}
		}
	}()

	select {
	case <-time.After(time.Second * 5):
		done <- struct{}{}
	case <-done:
	}

	if err := b.Get(cli); err != nil {
		t.Fatal(err)
	}

	if err := b.Kill(cli); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)

	if err := b.Get(cli); err != nil {
		t.Fatal(err)
	}

	if b.Status != djinn.Killed {
		t.Fatalf("unexpected status, expected=%q, got=%q\n", djinn.Killed, b.Status)
	}
}

func Test_WorkerCollectArtifacts(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, env.DJINN_API_SERVER, t)

	b, err := djinn.SubmitBuild(cli, djinn.BuildParams{
		Manifest: djinn.Manifest{
			Driver: map[string]string{
				"type": "os",
			},
			Stages: []string{"collect"},
			Jobs: []djinn.ManifestJob{
				{
					Stage:    "collect",
					Commands: []string{"cat /etc/os-release"},
					Artifacts: djinn.ManifestPassthrough{
						"/etc/os-release": "",
						"/etc/hosts":      "",
						"/etc/shells":     "",
					},
				},
			},
		},
		Comment: "Test_WorkerCollectArtifacts",
		Tags:    []string{"artifacts"},
	})

	if err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(time.Second):
				if err := b.Get(cli); err != nil {
					t.Fatal(err)
				}

				if b.Status != djinn.Queued && b.Status != djinn.Running {
					done <- struct{}{}
					return
				}
			}
		}
	}()

	select {
	case <-time.After(time.Second * 5):
		done <- struct{}{}
	case <-done:
	}

	aa := make([]*djinn.Artifact, 0)

	resp, err := cli.Get(b.ArtifactsURL.Path, "application/json")

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&aa); err != nil {
		t.Fatal(err)
	}

	if l := len(aa); l != 3 {
		t.Fatalf("unexpected artifacts length, expected=%d, got=%d\n", 3, l)
	}

	for _, a := range aa {
		func() {
			resp, err := cli.Get(a.URL.Path, "application/octet-stream")

			if err != nil {
				t.Fatal(err)
			}

			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("could not download artifact %q - %s\n", a.Name, http.StatusText(resp.StatusCode))
			}
		}()
	}
}
