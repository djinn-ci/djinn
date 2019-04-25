package model

import (
	"database/sql"
	"time"

	"github.com/andrewpillar/thrall/errors"

	"github.com/lib/pq"
)

type Job struct {
	Model

	BuildID    int64          `db:"build_id"`
	StageID    int64          `db:"stage_id"`
	Parent     sql.NullInt64  `db:"parent"`
	Name       string         `db:"name"`
	Commands   string         `db:"commands"`
	Status     Status         `db:"status"`
	Output     sql.NullString `db:"output"`
	StartedAt  *pq.NullTime   `db:"started_at"`
	FinishedAt *pq.NullTime   `db:"finished_at"`

	Build        *Build
	Stage        *Stage
	Artifacts    []*Artifact
	Dependencies []*Job
}

func (j *Job) Create() error {
	stmt, err := DB.Prepare(`
		INSERT INTO jobs (build_id, stage_id, name, commands)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`)

	if err != nil {
		return errors.Err(err)
	}

	defer stmt.Close()

	err = stmt.QueryRow(j.BuildID, j.StageID, j.Name, j.Commands).Scan(&j.ID)

	return errors.Err(err)
}

func (j *Job) LoadRelations() error {
	if j.Build == nil {
		j.Build = &Build{}

		err := DB.Get(j.Build, "SELECT * FROM builds WHERE id = $1", j.BuildID)

		return errors.Err(err)
	}

	if j.Stage == nil {
		j.Stage = &Stage{}

		err := DB.Get(j.Stage, "SELECT * FROM stages WHERE id = $1", j.StageID)

		return errors.Err(err)
	}

	return  nil
}

func (j Job) IsZero() bool {
	return j.ID == 0                                          &&
			j.BuildID == 0                                    &&
			j.StageID == 0                                    &&
			!j.Parent.Valid                                   &&
			j.Name == ""                                      &&
			j.Status == Status(0)                             &&
			j.Output.String == ""                             &&
			!j.Output.Valid                                   &&
			j.CreatedAt == nil || *j.CreatedAt == time.Time{} &&
			j.StartedAt.Time == time.Time{}                   &&
			!j.StartedAt.Valid                                &&
			j.FinishedAt.Time == time.Time{}                  &&
			!j.FinishedAt.Valid
}
