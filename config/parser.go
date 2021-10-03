package config

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type node struct {
	pos   pos
	lit   litKind
	name  string
	label string
	value string
	body  *node
	list  *node
	next  *node
}

type parser struct {
	*scanner

	errc int
}

func newParser(name string, r io.Reader, errh func(string, int, int, string)) *parser {
	p := &parser{}

	src := newSource(name, r, func(name string, line, col int, msg string) {
		p.errc++
		errh(name, line, col, msg)
	})
	p.scanner = newScanner(src)
	return p
}

// err returns an error with the given message for the current node. This will
// include the node's position in the error message.
func (n *node) err(msg string) error {
	return fmt.Errorf("%s - %s", n.pos.String(), msg)
}

func (n *node) walk(visit func(*node)) {
	if n.body != nil {
		n.body.walk(visit)
	}

	if n.list != nil {
		n.list.walk(visit)
	}

	if n.next != nil {
		n.next.walk(visit)
	}
	visit(n)
}

func (p *parser) syntaxError(toks ...token) {
	msg := "unexpected token " + p.tok.String()

	if len(toks) == 0 {
		p.scanner.err(msg)
		return
	}

	if len(toks) > 1 {
		tokstrings := make([]string, 0, len(toks))

		for _, tok := range toks {
			tokstrings = append(tokstrings, tok.String())
		}
		p.scanner.err(msg + ", expected one of " + strings.Join(tokstrings, ", "))
		return
	}
	p.scanner.err("unexpected token " + p.tok.String() + ", expected " + toks[0].String())
}

func (p *parser) err() error {
	if p.errc > 0 {
		return fmt.Errorf("parser encountered %d error(s)", p.errc)
	}
	return nil
}

func (p *parser) list(root **node, start, sep, end token, parse func(*parser) (*node, bool)) {
	if p.tok != start {
		p.syntaxError(start)
		return
	}

	p.next()

	for p.tok != _EOF && p.tok != end {
		n, ok := parse(p)

		if !ok {
			break
		}

		(*root) = n
		root = &(*root).next

		p.next()

		if p.tok != sep {
			if p.tok != end && p.tok != _EOF {
				p.syntaxError(sep)
				break
			}
			continue
		}
		p.next()
	}
	p.next()
}

func (p *parser) array(label, name string) *node {
	n := &node{
		pos:   p.pos,
		label: label,
		name:  name,
	}

	p.list(&n.list, _Lbrack, _Comma, _Rbrack, func(p *parser) (*node, bool) {
		if p.tok != _Literal {
			p.syntaxError(_Literal)
			return nil, false
		}
		return &node{
			pos:   p.pos,
			value: p.lit,
			lit:   p.litKind,
		}, true
	})
	return n
}

func (p *parser) block(label, name string) *node {
	n := &node{
		pos:   p.pos,
		label: label,
		name:  name,
	}

	if p.tok != _Lbrace {
		p.syntaxError(_Lbrace)
		return nil
	}

	p.next()

	root := &n.body

	for p.tok != _EOF && p.tok != _Rbrace {
		name := ""
		label := ""

		if p.tok == _Name {
			name = p.lit
			p.next()
		}

		if p.tok == _Name {
			label = p.lit
			p.next()
		}

		if p.tok == _Literal {
			(*root) = &node{
				pos:   p.pos,
				lit:   p.litKind,
				label: label,
				name:  name,
				value: p.lit,
			}
			root = &(*root).next
			p.next()
			continue
		}

		if p.tok == _Lbrack {
			(*root) = p.array(label, name)
			root = &(*root).next
			continue
		}

		if p.tok == _Lbrace {
			(*root) = p.block(label, name)
			root = &(*root).next
			continue
		}
		p.syntaxError()
		break
	}
	p.next()
	return n
}

func (p *parser) parse() []*node {
	nodes := make([]*node, 0)

	for p.tok != _EOF {
		name := ""
		label := ""

		if p.tok == _Include {
			p.next()

			if p.tok != _Literal {
				p.scanner.err("expected string literal")
				p.next()
				continue
			}

			if p.litKind != stringLit {
				p.scanner.err("expected string literal")
				p.next()
				continue
			}

			f, err := os.Open(p.lit)

			if err != nil {
				p.scanner.err(err.Error())
				p.next()
				continue
			}

			defer f.Close()

			p2 := newParser(f.Name(), f, p.errh)

			nodes = append(nodes, p2.parse()...)
			p.next()
			continue
		}

		if p.tok == _Name {
			name = p.lit
			p.next()
		}

		if p.tok == _Name {
			label = p.lit
			p.next()
		}

		if p.tok == _Literal {
			nodes = append(nodes, &node{
				pos:   p.pos,
				lit:   p.litKind,
				label: label,
				name:  name,
				value: p.lit,
			})
			p.next()
			continue
		}

		if p.tok == _Lbrack {
			nodes = append(nodes, p.array(label, name))
			continue
		}

		if p.tok == _Lbrace {
			nodes = append(nodes, p.block(label, name))
			continue
		}

		if p.tok == _Semi {
			p.next()
			continue
		}

		if name == "provider" {
			nodes = append(nodes, &node{
				pos:   p.pos,
				label: label,
				name:  name,
			})
			continue
		}
		p.syntaxError()
		p.next()
	}
	return nodes
}
