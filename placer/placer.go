package placer

import (
	"github.com/andrewpillar/thrall/config"
	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/runner"
)

func New(cfg config.Placer) (runner.Placer, error) {
	switch cfg.Type {
		case "filesystem":
			return NewFileSystem(cfg.Path), nil
		default:
			return nil, errors.Err(errors.New("unknown placer " + cfg.Type))
	}
}
