package integration

import (
	"testing"

	"djinn-ci.com/integration/djinn"
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
