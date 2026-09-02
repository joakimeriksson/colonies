package database

import (
	"time"

	"github.com/colonyos/colonies/pkg/core"
)

// UsageDatabase is the append-only ledger of who ran what on whose executor.
// It is deliberately separate from the process tables: process rows are
// mutable and purged by the retention policy, usage records are neither.
type UsageDatabase interface {
	// EnsureUsageTables creates the ledger tables if they do not exist. Safe to
	// call on every server start, so an existing deployment gets them without
	// re-initializing the database.
	EnsureUsageTables() error
	// OpenUsageRecord inserts a record with outcome USAGE_RUNNING.
	OpenUsageRecord(record *core.UsageRecord) error
	// CloseUsageRecord closes the open record for a process, if any.
	CloseUsageRecord(processID string, endTime time.Time, outcome string) error
	// CloseUsageRecordsByExecutor closes every open record on an executor,
	// used when the executor is removed or found stale.
	CloseUsageRecordsByExecutor(colonyName string, executorName string, endTime time.Time, outcome string) error
	// SetMeasuredUsage updates the measured columns of the open record for a process.
	SetMeasuredUsage(processID string, cpuSeconds float64, gpuSeconds float64, memPeakBytes int64, ioReadBytes int64, ioWriteBytes int64, source string, reportTime time.Time) error

	GetUsageRecordsByProcessID(processID string) ([]*core.UsageRecord, error)
	// QueryUsageRecords returns records whose start time falls in [from, to].
	// Empty filter strings match everything.
	QueryUsageRecords(colonyName string, from time.Time, to time.Time, initiatorID string, providerID string, executorName string, limit int) ([]*core.UsageRecord, error)
	// SummarizeUsage aggregates records by one of the core.USAGE_GROUP_* keys.
	SummarizeUsage(colonyName string, from time.Time, to time.Time, groupBy string, initiatorID string, providerID string) ([]*core.UsageSummary, error)
	// UsageBalance computes consumed, provided and self-use per colony member.
	UsageBalance(colonyName string, from time.Time, to time.Time) ([]*core.UsageBalance, error)

	// SetExecutorOwner closes the current ownership row and opens a new one.
	// An empty ownerID means the colony owns the executor.
	SetExecutorOwner(colonyName string, executorName string, ownerID string, ownerName string) error
	// GetExecutorOwner returns the current owner, or nil if none was ever set.
	GetExecutorOwner(colonyName string, executorName string) (*core.ExecutorOwner, error)
}
