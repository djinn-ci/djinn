package oauth2

import "testing"

var (
	buildReadWrite = Scope([]scopeItem{
		{Build, Read|Write},
	})

	buildAllNamespaceRead = Scope([]scopeItem{
		{Build, Read|Write|Delete},
		{Namespace, Read},
	})

	variableAll = Scope([]scopeItem{
		{Variable, Read|Write|Delete},
	})
)

func Test_UnmarshalScope(t *testing.T) {
	tests := []struct{
		scope    string
		expected Scope
	}{
		{"build:read build:write", buildReadWrite},
		{"build:read,write", buildReadWrite},
		{"build:read,write,delete namespace:read", buildAllNamespaceRead},
		{"variable:read variable:write variable:delete", variableAll},
	}

	for i, test := range tests {
		sc, err := UnmarshalScope(test.scope)

		if err != nil {
			t.Fatal(err)
		}

		if diff := ScopeDiff(sc, test.expected); len(diff) > 0 {
			t.Errorf("test[%d] - expected = '%s' actual = '%s'\n", i, test.expected, sc)
		}
	}
}

func Test_ScopeScan(t *testing.T) {
	tests := []struct{
		b        []byte
		expected Scope
	}{
		{[]byte{1, 3}, buildReadWrite},
		{[]byte{1, 7, 5, 1}, buildAllNamespaceRead},
	}

	for i, test := range tests {
		sc := Scope(make([]scopeItem, 0))

		if err := sc.Scan(test.b); err != nil {
			t.Fatal(err)
		}

		if diff := ScopeDiff(sc, test.expected); len(diff) > 0 {
			t.Errorf("test[%d] - expected = '%s' actual = '%s'\n", i, test.expected, sc)
		}
	}
}
