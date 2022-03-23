package integration

import (
	"strings"
	"testing"
	"time"

	"djinn-ci.com/integration/djinn"
	"djinn-ci.com/variable"
)

func Test_VariableCreate(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	v, err := djinn.CreateVariable(cli, djinn.VariableParams{
		Key:   "Test_VariableCreate",
		Value: "foo",
	})

	if err != nil {
		t.Fatal(err)
	}

	if err := v.Get(cli); err != nil {
		t.Fatal(err)
	}
}

func Test_VariableCreateMasked(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	v, err := djinn.CreateVariable(cli, djinn.VariableParams{
		Key:   "Test_VariableCreateMasked",
		Value: "foo",
		Mask:  true,
	})

	if err == nil {
		t.Fatal("expected call to djinn.CreateVariable to fail, it did not")
	}

	derr, ok := err.(*djinn.Error)

	if !ok {
		t.Fatalf("unexpected error from djinn.CreateVariable, expected=%T, got=%T(%q)\n", &djinn.Error{}, err, err)
	}

	msg, ok := derr.Params["value"]

	if !ok {
		t.Fatalf("expected parameter %q in errors\n", "value")
	}

	expectedmsg := "Masked variable length cannot be shorter than 6 characters"

	if msg[0] != expectedmsg {
		t.Fatalf("unexpected error message, expected=%q, got=%q\n", expectedmsg, msg[0])
	}

	v, err = djinn.CreateVariable(cli, djinn.VariableParams{
		Key:   "Test_VariableCreateMasked",
		Value: "foobar",
		Mask:  true,
	})

	if err != nil {
		t.Fatal(err)
	}

	if v.Value != variable.MaskString {
		t.Fatalf("unexpected value for masked variable, expected=%q, got=%q\n", variable.MaskString, v.Value)
	}
}

func Test_VariableMasking(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	secret := "secret_api_token"

	_, err := djinn.CreateVariable(cli, djinn.VariableParams{
		Namespace: "maskedvariables",
		Key:       "Test_VariableMasking",
		Value:     secret,
		Mask:      true,
	})

	if err != nil {
		t.Fatal(err)
	}

	b, err := djinn.SubmitBuild(cli, djinn.BuildParams{
		Manifest: djinn.Manifest{
			Namespace: "maskedvariables",
			Driver: map[string]string{"type": "os"},
			Stages: []string{"env1", "env2", "env3"},
			Jobs:   []djinn.ManifestJob{
				{
					Stage:    "env1",
					Commands: []string{"printenv"},
				},
				{
					Stage:    "env2",
					Commands: []string{"printenv"},
				},
				{
					Stage:    "env3",
					Commands: []string{"printenv"},
				},
			},
		},
		Comment: "Test_VariableMasking",
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

	if strings.Contains(b.Output.String, secret) {
		t.Fatalf("found unmasked secret in build output\n%s\n", b.Output.String)
	}

	jj, err := b.GetJobs(cli)

	if err != nil {
		t.Fatal(err)
	}

	for _, j := range jj {
		if strings.Contains(j.Output.String, secret) {
			t.Fatalf("found unmasked secret in build job output for job %d\n", j.ID)
		}
	}
}

func Test_VariableDelete(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, apiEndpoint, t)

	v, err := djinn.CreateVariable(cli, djinn.VariableParams{
		Key:   "Test_VariableDelete",
		Value: "foo",
	})

	if err != nil {
		t.Fatal(err)
	}

	if err := v.Delete(cli); err != nil {
		t.Fatal(err)
	}
}
