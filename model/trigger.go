package model

type Trigger struct {
	model

	BuildID int64       `db:"build_id"`
	Type    TriggerType `db:"trigger_type"`
	Comment string      `db:"comment"`
	Data    struct{}    `db:"data"`
}
