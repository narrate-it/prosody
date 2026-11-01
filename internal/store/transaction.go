package store

import (
	"context"
	"database/sql"
)

// Row keeps the adapter compatible with AMOS's callback-facing row interface.
type Row interface{ Scan(...any) error }

// Transaction exposes only caller-held transaction operations, never commit or
// rollback. Construct it from the AMOS InvocationContext DB in the host adapter.
// Supplying callbacks is not proof of authorization or held-authority finality.
type Transaction struct {
	ExecContext     func(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext func(context.Context, string, ...any) Row
}

func (t Transaction) valid() bool { return t.ExecContext != nil && t.QueryRowContext != nil }

// FromSQL adapts an already-open transaction for migrations/tests and native
// host-owned operations. Production MCP handlers use their restricted DBTX.
func FromSQL(tx *sql.Tx) Transaction {
	if tx == nil {
		return Transaction{}
	}
	return Transaction{ExecContext: tx.ExecContext, QueryRowContext: func(ctx context.Context, q string, args ...any) Row { return tx.QueryRowContext(ctx, q, args...) }}
}
