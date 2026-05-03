package migrations

import (
	"gofr.dev/pkg/gofr/migration"
)

func createUsersTable() migration.Migrate {
	return migration.Migrate{
		UP: func(d migration.Datasource) error {
			query := `
			CREATE TABLE IF NOT EXISTS users (
				id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				username VARCHAR(255) NOT NULL UNIQUE,
				password VARCHAR(255) NOT NULL,
				balance  DECIMAL(10, 2) DEFAULT 0,
				email    VARCHAR(255) NOT NULL UNIQUE,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);`
			_, err := d.SQL.Exec(query)
			return err
		},
	}
}
