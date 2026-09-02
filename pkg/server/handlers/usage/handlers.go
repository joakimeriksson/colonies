package usage

import (
	"errors"
	"net/http"
	"time"

	"github.com/colonyos/colonies/pkg/backends"
	"github.com/colonyos/colonies/pkg/core"
	"github.com/colonyos/colonies/pkg/database"
	"github.com/colonyos/colonies/pkg/rpc"
	"github.com/colonyos/colonies/pkg/security"
	"github.com/colonyos/colonies/pkg/server/registry"
	log "github.com/sirupsen/logrus"
)

const maxUsageRecords = 1000
const defaultUsageWindow = 30 * 24 * time.Hour

type Server interface {
	HandleHTTPError(c backends.Context, err error, errorCode int) bool
	SendHTTPReply(c backends.Context, payloadType string, jsonString string)
	SendEmptyHTTPReply(c backends.Context, payloadType string)
	GetUsageDB() database.UsageDatabase
	GetUserDB() database.UserDatabase
	GetColonyDB() database.ColonyDatabase
	GetExecutorDB() database.ExecutorDatabase
	GetValidator() security.Validator
}

type Handlers struct {
	server Server
}

func NewHandlers(server Server) *Handlers {
	return &Handlers{server: server}
}

func (h *Handlers) RegisterHandlers(handlerRegistry *registry.HandlerRegistry) error {
	if err := handlerRegistry.Register(rpc.GetUsagePayloadType, h.HandleGetUsage); err != nil {
		return err
	}
	if err := handlerRegistry.Register(rpc.SetExecutorOwnerPayloadType, h.HandleSetExecutorOwner); err != nil {
		return err
	}
	if err := handlerRegistry.Register(rpc.GetExecutorOwnerPayloadType, h.HandleGetExecutorOwner); err != nil {
		return err
	}
	return nil
}

func (h *Handlers) requireColony(c backends.Context, colonyName string) bool {
	colony, err := h.server.GetColonyDB().GetColonyByName(colonyName)
	if h.server.HandleHTTPError(c, err, http.StatusBadRequest) {
		return false
	}
	if colony == nil {
		h.server.HandleHTTPError(c, errors.New("Colony with name <"+colonyName+"> does not exists"), http.StatusBadRequest)
		return false
	}
	return true
}

// HandleGetUsage serves the ledger. The colony owner sees everything. A
// member sees what they ran, and what ran on executors they own; the balance
// view is aggregated per member and open to every member.
func (h *Handlers) HandleGetUsage(c backends.Context, recoveredID string, payloadType string, jsonString string) {
	msg, err := rpc.CreateGetUsageMsgFromJSON(jsonString)
	if err != nil {
		if h.server.HandleHTTPError(c, errors.New("Failed to get usage, invalid JSON"), http.StatusBadRequest) {
			return
		}
	}

	if msg.MsgType != payloadType {
		h.server.HandleHTTPError(c, errors.New("Failed to get usage, msg.MsgType does not match payloadType"), http.StatusBadRequest)
		return
	}

	if !h.requireColony(c, msg.ColonyName) {
		return
	}

	isColonyOwner := h.server.GetValidator().RequireColonyOwner(recoveredID, msg.ColonyName) == nil
	if !isColonyOwner {
		err = h.server.GetValidator().RequireMembership(recoveredID, msg.ColonyName, true)
		if h.server.HandleHTTPError(c, err, http.StatusForbidden) {
			return
		}
	}

	if !isColonyOwner && msg.Mode != rpc.UsageModeBalance && msg.ProviderID != recoveredID {
		msg.InitiatorID = recoveredID
	}

	to := time.Now()
	if msg.To > 0 {
		to = time.Unix(msg.To, 0)
	}
	from := to.Add(-defaultUsageWindow)
	if msg.From > 0 {
		from = time.Unix(msg.From, 0)
	}

	var reply string
	switch msg.Mode {
	case rpc.UsageModeRecords:
		count := msg.Count
		if count <= 0 || count > maxUsageRecords {
			count = maxUsageRecords
		}
		records, err := h.server.GetUsageDB().QueryUsageRecords(msg.ColonyName, from, to, msg.InitiatorID, msg.ProviderID, msg.ExecutorName, count)
		if h.server.HandleHTTPError(c, err, http.StatusBadRequest) {
			return
		}
		if records == nil {
			records = []*core.UsageRecord{}
		}
		reply, err = core.ConvertUsageRecordArrayToJSON(records)
		if h.server.HandleHTTPError(c, err, http.StatusInternalServerError) {
			return
		}
	case rpc.UsageModeBalance:
		balances, err := h.server.GetUsageDB().UsageBalance(msg.ColonyName, from, to)
		if h.server.HandleHTTPError(c, err, http.StatusBadRequest) {
			return
		}
		if balances == nil {
			balances = []*core.UsageBalance{}
		}
		reply, err = core.ConvertUsageBalanceArrayToJSON(balances)
		if h.server.HandleHTTPError(c, err, http.StatusInternalServerError) {
			return
		}
	default:
		groupBy := msg.GroupBy
		if groupBy == "" {
			groupBy = core.USAGE_GROUP_INITIATOR
		}
		summaries, err := h.server.GetUsageDB().SummarizeUsage(msg.ColonyName, from, to, groupBy, msg.InitiatorID, msg.ProviderID)
		if h.server.HandleHTTPError(c, err, http.StatusBadRequest) {
			return
		}
		if summaries == nil {
			summaries = []*core.UsageSummary{}
		}
		reply, err = core.ConvertUsageSummaryArrayToJSON(summaries)
		if h.server.HandleHTTPError(c, err, http.StatusInternalServerError) {
			return
		}
	}

	log.WithFields(log.Fields{"ColonyName": msg.ColonyName, "Mode": msg.Mode, "GroupBy": msg.GroupBy}).Debug("Getting usage")

	h.server.SendHTTPReply(c, payloadType, reply)
}

func (h *Handlers) HandleSetExecutorOwner(c backends.Context, recoveredID string, payloadType string, jsonString string) {
	msg, err := rpc.CreateSetExecutorOwnerMsgFromJSON(jsonString)
	if err != nil {
		if h.server.HandleHTTPError(c, errors.New("Failed to set executor owner, invalid JSON"), http.StatusBadRequest) {
			return
		}
	}

	if msg.MsgType != payloadType {
		h.server.HandleHTTPError(c, errors.New("Failed to set executor owner, msg.MsgType does not match payloadType"), http.StatusBadRequest)
		return
	}

	if !h.requireColony(c, msg.ColonyName) {
		return
	}

	err = h.server.GetValidator().RequireColonyOwner(recoveredID, msg.ColonyName)
	if h.server.HandleHTTPError(c, err, http.StatusForbidden) {
		return
	}

	executor, err := h.server.GetExecutorDB().GetExecutorByName(msg.ColonyName, msg.ExecutorName)
	if h.server.HandleHTTPError(c, err, http.StatusBadRequest) {
		return
	}
	if executor == nil {
		h.server.HandleHTTPError(c, errors.New("Executor with name <"+msg.ExecutorName+"> does not exists"), http.StatusBadRequest)
		return
	}

	ownerID := ""
	if msg.OwnerName != "" {
		user, err := h.server.GetUserDB().GetUserByName(msg.ColonyName, msg.OwnerName)
		if h.server.HandleHTTPError(c, err, http.StatusBadRequest) {
			return
		}
		if user == nil {
			h.server.HandleHTTPError(c, errors.New("User with name <"+msg.OwnerName+"> does not exists in colony <"+msg.ColonyName+">"), http.StatusBadRequest)
			return
		}
		ownerID = user.ID
	}

	err = h.server.GetUsageDB().SetExecutorOwner(msg.ColonyName, msg.ExecutorName, ownerID, msg.OwnerName)
	if h.server.HandleHTTPError(c, err, http.StatusBadRequest) {
		return
	}

	owner, err := h.server.GetUsageDB().GetExecutorOwner(msg.ColonyName, msg.ExecutorName)
	if h.server.HandleHTTPError(c, err, http.StatusInternalServerError) {
		return
	}

	jsonString, err = owner.ToJSON()
	if h.server.HandleHTTPError(c, err, http.StatusInternalServerError) {
		return
	}

	log.WithFields(log.Fields{"ColonyName": msg.ColonyName, "ExecutorName": msg.ExecutorName, "OwnerName": msg.OwnerName}).Debug("Setting executor owner")

	h.server.SendHTTPReply(c, payloadType, jsonString)
}

func (h *Handlers) HandleGetExecutorOwner(c backends.Context, recoveredID string, payloadType string, jsonString string) {
	msg, err := rpc.CreateGetExecutorOwnerMsgFromJSON(jsonString)
	if err != nil {
		if h.server.HandleHTTPError(c, errors.New("Failed to get executor owner, invalid JSON"), http.StatusBadRequest) {
			return
		}
	}

	if msg.MsgType != payloadType {
		h.server.HandleHTTPError(c, errors.New("Failed to get executor owner, msg.MsgType does not match payloadType"), http.StatusBadRequest)
		return
	}

	if !h.requireColony(c, msg.ColonyName) {
		return
	}

	if h.server.GetValidator().RequireColonyOwner(recoveredID, msg.ColonyName) != nil {
		err = h.server.GetValidator().RequireMembership(recoveredID, msg.ColonyName, true)
		if h.server.HandleHTTPError(c, err, http.StatusForbidden) {
			return
		}
	}

	owner, err := h.server.GetUsageDB().GetExecutorOwner(msg.ColonyName, msg.ExecutorName)
	if h.server.HandleHTTPError(c, err, http.StatusBadRequest) {
		return
	}
	if owner == nil {
		// Never set: the colony owns it
		owner = &core.ExecutorOwner{ColonyName: msg.ColonyName, ExecutorName: msg.ExecutorName}
	}

	jsonString, err = owner.ToJSON()
	if h.server.HandleHTTPError(c, err, http.StatusInternalServerError) {
		return
	}

	h.server.SendHTTPReply(c, payloadType, jsonString)
}
