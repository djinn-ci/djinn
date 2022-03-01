package integration

import (
	"testing"

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

	if v.Value != variable.MaskString {
		t.Fatalf("unexpected value for masked variable, expected=%q, got=%q\n", variable.MaskString, v.Value)
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
