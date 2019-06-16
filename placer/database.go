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
	Users *model.UserStore
}

func NewDatabase(p runner.Placer) *Database {
	return &Database{Placer: p}
}

func (p *Database) Place(name string, w io.Writer) (int64, error) {
	u, err := p.Users.Find(p.Build.UserID)

	if err != nil {
		return 0, errors.Err(err)
	}

	o, err := u.ObjectStore().FindByName(name)

	if err != nil {
		return 0, errors.Err(err)
	}

	if o.IsZero() {
		return 0, errors.Err(errors.New("could not find object '" + name + "'"))
	}

	bo, err := o.BuildObjectStore().First()

	if err != nil {
		return 0, errors.Err(err)
	}

	n, placeErr := p.Placer.Place(name, w)

	bo.Placed = placeErr == nil

	if err := bo.Update(); err != nil {
		return n, errors.Err(err)
	}

	return n, errors.Err(placeErr)
}
