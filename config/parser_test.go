package config

import (
	"strings"
	"testing"
)

func Test_Parser(t *testing.T) {
	p := newParser("-", strings.NewReader(cfg), errh(t))

	nodes := p.parse()

	if err := p.err(); err != nil {
		t.Fatal(err)
	}

	expected := 4

	if l := len(nodes); l != expected {
		t.Fatalf("unexpected number of nodes, expected=%d, got=%d\n", expected, l)
	}

	for _, n := range nodes {
		n.walk(func(n *node) {


			println(n.name, n.label, n.value)
		})
	}
}
