package database

import (
	"testing"
	"time"
)

func Test_Params(t *testing.T) {
	p := Params{
		"id":         CreateOnlyParam(10),
		"user_id":    CreateOnlyParam(7),
		"number":     CreateOnlyParam(365),
		"created_at": CreateOnlyParam(time.Now()),
	}

	p.Only("*")

	for _, name := range [...]string{"id", "user_id", "number", "created_at"} {
		if _, ok := p[name]; !ok {
			t.Fatalf("expected param %q to be in Params, it was not\n", name)
		}
	}

	only := []string{"id", "number"}

	p.Only(only...)

	for _, name := range [...]string{"user_id", "created_at"} {
		if _, ok := p[name]; ok {
			t.Fatalf("expected param %q to not be in Params, it was\n", name)
		}
	}
}
