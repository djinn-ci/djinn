package integration

import (
	"net/http"
	"testing"

	"djinn-ci.com/env"
	"djinn-ci.com/integration/djinn"
)

func Test_NamespaceCreate(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, env.DJINN_API_SERVER, t)

	n, err := djinn.CreateNamespace(cli, djinn.NamespaceParams{
		Name:       "TestNamespaceCreate",
		Visibility: djinn.Private,
	})

	if err != nil {
		t.Fatal(err)
	}

	if _, err := djinn.GetNamespace(cli, n.User.Username, n.Path); err != nil {
		t.Fatal(err)
	}
}

func Test_NamespaceParentCreate(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, env.DJINN_API_SERVER, t)

	n, err := djinn.CreateNamespace(cli, djinn.NamespaceParams{
		Name:       "TestNamespaceParentCreate",
		Visibility: djinn.Private,
	})

	if err != nil {
		t.Fatal(err)
	}

	child, err := djinn.CreateNamespace(cli, djinn.NamespaceParams{
		Parent:     n.Path,
		Name:       "TestNamespaceParentCreateChild",
		Visibility: djinn.Public,
	})

	if err != nil {
		t.Fatal(err)
	}

	if child.RootID != n.ID {
		t.Fatalf("unexpected namespace root_id, expected=%d, got=%d\n", n.ID, child.RootID)
	}

	if child.ParentID.Int64 != n.ID {
		t.Fatalf("unexpected namespace parent_id, expected=%d, got=%d\n", n.ID, child.ParentID.Int64)
	}

	if child.Visibility != n.Visibility {
		t.Fatalf("unexpected namespace visiblity, expected=%q, got=%q\n", n.Visibility, child.Visibility)
	}
}

func Test_NamespaceUpdate(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, env.DJINN_API_SERVER, t)

	n, err := djinn.CreateNamespace(cli, djinn.NamespaceParams{
		Name:       "TestNamespaceUpdate",
		Visibility: djinn.Private,
	})

	if err != nil {
		t.Fatal(err)
	}

	child, err := djinn.CreateNamespace(cli, djinn.NamespaceParams{
		Parent: n.Path,
		Name:   "TestNamespaceUpdateChild",
	})

	if err != nil {
		t.Fatal(err)
	}

	if err := n.Update(cli, djinn.NamespaceParams{Visibility: djinn.Internal}); err != nil {
		t.Fatal(err)
	}

	if err := child.Get(cli); err != nil {
		t.Fatal(err)
	}

	if child.Visibility != n.Visibility {
		t.Fatalf("unexpected namespace visibility for child, expected=%q, got=%q\n", n.Visibility, child.Visibility)
	}
}

func Test_NamespaceDelete(t *testing.T) {
	cli, _ := djinn.NewClientWithLogger(tokens.get("gordon.freeman").Token, env.DJINN_API_SERVER, t)

	n, err := djinn.CreateNamespace(cli, djinn.NamespaceParams{
		Name:       "TestNamespaceDelete",
		Visibility: djinn.Private,
	})

	if err != nil {
		t.Fatal(err)
	}

	child, err := djinn.CreateNamespace(cli, djinn.NamespaceParams{
		Parent: n.Path,
		Name:   "TestNamespaceDeleteChild",
	})

	if err != nil {
		t.Fatal(err)
	}

	if err := n.Delete(cli); err != nil {
		t.Fatal(err)
	}

	err = child.Get(cli)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	djinnerr, ok := err.(*djinn.Error)

	if !ok {
		t.Fatalf("unexpected error type, expected=%T, got=%T (%q)\n", djinn.Error{}, err, err)
	}

	if djinnerr.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected http status, expected=%q, got=%q\n", http.StatusText(http.StatusNotFound), http.StatusText(djinnerr.StatusCode))
	}
}
