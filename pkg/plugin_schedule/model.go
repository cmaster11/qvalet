package plugin_schedule

import (
	"time"

	"github.com/uptrace/bun"
)

const tableNameScheduledTask = "scheduled_tasks"

type ScheduledTask struct {
	bun.BaseModel `bun:"scheduled_tasks"`

	Id int64 `bun:",autoincrement"`

	// When does this task need to be executed?
	ExecuteAt time.Time `bun:",nullzero,notnull"`

	// Which listener has been invoked?
	ListenerId string `bun:",nullzero,notnull"`

	// What arguments have we passed?
	Args map[string]interface{} `bun:"type:json,nullzero"`
}
