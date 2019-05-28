package placer

import (
	"io"
	"os"

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

func (p *Database) Place(name string, w io.Writer) error {
	u, err := p.Users.Find(p.Build.UserID)

	if err != nil {
		return errors.Err(err)
	}

	o, err := u.ObjectStore().FindByName(name)

	if err != nil {
		return errors.Err(err)
	}

	if o.IsZero() {
		return errors.Err(errors.New("could not find object '" + name + "'"))
	}

	bo, err := o.BuildObjectStore().First()

	if err != nil {
		return errors.Err(err)
	}

	placeErr := p.Placer.Place(name, w)

	bo.Placed = placeErr == nil

	if err := bo.Update(); err != nil {
		return errors.Err(err)
	}

	return errors.Err(placeErr)
}

func (p *Database) Stat(name string) (os.FileInfo, error) {
	u, err := p.Users.Find(p.Build.UserID)

	if err != nil {
		return nil, errors.Err(err)
	}

	o, err := u.ObjectStore().FindByName(name)

	if err != nil {
		return nil, errors.Err(err)
	}

	if o.IsZero() {
		return nil, errors.Err(errors.New("could not find object '" + name + "'"))
	}

	info, err := p.Placer.Stat(name)

	return info, errors.Err(err)
}
