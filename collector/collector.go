package collector

import (
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"
)

func New(cfg config.Collector) (runner.Collector, error) {
	switch cfg.Type {
		case "filesystem":
			return NewFileSystem(cfg.Path), nil
		default:
			return nil, errors.Err(errors.New("unknown collector " + cfg.Type))
	}
}
