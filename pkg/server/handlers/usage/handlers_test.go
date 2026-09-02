package usage_test

import (
	"testing"
	"time"

	"github.com/colonyos/colonies/pkg/client"
	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/server"
	"github.com/colonyos/colonies/pkg/utils"
	"github.com/stretchr/testify/assert"
)

type usageTestEnv struct {
	colony          *core.Colony
	colonyPrvKey    string
	alice           *core.User
	alicePrvKey     string
	bob             *core.User
	bobPrvKey       string
	executor        *core.Executor
	executorPrvKey  string
	otherExecutor   *core.Executor
	otherExecPrvKey string
}

// setupUsageEnv builds a colony with two users and one approved executor
// owned by alice.
func setupUsageEnv(t *testing.T, c *client.ColoniesClient, serverPrvKey string) *usageTestEnv {
	env := &usageTestEnv{}
	var err error

	env.colony, env.colonyPrvKey, err = utils.CreateTestColonyWithKey()
	assert.Nil(t, err)
	_, err = c.AddColony(env.colony, serverPrvKey)
	assert.Nil(t, err)

	env.alice, env.alicePrvKey, err = utils.CreateTestUserWithKey(env.colony.Name, "alice")
	assert.Nil(t, err)
	_, err = c.AddUser(env.alice, env.colonyPrvKey)
	assert.Nil(t, err)

	env.bob, env.bobPrvKey, err = utils.CreateTestUserWithKey(env.colony.Name, "bob")
	assert.Nil(t, err)
	_, err = c.AddUser(env.bob, env.colonyPrvKey)
	assert.Nil(t, err)

	env.executor, env.executorPrvKey, err = utils.CreateTestExecutorWithKey(env.colony.Name)
	assert.Nil(t, err)
	_, err = c.AddExecutor(env.executor, env.colonyPrvKey)
	assert.Nil(t, err)
	err = c.ApproveExecutor(env.colony.Name, env.executor.Name, env.colonyPrvKey)
	assert.Nil(t, err)

	owner, err := c.SetExecutorOwner(env.colony.Name, env.executor.Name, "alice", env.colonyPrvKey)
	assert.Nil(t, err)
	assert.Equal(t, env.alice.ID, owner.OwnerID)
	assert.Equal(t, "alice", owner.OwnerName)

	return env
}

func TestExecutorOwner(t *testing.T) {
	c, s, serverPrvKey, done := server.PrepareTests(t)
	env := setupUsageEnv(t, c, serverPrvKey)

	// Any member can read the owner
	owner, err := c.GetExecutorOwner(env.colony.Name, env.executor.Name, env.bobPrvKey)
	assert.Nil(t, err)
	assert.Equal(t, env.alice.ID, owner.OwnerID)

	// Only the colony owner can set it
	_, err = c.SetExecutorOwner(env.colony.Name, env.executor.Name, "bob", env.alicePrvKey)
	assert.NotNil(t, err)

	// Unknown user is rejected
	_, err = c.SetExecutorOwner(env.colony.Name, env.executor.Name, "nobody", env.colonyPrvKey)
	assert.NotNil(t, err)

	// Unknown executor is rejected
	_, err = c.SetExecutorOwner(env.colony.Name, "no-such-executor", "alice", env.colonyPrvKey)
	assert.NotNil(t, err)

	// Handing back to the colony
	owner, err = c.SetExecutorOwner(env.colony.Name, env.executor.Name, "", env.colonyPrvKey)
	assert.Nil(t, err)
	assert.Equal(t, "", owner.OwnerID)

	// An executor whose owner was never set reads as colony-owned
	other, _, err := utils.CreateTestExecutorWithKey(env.colony.Name)
	assert.Nil(t, err)
	_, err = c.AddExecutor(other, env.colonyPrvKey)
	assert.Nil(t, err)
	owner, err = c.GetExecutorOwner(env.colony.Name, other.Name, env.bobPrvKey)
	assert.Nil(t, err)
	assert.Equal(t, "", owner.OwnerID)
	assert.Equal(t, other.Name, owner.ExecutorName)

	s.Shutdown()
	<-done
}

func TestUsageLifecycle(t *testing.T) {
	c, s, serverPrvKey, done := server.PrepareTests(t)
	env := setupUsageEnv(t, c, serverPrvKey)

	// bob submits a process asking for 2 cores and 1 GPU
	funcSpec := utils.CreateTestFunctionSpec(env.colony.Name)
	funcSpec.Conditions.CPU = "2000m"
	funcSpec.Conditions.Memory = "512Mi"
	funcSpec.Conditions.GPU = core.GPU{Name: "test-gpu", Count: 1}
	submitted, err := c.Submit(funcSpec, env.bobPrvKey)
	assert.Nil(t, err)

	// alice's executor picks it up
	assigned, err := c.Assign(env.colony.Name, 10, "", "", env.executorPrvKey)
	assert.Nil(t, err)
	assert.Equal(t, submitted.ID, assigned.ID)

	records, err := c.GetUsageRecords(env.colony.Name, 0, 0, "", "", "", 10, env.colonyPrvKey)
	assert.Nil(t, err)
	assert.Len(t, records, 1)
	r := records[0]
	assert.Equal(t, submitted.ID, r.ProcessID)
	assert.Equal(t, 0, r.Attempt)
	assert.Equal(t, core.USAGE_RUNNING, r.Outcome)
	assert.Equal(t, env.bob.ID, r.InitiatorID)
	assert.Equal(t, "bob", r.InitiatorName)
	assert.Equal(t, env.executor.ID, r.ExecutorID)
	assert.Equal(t, env.executor.Name, r.ExecutorName)
	assert.Equal(t, env.alice.ID, r.ProviderID)
	assert.Equal(t, "alice", r.ProviderName)
	assert.Equal(t, int64(2000), r.CPU)
	assert.Equal(t, int64(512*1024*1024), r.Memory)
	assert.Equal(t, 1, r.GPUCount)
	assert.Equal(t, "test_label", r.Label)
	assert.Equal(t, "test_func", r.FuncName)
	assert.True(t, r.EndTime.IsZero())

	time.Sleep(1100 * time.Millisecond)
	err = c.Close(assigned.ID, env.executorPrvKey)
	assert.Nil(t, err)

	records, err = c.GetUsageRecords(env.colony.Name, 0, 0, "", "", "", 10, env.colonyPrvKey)
	assert.Nil(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, core.USAGE_SUCCESS, records[0].Outcome)
	assert.False(t, records[0].EndTime.IsZero())
	assert.True(t, records[0].WallTime() >= time.Second)

	// Consumer view: grouped by initiator
	byInitiator, err := c.GetUsageSummary(env.colony.Name, 0, 0, core.USAGE_GROUP_INITIATOR, "", "", env.colonyPrvKey)
	assert.Nil(t, err)
	assert.Len(t, byInitiator, 1)
	assert.Equal(t, env.bob.ID, byInitiator[0].Key)
	assert.Equal(t, "bob", byInitiator[0].Name)
	assert.Equal(t, 1, byInitiator[0].Attempts)
	assert.Equal(t, 1, byInitiator[0].Succeeded)
	assert.True(t, byInitiator[0].WallSeconds >= 1.0)
	assert.InDelta(t, byInitiator[0].WallSeconds*2, byInitiator[0].CoreSeconds, 0.001)
	assert.InDelta(t, byInitiator[0].WallSeconds, byInitiator[0].GPUSeconds, 0.001)

	// Provider view: grouped by provider
	byProvider, err := c.GetUsageSummary(env.colony.Name, 0, 0, core.USAGE_GROUP_PROVIDER, "", "", env.colonyPrvKey)
	assert.Nil(t, err)
	assert.Len(t, byProvider, 1)
	assert.Equal(t, env.alice.ID, byProvider[0].Key)
	assert.Equal(t, "alice", byProvider[0].Name)

	// Balance: bob consumed what alice provided
	balances, err := c.GetUsageBalance(env.colony.Name, 0, 0, env.bobPrvKey)
	assert.Nil(t, err)
	assert.Len(t, balances, 2)
	byID := map[string]*core.UsageBalance{}
	for _, b := range balances {
		byID[b.UserID] = b
	}
	assert.True(t, byID[env.bob.ID].ConsumedCoreSeconds > 0)
	assert.Equal(t, 0.0, byID[env.bob.ID].ProvidedCoreSeconds)
	assert.InDelta(t, byID[env.bob.ID].ConsumedCoreSeconds, byID[env.alice.ID].ProvidedCoreSeconds, 0.001)
	assert.True(t, byID[env.alice.ID].NetCoreSeconds() > 0)
	assert.True(t, byID[env.bob.ID].NetCoreSeconds() < 0)

	s.Shutdown()
	<-done
}

func TestUsageVisibility(t *testing.T) {
	c, s, serverPrvKey, done := server.PrepareTests(t)
	env := setupUsageEnv(t, c, serverPrvKey)

	funcSpec := utils.CreateTestFunctionSpec(env.colony.Name)
	funcSpec.Conditions.CPU = "1000m"
	_, err := c.Submit(funcSpec, env.bobPrvKey)
	assert.Nil(t, err)
	assigned, err := c.Assign(env.colony.Name, 10, "", "", env.executorPrvKey)
	assert.Nil(t, err)
	err = c.Close(assigned.ID, env.executorPrvKey)
	assert.Nil(t, err)

	// bob sees what he ran
	mine, err := c.GetUsageSummary(env.colony.Name, 0, 0, core.USAGE_GROUP_EXECUTOR, "", "", env.bobPrvKey)
	assert.Nil(t, err)
	assert.Len(t, mine, 1)
	assert.Equal(t, env.executor.Name, mine[0].Name)

	// alice ran nothing, so the plain view is empty for her
	nothing, err := c.GetUsageSummary(env.colony.Name, 0, 0, core.USAGE_GROUP_INITIATOR, "", "", env.alicePrvKey)
	assert.Nil(t, err)
	assert.Len(t, nothing, 0)

	// but as provider she sees who ran on her machine
	onMine, err := c.GetUsageSummary(env.colony.Name, 0, 0, core.USAGE_GROUP_INITIATOR, "", env.alice.ID, env.alicePrvKey)
	assert.Nil(t, err)
	assert.Len(t, onMine, 1)
	assert.Equal(t, "bob", onMine[0].Name)

	// bob cannot read the provider view for alice's machines; the server
	// narrows it to his own runs, which on her machine is still his one run
	onAliceAsBob, err := c.GetUsageRecords(env.colony.Name, 0, 0, "", env.alice.ID, "", 10, env.bobPrvKey)
	assert.Nil(t, err)
	assert.Len(t, onAliceAsBob, 1)
	assert.Equal(t, env.bob.ID, onAliceAsBob[0].InitiatorID)

	// a non-member is refused
	_, strangerPrvKey, err := utils.CreateTestUserWithKey(env.colony.Name, "stranger")
	assert.Nil(t, err)
	_, err = c.GetUsageSummary(env.colony.Name, 0, 0, core.USAGE_GROUP_INITIATOR, "", "", strangerPrvKey)
	assert.NotNil(t, err)

	// a bad group key is refused
	_, err = c.GetUsageSummary(env.colony.Name, 0, 0, "bogus", "", "", env.colonyPrvKey)
	assert.NotNil(t, err)

	s.Shutdown()
	<-done
}

func TestUsageFailedAndRetried(t *testing.T) {
	c, s, serverPrvKey, done := server.PrepareTests(t)
	env := setupUsageEnv(t, c, serverPrvKey)

	// Failed run
	funcSpec := utils.CreateTestFunctionSpec(env.colony.Name)
	_, err := c.Submit(funcSpec, env.bobPrvKey)
	assert.Nil(t, err)
	assigned, err := c.Assign(env.colony.Name, 10, "", "", env.executorPrvKey)
	assert.Nil(t, err)
	err = c.Fail(assigned.ID, []string{"boom"}, env.executorPrvKey)
	assert.Nil(t, err)

	records, err := c.GetUsageRecords(env.colony.Name, 0, 0, "", "", "", 10, env.colonyPrvKey)
	assert.Nil(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, core.USAGE_FAILED, records[0].Outcome)

	// Timed-out run that is retried on a second executor: two records
	other, otherPrvKey, err := utils.CreateTestExecutorWithKey(env.colony.Name)
	assert.Nil(t, err)
	_, err = c.AddExecutor(other, env.colonyPrvKey)
	assert.Nil(t, err)
	err = c.ApproveExecutor(env.colony.Name, other.Name, env.colonyPrvKey)
	assert.Nil(t, err)

	funcSpec = utils.CreateTestFunctionSpec(env.colony.Name)
	funcSpec.MaxExecTime = 1
	funcSpec.MaxRetries = 1
	retried, err := c.Submit(funcSpec, env.bobPrvKey)
	assert.Nil(t, err)
	first, err := c.Assign(env.colony.Name, 10, "", "", env.executorPrvKey)
	assert.Nil(t, err)
	assert.Equal(t, retried.ID, first.ID)

	// Wait for the timeout loop to unassign it
	deadline := time.Now().Add(10 * time.Second)
	for {
		p, err := c.GetProcess(retried.ID, env.bobPrvKey)
		assert.Nil(t, err)
		if p.State == core.WAITING || time.Now().After(deadline) {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	second, err := c.Assign(env.colony.Name, 10, "", "", otherPrvKey)
	assert.Nil(t, err)
	assert.Equal(t, retried.ID, second.ID)
	err = c.Close(second.ID, otherPrvKey)
	assert.Nil(t, err)

	records, err = c.GetUsageRecords(env.colony.Name, 0, 0, "", "", "", 10, env.colonyPrvKey)
	assert.Nil(t, err)
	var attempts []*core.UsageRecord
	for _, r := range records {
		if r.ProcessID == retried.ID {
			attempts = append(attempts, r)
		}
	}
	assert.Len(t, attempts, 2)
	// Records come back newest first
	assert.Equal(t, 1, attempts[0].Attempt)
	assert.Equal(t, core.USAGE_SUCCESS, attempts[0].Outcome)
	assert.Equal(t, other.Name, attempts[0].ExecutorName)
	assert.Equal(t, 0, attempts[1].Attempt)
	assert.Equal(t, core.USAGE_TIMEOUT, attempts[1].Outcome)
	assert.Equal(t, env.executor.Name, attempts[1].ExecutorName)
	assert.Equal(t, env.alice.ID, attempts[1].ProviderID)

	s.Shutdown()
	<-done
}

func TestUsageExecutorRemoved(t *testing.T) {
	c, s, serverPrvKey, done := server.PrepareTests(t)
	env := setupUsageEnv(t, c, serverPrvKey)

	funcSpec := utils.CreateTestFunctionSpec(env.colony.Name)
	_, err := c.Submit(funcSpec, env.bobPrvKey)
	assert.Nil(t, err)
	assigned, err := c.Assign(env.colony.Name, 10, "", "", env.executorPrvKey)
	assert.Nil(t, err)

	err = c.RemoveExecutor(env.colony.Name, env.executor.Name, env.colonyPrvKey)
	assert.Nil(t, err)

	records, err := c.GetUsageRecords(env.colony.Name, 0, 0, "", "", env.executor.Name, 10, env.colonyPrvKey)
	assert.Nil(t, err)
	assert.Len(t, records, 1)
	assert.Equal(t, assigned.ID, records[0].ProcessID)
	assert.Equal(t, core.USAGE_EXECUTOR_LOST, records[0].Outcome)

	s.Shutdown()
	<-done
}
