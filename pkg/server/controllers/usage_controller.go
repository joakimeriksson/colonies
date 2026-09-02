package controllers

import (
	"time"

	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/parsers"
	log "github.com/sirupsen/logrus"
)

// openUsageRecord writes one ledger row for the attempt that just started.
// Failures are logged, never returned: the assignment has already happened
// and must not be rolled back because bookkeeping failed.
func (controller *ColoniesController) openUsageRecord(process *core.Process, executor *core.Executor) {
	if controller.usageDB == nil || process == nil || executor == nil {
		return
	}

	conditions := process.FunctionSpec.Conditions

	cpu, err := parsers.ConvertCPUToInt(conditions.CPU)
	if err != nil {
		cpu = 0
	}
	memory, err := parsers.ConvertMemoryToBytes(conditions.Memory)
	if err != nil {
		memory = 0
	}
	storage, err := parsers.ConvertMemoryToBytes(conditions.Storage)
	if err != nil {
		storage = 0
	}
	gpuMemory, err := parsers.ConvertMemoryToBytes(conditions.GPU.Memory)
	if err != nil {
		gpuMemory = 0
	}

	record := &core.UsageRecord{
		ColonyName:     conditions.ColonyName,
		ProcessID:      process.ID,
		Attempt:        process.Retries,
		ProcessGraphID: process.ProcessGraphID,
		InitiatorID:    process.InitiatorID,
		InitiatorName:  process.InitiatorName,
		Label:          process.FunctionSpec.Label,
		FuncName:       process.FunctionSpec.FuncName,
		ExecutorID:     executor.ID,
		ExecutorName:   executor.Name,
		ExecutorType:   executor.Type,
		LocationName:   executor.LocationName,
		Hardware:       executor.Capabilities.Hardware,
		CPU:            cpu,
		Memory:         memory,
		Storage:        storage,
		GPUName:        conditions.GPU.Name,
		GPUCount:       conditions.GPU.Count,
		GPUMemory:      gpuMemory,
		Nodes:          conditions.Nodes,
		StartTime:      time.Now(),
	}

	owner, err := controller.usageDB.GetExecutorOwner(executor.ColonyName, executor.Name)
	if err != nil {
		log.WithFields(log.Fields{"ExecutorName": executor.Name, "Error": err}).Warn("Failed to look up executor owner for usage record")
	} else if owner != nil {
		record.ProviderID = owner.OwnerID
		record.ProviderName = owner.OwnerName
	}

	err = controller.usageDB.OpenUsageRecord(record)
	if err != nil {
		log.WithFields(log.Fields{"ProcessId": process.ID, "ExecutorName": executor.Name, "Error": err}).Error("Failed to open usage record")
	}
}

// closeUsageRecord closes the open ledger row for a process. Closing a
// process that has no open row is a no-op, so callers may close on every
// terminal path without checking state first.
func (controller *ColoniesController) closeUsageRecord(processID string, outcome string) {
	if controller.usageDB == nil {
		return
	}

	err := controller.usageDB.CloseUsageRecord(processID, time.Now(), outcome)
	if err != nil {
		log.WithFields(log.Fields{"ProcessId": processID, "Outcome": outcome, "Error": err}).Error("Failed to close usage record")
	}
}
