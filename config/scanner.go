package config

import "fmt"

type token uint

type litKind uint

type pos struct {
	name string
	line int
	col  int
}

type scanner struct {
	*source

	pos     pos
	tok     token
	litKind litKind
	lit     string
}

//go:generate stringer -type token -linecomment
const (
	_EOF token = iota  // eof

	_Name             // name
	_Literal          // literal

	_Semi             // newline
	_Comma            // ,

	_Lbrace           // {
	_Rbrace           // }
	_Lbrack           // [
	_Rbrack           // ]
)

const (
	boolLit litKind = iota + 1
	numberLit
	stringLit
)

func isLetter(r rune) bool {
	return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || r == '-' || r == '_';
}

func isDigit(r rune) bool {
	return '0' <= r && r <= '9';
}

func newScanner(s *source) *scanner {
	sc := &scanner{
		source: s,
	}
	sc.next()
	return sc
}

func (p pos) String() string {
	return fmt.Sprintf("%s,%d:%d", p.name, p.line, p.col)
}

func (s *scanner) ident() {
	s.startLit()

	r := s.get()

	for isLetter(r) || isDigit(r) {
		r = s.get()
	}
	s.unget()

	s.tok = _Name
	s.lit = s.stopLit()

	if s.lit == "true" || s.lit == "false" {
		s.tok = _Literal
		s.litKind = boolLit
	}
}

func (s *scanner) number() {
	s.startLit()

	r := s.get()

	for isDigit(r) {
		r = s.get()
	}
	s.unget()

	s.tok = _Literal
	s.lit = s.stopLit()
	s.litKind = numberLit
}

func (s *scanner) string() {
	s.startLit()

	r := s.get()

	for r != '"' {
		r = s.get()
	}

	lit := s.stopLit()

	s.tok = _Literal
	s.litKind = stringLit
	s.lit = lit[1:len(lit)-1]
}

func (s *scanner) comment() {
	r := s.get()

	for r != '\n' {
		r = s.get()
	}
}

func (s *scanner) next() {
redo:
	s.lit = s.lit[0:0]

	r := s.get()

	for r == ' ' || r == '\t' || r == '\r' || r == '\n' {
		r = s.get()
	}

	s.pos = s.getPos()

	if isLetter(r) {
		s.ident()
		return
	}

	if isDigit(r) {
		s.number()
		return
	}

	switch r {
	case -1:
		s.tok = _EOF
	case '{':
		s.tok = _Lbrace
	case '}':
		s.tok = _Rbrace
	case '[':
		s.tok = _Lbrack
	case ']':
		s.tok = _Rbrack
	case ',':
		s.tok = _Comma
	case '\n':
		s.tok = _Semi
	case '"':
		s.string()
	case '#':
		s.comment()
		goto redo
	default:
		s.err(fmt.Sprintf("unexpected token %U", r))
		goto redo
	}
}
