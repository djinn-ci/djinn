package types

import (
	"testing"
)

func TestScopeUnmarshal(t *testing.T) {
	buildReadWrite := Scope([]scopeItem{
		{Build, Read|Write},
	})

	buildAllNamespaceRead := Scope([]scopeItem{
		{Build, Read|Write|Delete},
		{Namespace, Read},
	})

	variableAll := Scope([]scopeItem{
		{Variable, Read|Write|Delete},
	})

	tests := []struct{
		scopeString   string
		expectedScope Scope
	}{
		{"build:read build:write", buildReadWrite},
		{"build:read,write", buildReadWrite},
		{"build:read,write,delete namespace:read", buildAllNamespaceRead},
		{"variable:read variable:write variable:delete", variableAll},
	}

	for _, test := range tests {
		sc, err := UnmarshalScope(test.scopeString)

		if err != nil {
			t.Fatal(err)
		}

		diff := ScopeDiff(sc, test.expectedScope)

		if len(diff) > 0 {
			t.Errorf("scope mismatch\n\texpected = '%s'\n\tactual   = '%s'\n", test.expectedScope.String(), sc.String())
		}
	}
}

func TestScopeScan(t *testing.T) {
	buildReadWrite := Scope([]scopeItem{
		{Build, Read|Write},
	})

	buildAllNamespaceRead := Scope([]scopeItem{
		{Build, Read|Write|Delete},
		{Namespace, Read},
	})

	tests := []struct{
		bytes         []byte
		expectedScope Scope
	}{
		{[]byte{2, 3}, buildReadWrite},
		{[]byte{2, 7, 6, 1}, buildAllNamespaceRead},
	}

	for _, test := range tests {
		sc := Scope(make([]scopeItem, 0))

		if err := sc.Scan(test.bytes); err != nil {
			t.Fatal(err)
		}

		diff := ScopeDiff(sc, test.expectedScope)

		if len(diff) > 0 {
			t.Errorf("scope mismatch\n\texpected = '%s'\n\tactual   = '%s'\n", test.expectedScope.String(), sc.String())
		}
	}
}
