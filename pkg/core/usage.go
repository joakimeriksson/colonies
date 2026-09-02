package core

import (
	"encoding/json"
	"time"
)

// Outcome of a usage record. A record is opened when a process is assigned
// to an executor and closed exactly once with one of the outcomes below.
const (
	USAGE_RUNNING       = "running"
	USAGE_SUCCESS       = "success"
	USAGE_FAILED        = "failed"
	USAGE_TIMEOUT       = "timeout"
	USAGE_UNASSIGNED    = "unassigned"
	USAGE_EXECUTOR_LOST = "executor_lost"
)

// Group-by keys accepted by usage summaries.
const (
	USAGE_GROUP_INITIATOR     = "initiator"
	USAGE_GROUP_PROVIDER      = "provider"
	USAGE_GROUP_EXECUTOR      = "executor"
	USAGE_GROUP_EXECUTOR_TYPE = "executortype"
	USAGE_GROUP_LABEL         = "label"
)

// UsageRecord is one attempt to run one process on one executor. It is
// append-only: the reserved amounts and the identities are copied in when the
// record is opened and never change, so later changes to the executor row
// (re-registration, new owner, new hardware) cannot rewrite history.
type UsageRecord struct {
	ID             string `json:"usageid"`
	ColonyName     string `json:"colonyname"`
	ProcessID      string `json:"processid"`
	Attempt        int    `json:"attempt"`
	ProcessGraphID string `json:"processgraphid"`
	InitiatorID    string `json:"initiatorid"`
	InitiatorName  string `json:"initiatorname"`
	Label          string `json:"label"`
	FuncName       string `json:"funcname"`

	ExecutorID   string     `json:"executorid"`
	ExecutorName string     `json:"executorname"`
	ExecutorType string     `json:"executortype"`
	ProviderID   string     `json:"providerid"`
	ProviderName string     `json:"providername"`
	LocationName string     `json:"locationname"`
	Hardware     []Hardware `json:"hardware"`

	// Reserved amounts, taken from the FunctionSpec conditions.
	CPU       int64  `json:"cpu"` // millicores
	Memory    int64  `json:"mem"` // bytes
	Storage   int64  `json:"storage"`
	GPUName   string `json:"gpuname"`
	GPUCount  int    `json:"gpucount"`
	GPUMemory int64  `json:"gpumem"`
	Nodes     int    `json:"nodes"`

	StartTime time.Time `json:"starttime"`
	EndTime   time.Time `json:"endtime"`
	Outcome   string    `json:"outcome"`

	// Measured amounts, reported by the executor. Zero when not reported.
	CPUSeconds     float64   `json:"cpuseconds"`
	GPUSeconds     float64   `json:"gpuseconds"`
	MemPeakBytes   int64     `json:"mempeakbytes"`
	IOReadBytes    int64     `json:"ioreadbytes"`
	IOWriteBytes   int64     `json:"iowritebytes"`
	MeasuredSource string    `json:"measuredsource"`
	LastReportTime time.Time `json:"lastreporttime"`
}

// WallTime returns how long the attempt has been, or was, running.
func (r *UsageRecord) WallTime() time.Duration {
	if r.EndTime.IsZero() {
		return time.Since(r.StartTime)
	}
	return r.EndTime.Sub(r.StartTime)
}

// UsageSummary is the aggregate of many usage records sharing one group key.
// All seconds are wall-clock seconds multiplied by the reserved amount, so
// CoreSeconds / 3600 is core hours.
type UsageSummary struct {
	Key            string  `json:"key"`
	Name           string  `json:"name"`
	Attempts       int     `json:"attempts"`
	Succeeded      int     `json:"succeeded"`
	Failed         int     `json:"failed"`
	Running        int     `json:"running"`
	WallSeconds    float64 `json:"wallseconds"`
	CoreSeconds    float64 `json:"coreseconds"`
	GPUSeconds     float64 `json:"gpuseconds"`
	MemByteSeconds float64 `json:"membyteseconds"`
	// Measured, summed over records that reported them.
	MeasuredCPUSeconds float64 `json:"measuredcpuseconds"`
	MeasuredGPUSeconds float64 `json:"measuredgpuseconds"`
}

// UsageBalance is the provider view for one colony member: what they
// consumed on other members' executors, what other members consumed on
// theirs, and what they ran on their own. Units are reserved core seconds and
// GPU seconds. Turning the net into credits is left to the reader.
type UsageBalance struct {
	UserID              string  `json:"userid"`
	UserName            string  `json:"username"`
	ConsumedCoreSeconds float64 `json:"consumedcoreseconds"`
	ProvidedCoreSeconds float64 `json:"providedcoreseconds"`
	SelfCoreSeconds     float64 `json:"selfcoreseconds"`
	ConsumedGPUSeconds  float64 `json:"consumedgpuseconds"`
	ProvidedGPUSeconds  float64 `json:"providedgpuseconds"`
	SelfGPUSeconds      float64 `json:"selfgpuseconds"`
}

func (b *UsageBalance) NetCoreSeconds() float64 {
	return b.ProvidedCoreSeconds - b.ConsumedCoreSeconds
}

func (b *UsageBalance) NetGPUSeconds() float64 {
	return b.ProvidedGPUSeconds - b.ConsumedGPUSeconds
}

// ExecutorOwner records which colony user a machine belongs to. An empty
// OwnerID means the colony itself owns the executor.
type ExecutorOwner struct {
	ColonyName   string    `json:"colonyname"`
	ExecutorName string    `json:"executorname"`
	OwnerID      string    `json:"ownerid"`
	OwnerName    string    `json:"ownername"`
	ValidFrom    time.Time `json:"validfrom"`
	ValidTo      time.Time `json:"validto"`
}

func ConvertJSONToUsageRecordArray(jsonString string) ([]*UsageRecord, error) {
	var records []*UsageRecord
	err := json.Unmarshal([]byte(jsonString), &records)
	return records, err
}

func ConvertUsageRecordArrayToJSON(records []*UsageRecord) (string, error) {
	jsonBytes, err := json.Marshal(records)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func ConvertJSONToUsageSummaryArray(jsonString string) ([]*UsageSummary, error) {
	var summaries []*UsageSummary
	err := json.Unmarshal([]byte(jsonString), &summaries)
	return summaries, err
}

func ConvertUsageSummaryArrayToJSON(summaries []*UsageSummary) (string, error) {
	jsonBytes, err := json.Marshal(summaries)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func ConvertJSONToUsageBalanceArray(jsonString string) ([]*UsageBalance, error) {
	var balances []*UsageBalance
	err := json.Unmarshal([]byte(jsonString), &balances)
	return balances, err
}

func ConvertUsageBalanceArrayToJSON(balances []*UsageBalance) (string, error) {
	jsonBytes, err := json.Marshal(balances)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func ConvertJSONToExecutorOwner(jsonString string) (*ExecutorOwner, error) {
	var owner *ExecutorOwner
	err := json.Unmarshal([]byte(jsonString), &owner)
	return owner, err
}

func (owner *ExecutorOwner) ToJSON() (string, error) {
	jsonBytes, err := json.Marshal(owner)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
