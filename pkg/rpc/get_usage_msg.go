package rpc

import (
	"encoding/json"
)

const GetUsagePayloadType = "getusagemsg"

// Modes of a usage query.
const (
	UsageModeSummary = "summary"
	UsageModeRecords = "records"
	UsageModeBalance = "balance"
)

// GetUsageMsg asks the server for usage in [From, To] (unix seconds).
// Mode selects summaries grouped by GroupBy, raw records, or the per-member
// balance. InitiatorID, ProviderID and ExecutorName are optional filters.
type GetUsageMsg struct {
	MsgType      string `json:"msgtype"`
	ColonyName   string `json:"colonyname"`
	Mode         string `json:"mode"`
	From         int64  `json:"from"`
	To           int64  `json:"to"`
	GroupBy      string `json:"groupby"`
	InitiatorID  string `json:"initiatorid"`
	ProviderID   string `json:"providerid"`
	ExecutorName string `json:"executorname"`
	Count        int    `json:"count"`
}

func CreateGetUsageMsg(colonyName string, mode string, from int64, to int64) *GetUsageMsg {
	msg := &GetUsageMsg{}
	msg.MsgType = GetUsagePayloadType
	msg.ColonyName = colonyName
	msg.Mode = mode
	msg.From = from
	msg.To = to

	return msg
}

func (msg *GetUsageMsg) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *GetUsageMsg) ToJSONIndent() (string, error) {
	jsonBytes, err := json.MarshalIndent(msg, "", "    ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *GetUsageMsg) Equals(msg2 *GetUsageMsg) bool {
	if msg2 == nil {
		return false
	}

	return msg.MsgType == msg2.MsgType &&
		msg.ColonyName == msg2.ColonyName &&
		msg.Mode == msg2.Mode &&
		msg.From == msg2.From &&
		msg.To == msg2.To &&
		msg.GroupBy == msg2.GroupBy &&
		msg.InitiatorID == msg2.InitiatorID &&
		msg.ProviderID == msg2.ProviderID &&
		msg.ExecutorName == msg2.ExecutorName &&
		msg.Count == msg2.Count
}

func CreateGetUsageMsgFromJSON(jsonString string) (*GetUsageMsg, error) {
	var msg *GetUsageMsg

	err := json.Unmarshal([]byte(jsonString), &msg)
	if err != nil {
		return msg, err
	}

	return msg, nil
}
