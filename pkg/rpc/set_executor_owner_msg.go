package rpc

import (
	"encoding/json"
)

const SetExecutorOwnerPayloadType = "setexecutorownermsg"

// SetExecutorOwnerMsg assigns an executor to a colony user. An empty
// OwnerName hands the executor back to the colony.
type SetExecutorOwnerMsg struct {
	MsgType      string `json:"msgtype"`
	ColonyName   string `json:"colonyname"`
	ExecutorName string `json:"executorname"`
	OwnerName    string `json:"ownername"`
}

func CreateSetExecutorOwnerMsg(colonyName string, executorName string, ownerName string) *SetExecutorOwnerMsg {
	msg := &SetExecutorOwnerMsg{}
	msg.MsgType = SetExecutorOwnerPayloadType
	msg.ColonyName = colonyName
	msg.ExecutorName = executorName
	msg.OwnerName = ownerName

	return msg
}

func (msg *SetExecutorOwnerMsg) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *SetExecutorOwnerMsg) ToJSONIndent() (string, error) {
	jsonBytes, err := json.MarshalIndent(msg, "", "    ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *SetExecutorOwnerMsg) Equals(msg2 *SetExecutorOwnerMsg) bool {
	if msg2 == nil {
		return false
	}

	return msg.MsgType == msg2.MsgType &&
		msg.ColonyName == msg2.ColonyName &&
		msg.ExecutorName == msg2.ExecutorName &&
		msg.OwnerName == msg2.OwnerName
}

func CreateSetExecutorOwnerMsgFromJSON(jsonString string) (*SetExecutorOwnerMsg, error) {
	var msg *SetExecutorOwnerMsg

	err := json.Unmarshal([]byte(jsonString), &msg)
	if err != nil {
		return msg, err
	}

	return msg, nil
}
