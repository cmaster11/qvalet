package plugin_schedule

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		if _, err := db.NewCreateTable().Model((*ScheduledTask)(nil)).IfNotExists().Exec(ctx); err != nil {
			return err
		}

		if _, err := db.Exec(fmt.Sprintf("CREATE INDEX %s_listener_id ON %s (listener_id)", tableNameScheduledTask, tableNameScheduledTask)); err != nil {
			return err
		}

		return nil
	}, nil)
}
