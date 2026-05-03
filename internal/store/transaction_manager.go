package store

import (
	"github.com/arjunjgowda/rate-limitting/internal/service"
	"gofr.dev/pkg/gofr"
)

type txManager struct{}

func NewTransactionManager() service.TransactionManager {
	return &txManager{}
}

// WithTransaction implements the "Ideal" transaction orchestration.
// It handles the SQL Begin, Commit, and Rollback logic, keeping it out of the Service.
func (tm *txManager) WithTransaction(ctx *gofr.Context, fn func(txCtx *gofr.Context) error) error {
	// 1. Start the actual SQL transaction
	tx, err := ctx.SQL.Begin()
	if err != nil {
		return err
	}

	// 2. Execute the business logic provided by the Service
	// We pass the context along. In GoFr, the transaction is already bound to this context.
	err = fn(ctx)

	if err != nil {
		// 3. Rollback on any error from the Service
		_ = tx.Rollback()
		return err
	}

	// 4. Commit if everything went well
	return tx.Commit()
}
