package postgresql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/colonyos/colonies/pkg/core"
	_ "github.com/lib/pq"
)

// Column list shared by every SELECT so that scanning never depends on the
// physical column order of the table.
const usageRecordColumns = `USAGE_ID, COLONY_NAME, PROCESS_ID, ATTEMPT, PROCESSGRAPH_ID, INITIATOR_ID, INITIATOR_NAME, LABEL, FUNCNAME, EXECUTOR_ID, EXECUTOR_NAME, EXECUTOR_TYPE, PROVIDER_ID, PROVIDER_NAME, LOCATION_NAME, HARDWARE, CPU, MEMORY, STORAGE, GPUNAME, GPUCOUNT, GPUMEM, NODES, START_TIME, END_TIME, OUTCOME, CPU_SECONDS, GPU_SECONDS, MEM_PEAK_BYTES, IO_READ_BYTES, IO_WRITE_BYTES, MEASURED_SOURCE, LAST_REPORT_TIME`

// Wall-clock seconds of a record; open records count up to now.
const usageWallSeconds = `EXTRACT(EPOCH FROM (COALESCE(END_TIME, NOW()) - START_TIME))`

func (db *PQDatabase) OpenUsageRecord(record *core.UsageRecord) error {
	if record == nil {
		return errors.New("Usage record is nil")
	}
	if record.ID == "" {
		record.ID = core.GenerateRandomID()
	}
	if record.StartTime.IsZero() {
		record.StartTime = time.Now()
	}
	record.Outcome = core.USAGE_RUNNING

	hardwareJSON, err := json.Marshal(record.Hardware)
	if err != nil {
		return err
	}

	sqlStatement := `INSERT INTO ` + db.dbPrefix + `USAGE_RECORDS (` + usageRecordColumns + `) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, NULL, $25, 0, 0, 0, 0, 0, '', NULL)`
	_, err = db.postgresql.Exec(sqlStatement,
		record.ID, record.ColonyName, record.ProcessID, record.Attempt, record.ProcessGraphID,
		record.InitiatorID, record.InitiatorName, record.Label, record.FuncName,
		record.ExecutorID, record.ExecutorName, record.ExecutorType, record.ProviderID, record.ProviderName,
		record.LocationName, string(hardwareJSON),
		record.CPU, record.Memory, record.Storage, record.GPUName, record.GPUCount, record.GPUMemory, record.Nodes,
		record.StartTime, record.Outcome)
	return err
}

func (db *PQDatabase) CloseUsageRecord(processID string, endTime time.Time, outcome string) error {
	sqlStatement := `UPDATE ` + db.dbPrefix + `USAGE_RECORDS SET END_TIME=$1, OUTCOME=$2 WHERE PROCESS_ID=$3 AND OUTCOME=$4`
	_, err := db.postgresql.Exec(sqlStatement, endTime, outcome, processID, core.USAGE_RUNNING)
	return err
}

func (db *PQDatabase) CloseUsageRecordsByExecutor(colonyName string, executorName string, endTime time.Time, outcome string) error {
	sqlStatement := `UPDATE ` + db.dbPrefix + `USAGE_RECORDS SET END_TIME=$1, OUTCOME=$2 WHERE COLONY_NAME=$3 AND EXECUTOR_NAME=$4 AND OUTCOME=$5`
	_, err := db.postgresql.Exec(sqlStatement, endTime, outcome, colonyName, executorName, core.USAGE_RUNNING)
	return err
}

func (db *PQDatabase) SetMeasuredUsage(processID string, cpuSeconds float64, gpuSeconds float64, memPeakBytes int64, ioReadBytes int64, ioWriteBytes int64, source string, reportTime time.Time) error {
	sqlStatement := `UPDATE ` + db.dbPrefix + `USAGE_RECORDS SET CPU_SECONDS=$1, GPU_SECONDS=$2, MEM_PEAK_BYTES=$3, IO_READ_BYTES=$4, IO_WRITE_BYTES=$5, MEASURED_SOURCE=$6, LAST_REPORT_TIME=$7 WHERE PROCESS_ID=$8 AND OUTCOME=$9`
	_, err := db.postgresql.Exec(sqlStatement, cpuSeconds, gpuSeconds, memPeakBytes, ioReadBytes, ioWriteBytes, source, reportTime, processID, core.USAGE_RUNNING)
	return err
}

func (db *PQDatabase) parseUsageRecords(rows *sql.Rows) ([]*core.UsageRecord, error) {
	var records []*core.UsageRecord

	for rows.Next() {
		r := &core.UsageRecord{}
		var hardwareJSON string
		var endTime sql.NullTime
		var lastReport sql.NullTime
		if err := rows.Scan(&r.ID, &r.ColonyName, &r.ProcessID, &r.Attempt, &r.ProcessGraphID,
			&r.InitiatorID, &r.InitiatorName, &r.Label, &r.FuncName,
			&r.ExecutorID, &r.ExecutorName, &r.ExecutorType, &r.ProviderID, &r.ProviderName,
			&r.LocationName, &hardwareJSON,
			&r.CPU, &r.Memory, &r.Storage, &r.GPUName, &r.GPUCount, &r.GPUMemory, &r.Nodes,
			&r.StartTime, &endTime, &r.Outcome,
			&r.CPUSeconds, &r.GPUSeconds, &r.MemPeakBytes, &r.IOReadBytes, &r.IOWriteBytes, &r.MeasuredSource, &lastReport); err != nil {
			return nil, err
		}
		if endTime.Valid {
			r.EndTime = endTime.Time
		}
		if lastReport.Valid {
			r.LastReportTime = lastReport.Time
		}
		if hardwareJSON != "" {
			if err := json.Unmarshal([]byte(hardwareJSON), &r.Hardware); err != nil {
				return nil, err
			}
		}
		records = append(records, r)
	}

	return records, nil
}

func (db *PQDatabase) GetUsageRecordsByProcessID(processID string) ([]*core.UsageRecord, error) {
	sqlStatement := `SELECT ` + usageRecordColumns + ` FROM ` + db.dbPrefix + `USAGE_RECORDS WHERE PROCESS_ID=$1 ORDER BY START_TIME ASC`
	rows, err := db.postgresql.Query(sqlStatement, processID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return db.parseUsageRecords(rows)
}

func (db *PQDatabase) QueryUsageRecords(colonyName string, from time.Time, to time.Time, initiatorID string, providerID string, executorName string, limit int) ([]*core.UsageRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	sqlStatement := `SELECT ` + usageRecordColumns + ` FROM ` + db.dbPrefix + `USAGE_RECORDS WHERE COLONY_NAME=$1 AND START_TIME>=$2 AND START_TIME<=$3 AND ($4='' OR INITIATOR_ID=$4) AND ($5='' OR PROVIDER_ID=$5) AND ($6='' OR EXECUTOR_NAME=$6) ORDER BY START_TIME DESC LIMIT $7`
	rows, err := db.postgresql.Query(sqlStatement, colonyName, from, to, initiatorID, providerID, executorName, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return db.parseUsageRecords(rows)
}

// usageGroupColumns maps a group-by key to its (key column, name column).
// Whitelisted so the key can never reach the SQL unvalidated.
func usageGroupColumns(groupBy string) (string, string, error) {
	switch groupBy {
	case core.USAGE_GROUP_INITIATOR:
		return "INITIATOR_ID", "INITIATOR_NAME", nil
	case core.USAGE_GROUP_PROVIDER:
		return "PROVIDER_ID", "PROVIDER_NAME", nil
	case core.USAGE_GROUP_EXECUTOR:
		return "EXECUTOR_ID", "EXECUTOR_NAME", nil
	case core.USAGE_GROUP_EXECUTOR_TYPE:
		return "EXECUTOR_TYPE", "EXECUTOR_TYPE", nil
	case core.USAGE_GROUP_LABEL:
		return "LABEL", "LABEL", nil
	}
	return "", "", errors.New("Invalid usage group key <" + groupBy + ">")
}

const usageAggregates = `COUNT(*), ` +
	`SUM(CASE WHEN OUTCOME='` + core.USAGE_SUCCESS + `' THEN 1 ELSE 0 END), ` +
	`SUM(CASE WHEN OUTCOME IN ('` + core.USAGE_FAILED + `','` + core.USAGE_TIMEOUT + `') THEN 1 ELSE 0 END), ` +
	`SUM(CASE WHEN OUTCOME='` + core.USAGE_RUNNING + `' THEN 1 ELSE 0 END), ` +
	`SUM(` + usageWallSeconds + `), ` +
	`SUM(CPU * ` + usageWallSeconds + ` / 1000.0), ` +
	`SUM(GPUCOUNT * ` + usageWallSeconds + `), ` +
	`SUM(MEMORY * ` + usageWallSeconds + `), ` +
	`SUM(CPU_SECONDS), SUM(GPU_SECONDS)`

func scanUsageAggregates(rows *sql.Rows, s *core.UsageSummary, extra ...interface{}) error {
	var attempts, succeeded, failed, running sql.NullInt64
	var wall, core_, gpu, mem, mcpu, mgpu sql.NullFloat64
	dest := append(extra, &attempts, &succeeded, &failed, &running, &wall, &core_, &gpu, &mem, &mcpu, &mgpu)
	if err := rows.Scan(dest...); err != nil {
		return err
	}
	s.Attempts = int(attempts.Int64)
	s.Succeeded = int(succeeded.Int64)
	s.Failed = int(failed.Int64)
	s.Running = int(running.Int64)
	s.WallSeconds = wall.Float64
	s.CoreSeconds = core_.Float64
	s.GPUSeconds = gpu.Float64
	s.MemByteSeconds = mem.Float64
	s.MeasuredCPUSeconds = mcpu.Float64
	s.MeasuredGPUSeconds = mgpu.Float64
	return nil
}

func (db *PQDatabase) SummarizeUsage(colonyName string, from time.Time, to time.Time, groupBy string, initiatorID string, providerID string) ([]*core.UsageSummary, error) {
	keyCol, nameCol, err := usageGroupColumns(groupBy)
	if err != nil {
		return nil, err
	}

	sqlStatement := `SELECT ` + keyCol + `, ` + nameCol + `, ` + usageAggregates + ` FROM ` + db.dbPrefix + `USAGE_RECORDS WHERE COLONY_NAME=$1 AND START_TIME>=$2 AND START_TIME<=$3 AND ($4='' OR INITIATOR_ID=$4) AND ($5='' OR PROVIDER_ID=$5) GROUP BY ` + keyCol + `, ` + nameCol + ` ORDER BY 7 DESC`
	rows, err := db.postgresql.Query(sqlStatement, colonyName, from, to, initiatorID, providerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []*core.UsageSummary
	for rows.Next() {
		s := &core.UsageSummary{}
		if err := scanUsageAggregates(rows, s, &s.Key, &s.Name); err != nil {
			return nil, err
		}
		summaries = append(summaries, s)
	}

	return summaries, nil
}

func (db *PQDatabase) UsageBalance(colonyName string, from time.Time, to time.Time) ([]*core.UsageBalance, error) {
	sqlStatement := `SELECT INITIATOR_ID, INITIATOR_NAME, PROVIDER_ID, PROVIDER_NAME, ` + usageAggregates + ` FROM ` + db.dbPrefix + `USAGE_RECORDS WHERE COLONY_NAME=$1 AND START_TIME>=$2 AND START_TIME<=$3 GROUP BY INITIATOR_ID, INITIATOR_NAME, PROVIDER_ID, PROVIDER_NAME`
	rows, err := db.postgresql.Query(sqlStatement, colonyName, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	balances := make(map[string]*core.UsageBalance)
	get := func(id string, name string) *core.UsageBalance {
		if id == "" {
			name = "colony"
		}
		b, ok := balances[id]
		if !ok {
			b = &core.UsageBalance{UserID: id, UserName: name}
			balances[id] = b
		}
		return b
	}

	for rows.Next() {
		var initiatorID, initiatorName, providerID, providerName string
		s := &core.UsageSummary{}
		if err := scanUsageAggregates(rows, s, &initiatorID, &initiatorName, &providerID, &providerName); err != nil {
			return nil, err
		}
		if initiatorID == providerID {
			b := get(initiatorID, initiatorName)
			b.SelfCoreSeconds += s.CoreSeconds
			b.SelfGPUSeconds += s.GPUSeconds
			continue
		}
		consumer := get(initiatorID, initiatorName)
		consumer.ConsumedCoreSeconds += s.CoreSeconds
		consumer.ConsumedGPUSeconds += s.GPUSeconds
		provider := get(providerID, providerName)
		provider.ProvidedCoreSeconds += s.CoreSeconds
		provider.ProvidedGPUSeconds += s.GPUSeconds
	}

	result := make([]*core.UsageBalance, 0, len(balances))
	for _, b := range balances {
		result = append(result, b)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].NetCoreSeconds() > result[j].NetCoreSeconds() })

	return result, nil
}

func (db *PQDatabase) SetExecutorOwner(colonyName string, executorName string, ownerID string, ownerName string) error {
	now := time.Now()
	sqlStatement := `UPDATE ` + db.dbPrefix + `EXECUTOR_OWNERS SET VALID_TO=$1 WHERE COLONY_NAME=$2 AND EXECUTOR_NAME=$3 AND VALID_TO IS NULL`
	_, err := db.postgresql.Exec(sqlStatement, now, colonyName, executorName)
	if err != nil {
		return err
	}

	sqlStatement = `INSERT INTO ` + db.dbPrefix + `EXECUTOR_OWNERS (ID, COLONY_NAME, EXECUTOR_NAME, OWNER_ID, OWNER_NAME, VALID_FROM, VALID_TO) VALUES ($1, $2, $3, $4, $5, $6, NULL)`
	_, err = db.postgresql.Exec(sqlStatement, core.GenerateRandomID(), colonyName, executorName, ownerID, ownerName, now)
	return err
}

func (db *PQDatabase) GetExecutorOwner(colonyName string, executorName string) (*core.ExecutorOwner, error) {
	sqlStatement := `SELECT COLONY_NAME, EXECUTOR_NAME, OWNER_ID, OWNER_NAME, VALID_FROM FROM ` + db.dbPrefix + `EXECUTOR_OWNERS WHERE COLONY_NAME=$1 AND EXECUTOR_NAME=$2 AND VALID_TO IS NULL`
	rows, err := db.postgresql.Query(sqlStatement, colonyName, executorName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}
	owner := &core.ExecutorOwner{}
	if err := rows.Scan(&owner.ColonyName, &owner.ExecutorName, &owner.OwnerID, &owner.OwnerName, &owner.ValidFrom); err != nil {
		return nil, err
	}

	return owner, nil
}
