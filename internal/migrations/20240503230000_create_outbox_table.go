package migrations

import (
	"gofr.dev/pkg/gofr/migration"
)

func createOutboxTable() migration.Migrate {
	return migration.Migrate{
		UP: func(d migration.Datasource) error {
			query := `
			CREATE TABLE IF NOT EXISTS outbox (
				id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
				topic      VARCHAR(255) NOT NULL,
				payload    JSONB NOT NULL,
				status     VARCHAR(50) DEFAULT 'PENDING',
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);`
			_, err := d.SQL.Exec(query)
			return err
		},
	}
}
