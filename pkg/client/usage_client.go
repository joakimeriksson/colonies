package client

import (
	"context"

	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/rpc"
)

// GetUsageSummary returns usage in [from, to] (unix seconds, 0 for defaults)
// aggregated by one of the core.USAGE_GROUP_* keys. initiatorID and
// providerID are optional filters.
func (client *ColoniesClient) GetUsageSummary(colonyName string, from int64, to int64, groupBy string, initiatorID string, providerID string, prvKey string) ([]*core.UsageSummary, error) {
	msg := rpc.CreateGetUsageMsg(colonyName, rpc.UsageModeSummary, from, to)
	msg.GroupBy = groupBy
	msg.InitiatorID = initiatorID
	msg.ProviderID = providerID
	jsonString, err := msg.ToJSON()
	if err != nil {
		return nil, err
	}

	respBodyString, err := client.sendMessage(rpc.GetUsagePayloadType, jsonString, prvKey, false, context.TODO())
	if err != nil {
		return nil, err
	}

	return core.ConvertJSONToUsageSummaryArray(respBodyString)
}

// GetUsageRecords returns the raw per-attempt records in [from, to].
func (client *ColoniesClient) GetUsageRecords(colonyName string, from int64, to int64, initiatorID string, providerID string, executorName string, count int, prvKey string) ([]*core.UsageRecord, error) {
	msg := rpc.CreateGetUsageMsg(colonyName, rpc.UsageModeRecords, from, to)
	msg.InitiatorID = initiatorID
	msg.ProviderID = providerID
	msg.ExecutorName = executorName
	msg.Count = count
	jsonString, err := msg.ToJSON()
	if err != nil {
		return nil, err
	}

	respBodyString, err := client.sendMessage(rpc.GetUsagePayloadType, jsonString, prvKey, false, context.TODO())
	if err != nil {
		return nil, err
	}

	return core.ConvertJSONToUsageRecordArray(respBodyString)
}

// GetUsageBalance returns consumed, provided and self-use per colony member.
func (client *ColoniesClient) GetUsageBalance(colonyName string, from int64, to int64, prvKey string) ([]*core.UsageBalance, error) {
	msg := rpc.CreateGetUsageMsg(colonyName, rpc.UsageModeBalance, from, to)
	jsonString, err := msg.ToJSON()
	if err != nil {
		return nil, err
	}

	respBodyString, err := client.sendMessage(rpc.GetUsagePayloadType, jsonString, prvKey, false, context.TODO())
	if err != nil {
		return nil, err
	}

	return core.ConvertJSONToUsageBalanceArray(respBodyString)
}

// SetExecutorOwner assigns an executor to a colony user by name. An empty
// ownerName hands it back to the colony. Requires the colony private key.
func (client *ColoniesClient) SetExecutorOwner(colonyName string, executorName string, ownerName string, prvKey string) (*core.ExecutorOwner, error) {
	msg := rpc.CreateSetExecutorOwnerMsg(colonyName, executorName, ownerName)
	jsonString, err := msg.ToJSON()
	if err != nil {
		return nil, err
	}

	respBodyString, err := client.sendMessage(rpc.SetExecutorOwnerPayloadType, jsonString, prvKey, false, context.TODO())
	if err != nil {
		return nil, err
	}

	return core.ConvertJSONToExecutorOwner(respBodyString)
}

func (client *ColoniesClient) GetExecutorOwner(colonyName string, executorName string, prvKey string) (*core.ExecutorOwner, error) {
	msg := rpc.CreateGetExecutorOwnerMsg(colonyName, executorName)
	jsonString, err := msg.ToJSON()
	if err != nil {
		return nil, err
	}

	respBodyString, err := client.sendMessage(rpc.GetExecutorOwnerPayloadType, jsonString, prvKey, false, context.TODO())
	if err != nil {
		return nil, err
	}

	return core.ConvertJSONToExecutorOwner(respBodyString)
}
