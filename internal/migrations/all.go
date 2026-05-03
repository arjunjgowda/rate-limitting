package migrations

import (
	"gofr.dev/pkg/gofr/migration"
)

// All returns all the migrations for the application
func All() map[int64]migration.Migrate {
	return map[int64]migration.Migrate{
		20240502183000: createUsersTable(),
		20240503230000: createOutboxTable(),
	}
}
