package postgresql

import (
	"testing"
	"time"

	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func shortID(id string) string {
	if len(id) < 4 {
		return id
	}
	return id[:4]
}

func createTestUsageRecord(colonyName string, initiatorID string, providerID string, cpu int64, gpuCount int) *core.UsageRecord {
	return &core.UsageRecord{
		ColonyName:    colonyName,
		ProcessID:     core.GenerateRandomID(),
		InitiatorID:   initiatorID,
		InitiatorName: "user-" + shortID(initiatorID),
		ExecutorID:    core.GenerateRandomID(),
		ExecutorName:  "executor-" + shortID(providerID),
		ExecutorType:  "test_executor_type",
		ProviderID:    providerID,
		ProviderName:  "user-" + shortID(providerID),
		Label:         "test_label",
		FuncName:      "testfunc",
		CPU:           cpu,
		Memory:        1024 * 1024,
		GPUCount:      gpuCount,
		Hardware:      []core.Hardware{{Model: "test-model", Cores: 8}},
	}
}

func TestOpenAndCloseUsageRecord(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)
	defer db.Close()

	colony, _, err := utils.CreateTestColonyWithKey()
	assert.Nil(t, err)

	initiatorID := core.GenerateRandomID()
	providerID := core.GenerateRandomID()
	record := createTestUsageRecord(colony.Name, initiatorID, providerID, 2000, 1)

	err = db.OpenUsageRecord(record)
	assert.Nil(t, err)
	assert.NotEmpty(t, record.ID)

	records, err := db.GetUsageRecordsByProcessID(record.ProcessID)
	assert.Nil(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, core.USAGE_RUNNING, records[0].Outcome)
	assert.True(t, records[0].EndTime.IsZero())
	assert.Equal(t, int64(2000), records[0].CPU)
	assert.Equal(t, providerID, records[0].ProviderID)
	assert.Len(t, records[0].Hardware, 1)
	assert.Equal(t, "test-model", records[0].Hardware[0].Model)

	endTime := time.Now().Add(10 * time.Second)
	err = db.CloseUsageRecord(record.ProcessID, endTime, core.USAGE_SUCCESS)
	assert.Nil(t, err)

	records, err = db.GetUsageRecordsByProcessID(record.ProcessID)
	assert.Nil(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, core.USAGE_SUCCESS, records[0].Outcome)
	assert.False(t, records[0].EndTime.IsZero())

	// Closing again must not change the outcome
	err = db.CloseUsageRecord(record.ProcessID, time.Now(), core.USAGE_FAILED)
	assert.Nil(t, err)
	records, err = db.GetUsageRecordsByProcessID(record.ProcessID)
	assert.Nil(t, err)
	assert.Equal(t, core.USAGE_SUCCESS, records[0].Outcome)
}

func TestUsageRecordPerAttempt(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)
	defer db.Close()

	colony, _, err := utils.CreateTestColonyWithKey()
	assert.Nil(t, err)

	initiatorID := core.GenerateRandomID()
	first := createTestUsageRecord(colony.Name, initiatorID, core.GenerateRandomID(), 1000, 0)
	err = db.OpenUsageRecord(first)
	assert.Nil(t, err)
	err = db.CloseUsageRecord(first.ProcessID, time.Now(), core.USAGE_UNASSIGNED)
	assert.Nil(t, err)

	second := createTestUsageRecord(colony.Name, initiatorID, core.GenerateRandomID(), 1000, 0)
	second.ProcessID = first.ProcessID
	second.Attempt = 1
	err = db.OpenUsageRecord(second)
	assert.Nil(t, err)

	records, err := db.GetUsageRecordsByProcessID(first.ProcessID)
	assert.Nil(t, err)
	assert.Len(t, records, 2)
	assert.Equal(t, core.USAGE_UNASSIGNED, records[0].Outcome)
	assert.Equal(t, core.USAGE_RUNNING, records[1].Outcome)
	assert.NotEqual(t, records[0].ExecutorID, records[1].ExecutorID)
}

func TestCloseUsageRecordsByExecutor(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)
	defer db.Close()

	colony, _, err := utils.CreateTestColonyWithKey()
	assert.Nil(t, err)

	providerID := core.GenerateRandomID()
	r1 := createTestUsageRecord(colony.Name, core.GenerateRandomID(), providerID, 1000, 0)
	r2 := createTestUsageRecord(colony.Name, core.GenerateRandomID(), providerID, 1000, 0)
	other := createTestUsageRecord(colony.Name, core.GenerateRandomID(), core.GenerateRandomID(), 1000, 0)
	assert.Nil(t, db.OpenUsageRecord(r1))
	assert.Nil(t, db.OpenUsageRecord(r2))
	assert.Nil(t, db.OpenUsageRecord(other))

	err = db.CloseUsageRecordsByExecutor(colony.Name, r1.ExecutorName, time.Now(), core.USAGE_EXECUTOR_LOST)
	assert.Nil(t, err)

	for _, pid := range []string{r1.ProcessID, r2.ProcessID} {
		records, err := db.GetUsageRecordsByProcessID(pid)
		assert.Nil(t, err)
		assert.Equal(t, core.USAGE_EXECUTOR_LOST, records[0].Outcome)
	}
	records, err := db.GetUsageRecordsByProcessID(other.ProcessID)
	assert.Nil(t, err)
	assert.Equal(t, core.USAGE_RUNNING, records[0].Outcome)
}

func TestSummarizeUsage(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)
	defer db.Close()

	colony, _, err := utils.CreateTestColonyWithKey()
	assert.Nil(t, err)

	alice := core.GenerateRandomID()
	bob := core.GenerateRandomID()
	start := time.Now().Add(-1 * time.Hour)

	// alice runs two 60 s attempts with 2 cores each on bob's executor
	for i := 0; i < 2; i++ {
		r := createTestUsageRecord(colony.Name, alice, bob, 2000, 0)
		r.StartTime = start
		assert.Nil(t, db.OpenUsageRecord(r))
		assert.Nil(t, db.CloseUsageRecord(r.ProcessID, start.Add(60*time.Second), core.USAGE_SUCCESS))
	}
	// bob runs one 30 s attempt with 1 GPU on alice's executor, and it fails
	r := createTestUsageRecord(colony.Name, bob, alice, 1000, 1)
	r.StartTime = start
	assert.Nil(t, db.OpenUsageRecord(r))
	assert.Nil(t, db.CloseUsageRecord(r.ProcessID, start.Add(30*time.Second), core.USAGE_FAILED))

	from := start.Add(-time.Minute)
	to := time.Now()

	byInitiator, err := db.SummarizeUsage(colony.Name, from, to, core.USAGE_GROUP_INITIATOR, "", "")
	assert.Nil(t, err)
	assert.Len(t, byInitiator, 2)
	assert.Equal(t, alice, byInitiator[0].Key) // sorted by core seconds desc
	assert.Equal(t, 2, byInitiator[0].Attempts)
	assert.Equal(t, 2, byInitiator[0].Succeeded)
	assert.InDelta(t, 120.0, byInitiator[0].WallSeconds, 0.01)
	assert.InDelta(t, 240.0, byInitiator[0].CoreSeconds, 0.01)
	assert.Equal(t, bob, byInitiator[1].Key)
	assert.Equal(t, 1, byInitiator[1].Failed)
	assert.InDelta(t, 30.0, byInitiator[1].CoreSeconds, 0.01)
	assert.InDelta(t, 30.0, byInitiator[1].GPUSeconds, 0.01)

	byProvider, err := db.SummarizeUsage(colony.Name, from, to, core.USAGE_GROUP_PROVIDER, "", "")
	assert.Nil(t, err)
	assert.Len(t, byProvider, 2)
	assert.Equal(t, bob, byProvider[0].Key)
	assert.InDelta(t, 240.0, byProvider[0].CoreSeconds, 0.01)

	// Filter to what ran on bob's machines, grouped by who ran it
	onBob, err := db.SummarizeUsage(colony.Name, from, to, core.USAGE_GROUP_INITIATOR, "", bob)
	assert.Nil(t, err)
	assert.Len(t, onBob, 1)
	assert.Equal(t, alice, onBob[0].Key)

	// Window excludes everything
	none, err := db.SummarizeUsage(colony.Name, to.Add(time.Hour), to.Add(2*time.Hour), core.USAGE_GROUP_INITIATOR, "", "")
	assert.Nil(t, err)
	assert.Len(t, none, 0)

	_, err = db.SummarizeUsage(colony.Name, from, to, "DROP TABLE", "", "")
	assert.NotNil(t, err)
}

func TestUsageBalance(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)
	defer db.Close()

	colony, _, err := utils.CreateTestColonyWithKey()
	assert.Nil(t, err)

	alice := core.GenerateRandomID()
	bob := core.GenerateRandomID()
	start := time.Now().Add(-1 * time.Hour)

	open := func(initiator string, provider string, cpu int64, gpu int, seconds int) {
		r := createTestUsageRecord(colony.Name, initiator, provider, cpu, gpu)
		if provider == "" {
			r.ProviderName = ""
		}
		r.StartTime = start
		assert.Nil(t, db.OpenUsageRecord(r))
		assert.Nil(t, db.CloseUsageRecord(r.ProcessID, start.Add(time.Duration(seconds)*time.Second), core.USAGE_SUCCESS))
	}

	open(alice, bob, 1000, 0, 100)  // alice consumes 100 core s from bob
	open(bob, alice, 1000, 1, 40)   // bob consumes 40 core s and 40 gpu s from alice
	open(alice, alice, 1000, 0, 10) // alice self-use
	open(bob, "", 2000, 0, 50)      // bob uses a colony-owned machine

	balances, err := db.UsageBalance(colony.Name, start.Add(-time.Minute), time.Now())
	assert.Nil(t, err)
	assert.Len(t, balances, 3)

	byID := map[string]*core.UsageBalance{}
	for _, b := range balances {
		byID[b.UserID] = b
	}

	assert.InDelta(t, 100.0, byID[alice].ConsumedCoreSeconds, 0.01)
	assert.InDelta(t, 40.0, byID[alice].ProvidedCoreSeconds, 0.01)
	assert.InDelta(t, 10.0, byID[alice].SelfCoreSeconds, 0.01)
	assert.InDelta(t, 40.0, byID[alice].ProvidedGPUSeconds, 0.01)
	assert.InDelta(t, -60.0, byID[alice].NetCoreSeconds(), 0.01)

	assert.InDelta(t, 140.0, byID[bob].ConsumedCoreSeconds, 0.01)
	assert.InDelta(t, 100.0, byID[bob].ProvidedCoreSeconds, 0.01)
	assert.InDelta(t, 40.0, byID[bob].ConsumedGPUSeconds, 0.01)

	assert.Equal(t, "colony", byID[""].UserName)
	assert.InDelta(t, 100.0, byID[""].ProvidedCoreSeconds, 0.01)
}

func TestExecutorOwner(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)
	defer db.Close()

	colony, _, err := utils.CreateTestColonyWithKey()
	assert.Nil(t, err)

	owner, err := db.GetExecutorOwner(colony.Name, "executor1")
	assert.Nil(t, err)
	assert.Nil(t, owner)

	aliceID := core.GenerateRandomID()
	err = db.SetExecutorOwner(colony.Name, "executor1", aliceID, "alice")
	assert.Nil(t, err)

	owner, err = db.GetExecutorOwner(colony.Name, "executor1")
	assert.Nil(t, err)
	assert.NotNil(t, owner)
	assert.Equal(t, aliceID, owner.OwnerID)
	assert.Equal(t, "alice", owner.OwnerName)

	bobID := core.GenerateRandomID()
	err = db.SetExecutorOwner(colony.Name, "executor1", bobID, "bob")
	assert.Nil(t, err)

	owner, err = db.GetExecutorOwner(colony.Name, "executor1")
	assert.Nil(t, err)
	assert.Equal(t, bobID, owner.OwnerID)

	// Clearing the owner hands the executor back to the colony
	err = db.SetExecutorOwner(colony.Name, "executor1", "", "")
	assert.Nil(t, err)
	owner, err = db.GetExecutorOwner(colony.Name, "executor1")
	assert.Nil(t, err)
	assert.NotNil(t, owner)
	assert.Equal(t, "", owner.OwnerID)
}

func TestRemoveExecutorClosesUsageRecords(t *testing.T) {
	db, err := PrepareTests()
	assert.Nil(t, err)
	defer db.Close()

	colony, _, err := utils.CreateTestColonyWithKey()
	assert.Nil(t, err)
	assert.Nil(t, db.AddColony(colony))

	executor := utils.CreateTestExecutor(colony.Name)
	assert.Nil(t, db.AddExecutor(executor))

	record := createTestUsageRecord(colony.Name, core.GenerateRandomID(), core.GenerateRandomID(), 1000, 0)
	record.ExecutorID = executor.ID
	record.ExecutorName = executor.Name
	assert.Nil(t, db.OpenUsageRecord(record))

	err = db.RemoveExecutorByName(colony.Name, executor.Name)
	assert.Nil(t, err)

	records, err := db.GetUsageRecordsByProcessID(record.ProcessID)
	assert.Nil(t, err)
	assert.Equal(t, core.USAGE_EXECUTOR_LOST, records[0].Outcome)
}
