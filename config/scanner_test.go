package config

import (
	"strings"
	"testing"
)

var cfg = `
# Comment 1
# Comment 2
host "https://djinn-ci.com"

log debug "/dev/stdout"

drivers [
	"docker",
	"qemu-x86_64",
]

net {
	listen "localhost:8080"

	# Serve over TLS.
	ssl {
		cert "/var/lib/ssl/server.crt"
		key  "/var/lib/ssl/server.key"
	}
}`

func errh(t *testing.T) func(string, int, int, string) {
	return func(name string, line, col int, msg string) {
		t.Errorf("%s,%d:%d - %s\n", name, line, col, msg)
	}
}

func Test_Scanner(t *testing.T) {
	src := newSource("-", strings.NewReader(cfg), errh(t))
	sc := newScanner(src)

	toks := []token{
		_Name,
		_Literal,
		_Name,
		_Name,
		_Literal,
		_Name,
		_Lbrack,
		_Literal,
		_Comma,
		_Literal,
		_Comma,
		_Rbrack,
		_Name,
		_Lbrace,
		_Name,
		_Literal,
		_Name,
		_Lbrace,
		_Name,
		_Literal,
		_Name,
		_Literal,
		_Rbrace,
		_Rbrace,
	}

	for i, tok := range toks {
		if sc.tok != tok {
			t.Errorf("%d - unexpected token, expected=%q, got=%q\n", i, tok.String(), sc.tok.String())
		}
		sc.next()
	}
}
