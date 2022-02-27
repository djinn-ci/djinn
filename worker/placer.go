package worker

import (
	"io"
	"os"
	"strings"

	"djinn-ci.com/errors"
	"djinn-ci.com/key"
	"djinn-ci.com/runner"
)

type placer struct {
	chain  *key.Chain
	placer runner.Placer
}

var _ runner.Placer = (*placer)(nil)

func (p *placer) Place(name string, w io.Writer) (int64, error) {
	if name == p.chain.ConfigName() {
		n, err := io.WriteString(w, p.chain.Config())

		if err != nil {
			return 0, errors.Err(err)
		}
		return int64(n), nil
	}

	if strings.HasPrefix(name, "key:") {
		n, err := p.chain.Place(strings.TrimPrefix(name, "key:"), w)

		if err != nil {
			return 0, errors.Err(err)
		}
		return n, nil
	}

	n, err := p.placer.Place(name, w)

	if err != nil {
		return 0, errors.Err(err)
	}
	return int64(n), nil
}

func (p *placer) Stat(name string) (os.FileInfo, error) {
	if name == p.chain.ConfigName() {
		return p.chain.ConfigFileInfo(), nil
	}

	if strings.HasPrefix(name, "key:") {
		info, err := p.chain.Stat(strings.TrimPrefix(name, "key:"))

		if err != nil {
			return nil, errors.Err(err)
		}
		return info, nil
	}

	info, err := p.placer.Stat(name)

	if err != nil {
		return nil, errors.Err(err)
	}
	return info, nil
}
