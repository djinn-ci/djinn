package config

import (
	"io"
	"unicode/utf8"
)

type source struct {
	name        string
	r           io.Reader
	pos0, pos   int
	eof         int
	line0, line int
	col0, col   int
	buf         [4096]byte
	lit         int
	errh        func(string, int, int, string)
}

func newSource(name string, r io.Reader, errh func(string, int, int, string)) *source {
	return &source{
		name: name,
		r:    r,
		line: 1,
		errh: errh,
	}
}

func (s *source) err(msg string) {
	s.errh(s.name, s.line, s.col, msg)
}

func (s *source) getPos() pos {
	return pos{
		name: s.name,
		line: s.line,
		col:  s.col,
	}
}

func (s *source) get() rune {
redo:
	s.pos0, s.line0, s.col0 = s.pos, s.line, s.col

	if s.pos == 0 || s.pos >= len(s.buf) {
		n, err := s.r.Read(s.buf[0:])

		if err != nil {
			if err != io.EOF {
				s.err("io error: " + err.Error())
			}
			return -1
		}

		s.pos = 0
		s.eof = n
	}

	if s.pos == s.eof {
		return -1
	}

	b := s.buf[s.pos]

	if b >= utf8.RuneSelf {
		r, w := utf8.DecodeRune(s.buf[s.pos:])

		s.pos += w
		s.col += w

		return r
	}

	if b == 0 {
		s.err("invalid NUL byte")
		goto redo
	}

	s.pos++
	s.col++

	if b == '\n' {
		s.line++
		s.col = 0
	}
	return rune(b)
}

func (s *source) unget() {
	s.pos, s.line, s.col = s.pos0, s.line0, s.col0
}

func (s *source) startLit() { s.lit = s.pos0 }

func (s *source) stopLit() string {
	if s.lit < 0 {
		panic("negative lit pos")
	}

	lit := s.buf[s.lit:s.pos]
	s.lit = -1
	return string(lit)
}
