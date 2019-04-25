package placer

import (
	"io"

	"github.com/andrewpillar/thrall/errors"
	"github.com/andrewpillar/thrall/model"
	"github.com/andrewpillar/thrall/runner"
)

type Database struct {
	runner.Placer

	Build *model.Build
}

func NewDatabase(p runner.Placer) *Database {
	return &Database{Placer: p}
}

func (p *Database) Place(name string, w io.Writer) error {
	u, err := model.FindUser(p.Build.UserID)

	if err != nil {
		return errors.Err(err)
	}

	o, err := u.FindObjectByName(name)

	if err != nil {
		return errors.Err(err)
	}

	if o.IsZero() {
		return errors.Err(errors.New("could not find object in database"))
	}

	if err := p.Placer.Place(name, w); err != nil {
		return errors.Err(err)
	}

	o.Placed = true

	return errors.Err(o.Update())
}
