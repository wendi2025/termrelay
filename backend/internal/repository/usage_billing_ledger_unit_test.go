//go:build unit

package repository

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

const ledgerDebitSQLPattern = `(?s)^\s*WITH credits AS`

func TestRecordUsageBillingLedgerDebit_ExecutesFIFOAllocation(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &usageBillingRepository{db: db}
	mock.ExpectExec(ledgerDebitSQLPattern).
		WithArgs(int64(42), 2.5, "usage:req-1").
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo.recordUsageBillingLedgerDebit(context.Background(), 42, 2.5, "usage:req-1")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecordUsageBillingLedgerDebit_SwallowsWriteError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &usageBillingRepository{db: db}
	mock.ExpectExec(ledgerDebitSQLPattern).
		WithArgs(int64(42), 1.0, "usage:req-2").
		WillReturnError(errors.New("ledger unavailable"))

	// 账本写入失败必须被吞掉：不能影响已提交的计费事务
	repo.recordUsageBillingLedgerDebit(context.Background(), 42, 1.0, "usage:req-2")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRecordUsageBillingLedgerDebit_SkipsInvalidInput(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	repo := &usageBillingRepository{db: db}
	// 全部为无效输入：不应对数据库发起任何语句
	repo.recordUsageBillingLedgerDebit(context.Background(), 0, 1.0, "usage:req-3")
	repo.recordUsageBillingLedgerDebit(context.Background(), 42, 0, "usage:req-3")
	repo.recordUsageBillingLedgerDebit(context.Background(), 42, -1, "usage:req-3")
	repo.recordUsageBillingLedgerDebit(context.Background(), 42, 1.0, "   ")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLedgerDebitMemo(t *testing.T) {
	require.Equal(t, "usage:req-1", ledgerDebitMemo("usage", " req-1 "))
	require.Equal(t, "batch_image:b-1", ledgerDebitMemo("batch_image", "b-1"))
	require.Equal(t, "usage:req-2", ledgerDebitMemo("", "req-2"))
	require.Equal(t, "", ledgerDebitMemo("usage", "   "))
	require.True(t, strings.HasPrefix(ledgerDebitMemo("usage", "x"), "usage:"))
}
