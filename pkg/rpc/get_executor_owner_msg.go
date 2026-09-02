package rpc

import (
	"encoding/json"
)

const GetExecutorOwnerPayloadType = "getexecutorownermsg"

type GetExecutorOwnerMsg struct {
	MsgType      string `json:"msgtype"`
	ColonyName   string `json:"colonyname"`
	ExecutorName string `json:"executorname"`
}

func CreateGetExecutorOwnerMsg(colonyName string, executorName string) *GetExecutorOwnerMsg {
	msg := &GetExecutorOwnerMsg{}
	msg.MsgType = GetExecutorOwnerPayloadType
	msg.ColonyName = colonyName
	msg.ExecutorName = executorName

	return msg
}

func (msg *GetExecutorOwnerMsg) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *GetExecutorOwnerMsg) ToJSONIndent() (string, error) {
	jsonBytes, err := json.MarshalIndent(msg, "", "    ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

func (msg *GetExecutorOwnerMsg) Equals(msg2 *GetExecutorOwnerMsg) bool {
	if msg2 == nil {
		return false
	}

	return msg.MsgType == msg2.MsgType &&
		msg.ColonyName == msg2.ColonyName &&
		msg.ExecutorName == msg2.ExecutorName
}

func CreateGetExecutorOwnerMsgFromJSON(jsonString string) (*GetExecutorOwnerMsg, error) {
	var msg *GetExecutorOwnerMsg

	err := json.Unmarshal([]byte(jsonString), &msg)
	if err != nil {
		return msg, err
	}

	return msg, nil
}
