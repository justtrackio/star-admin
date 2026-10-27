package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gosoline-project/httpserver"
	"github.com/gosoline-project/sqlc"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Table struct {
	Database  string `json:"database" db:"database_name"`
	Name      string `json:"name" db:"name"`
	Engine    string `json:"engine" db:"engine"`
	Rows      *int64 `json:"rows" db:"table_rows"`
	DataBytes *int64 `json:"dataBytes" db:"data_bytes"`
}

type RoutineLoad struct {
	ID                string `json:"id" db:"id"`
	Database          string `json:"database" db:"database_name"`
	Table             string `json:"table" db:"table_name"`
	Name              string `json:"name" db:"name"`
	State             string `json:"state" db:"state"`
	Topic             string `json:"topic" db:"topic"`
	LoadedRows        string `json:"loadedRows" db:"loaded_rows"`
	ErrorRows         string `json:"errorRows" db:"error_rows"`
	RowsPerSecond     string `json:"rowsPerSecond" db:"rows_per_second"`
	Progress          string `json:"progress" db:"progress"`
	Reason            string `json:"reason" db:"reason"`
	TaskConsumeSecond string `json:"taskConsumeSecond" db:"task_consume_second"`
}

type TablesResponse struct {
	Tables []Table `json:"tables"`
}

type RoutineLoadsResponse struct {
	Jobs []RoutineLoad `json:"jobs"`
}

type HandlerTables struct {
	client sqlc.Client
}

func NewHandlerTables(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerTables, error) {
	var client sqlc.Client
	var err error

	if client, err = ProvideStarRocksClient(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not create tables handler: %w", err)
	}

	return &HandlerTables{client: client}, nil
}

func (h *HandlerTables) GetTables(ctx context.Context) (httpserver.Response, error) {
	const tablesQuery = `SELECT t.TABLE_SCHEMA AS database_name, t.TABLE_NAME AS name,
 COALESCE(t.ENGINE, '') AS engine, t.TABLE_ROWS AS table_rows, t.DATA_LENGTH AS data_bytes
 FROM information_schema.tables t
 LEFT JOIN information_schema.materialized_views mv
 ON mv.TABLE_SCHEMA = t.TABLE_SCHEMA AND mv.TABLE_NAME = t.TABLE_NAME
 WHERE t.TABLE_TYPE IN ('BASE TABLE', 'TABLE')
 AND t.TABLE_SCHEMA NOT IN ('information_schema', 'sys', '_statistics_')
 AND mv.MATERIALIZED_VIEW_ID IS NULL
 ORDER BY t.TABLE_SCHEMA, t.TABLE_NAME`

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	tables := make([]Table, 0)
	if err := h.client.Select(ctx, &tables, tablesQuery); err != nil {
		return nil, fmt.Errorf("could not list tables: %w", err)
	}

	return httpserver.NewJsonResponse(TablesResponse{Tables: tables}), nil
}

func (h *HandlerTables) GetRoutineLoads(ctx context.Context) (httpserver.Response, error) {
	const routineLoadsQuery = `SELECT CAST(ID AS CHAR) AS id, DB_NAME AS database_name,
 TABLE_NAME AS table_name, NAME AS name, COALESCE(STATE, '') AS state,
 COALESCE(get_json_string(DATA_SOURCE_PROPERTIES, '$.topic'), '') AS topic,
 COALESCE(get_json_string(STATISTICS, '$.loadedRows'), '') AS loaded_rows,
 COALESCE(get_json_string(STATISTICS, '$.errorRows'), '') AS error_rows,
 COALESCE(get_json_string(STATISTICS, '$.loadRowsRate'), '') AS rows_per_second,
 COALESCE(PROGRESS, '') AS progress, COALESCE(REASONS_OF_STATE_CHANGED, '') AS reason,
 COALESCE(get_json_string(JOB_PROPERTIES, '$.taskConsumeSecond'), '') AS task_consume_second
 FROM information_schema.routine_load_jobs
 WHERE DATA_SOURCE_TYPE = 'KAFKA'
 ORDER BY DB_NAME, TABLE_NAME, NAME, ID`

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	jobs := make([]RoutineLoad, 0)
	if err := h.client.Select(ctx, &jobs, routineLoadsQuery); err != nil {
		return nil, fmt.Errorf("could not list Kafka routine load jobs: %w", err)
	}

	return httpserver.NewJsonResponse(RoutineLoadsResponse{Jobs: jobs}), nil
}

func (h *HandlerTables) WatchTables(_ context.Context, request *http.Request, writer *httpserver.SseWriter) error {
	return watchSnapshots(request, writer, h.GetTables)
}

func (h *HandlerTables) WatchRoutineLoads(_ context.Context, request *http.Request, writer *httpserver.SseWriter) error {
	return watchSnapshots(request, writer, h.GetRoutineLoads)
}

type RoutineLoadControlInput struct {
	Database string `uri:"database" binding:"required"`
	ID       string `uri:"id" binding:"required,numeric"`
	Action   string `uri:"action" binding:"required,oneof=pause resume resume-earliest resume-latest"`
}

func (h *HandlerTables) ControlRoutineLoad(ctx context.Context, input *RoutineLoadControlInput) (httpserver.Response, error) {
	const routineLoadActionPause = "pause"

	recoverOffsets := input.Action == "resume-earliest" || input.Action == "resume-latest"
	if input.Action != routineLoadActionPause && input.Action != "resume" && !recoverOffsets {
		return nil, httpserver.NewErrorWithStatus(http.StatusBadRequest, errors.New("unsupported routine action"))
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	jobs := make([]RoutineLoad, 0)
	query := `SELECT NAME AS name, COALESCE(STATE, '') AS state, COALESCE(PROGRESS, '') AS progress,
 COALESCE(REASONS_OF_STATE_CHANGED, '') AS reason FROM information_schema.routine_load_jobs
 WHERE DB_NAME = ? AND ID = ? AND DATA_SOURCE_TYPE = 'KAFKA'`

	if err := h.client.Select(ctx, &jobs, query, input.Database, input.ID); err != nil {
		return nil, fmt.Errorf("could not find Kafka routine: %w", err)
	}

	if len(jobs) == 0 {
		return nil, httpserver.NewErrorWithStatus(http.StatusNotFound, errors.New("kafka routine not found"))
	}

	job := jobs[0]

	canPause := job.State == "RUNNING" || job.State == "NEED_SCHEDULE" || job.State == "UNSTABLE"
	if (input.Action != routineLoadActionPause && job.State != "PAUSED") || (input.Action == routineLoadActionPause && !canPause) {
		return nil, httpserver.NewErrorWithStatus(http.StatusConflict, fmt.Errorf("cannot %s a routine in state %s; reload the routine", input.Action, job.State))
	}

	// Routine Load commands require identifiers, which cannot use query parameters.
	identifier := "`" + strings.ReplaceAll(input.Database, "`", "``") + "`.`" + strings.ReplaceAll(job.Name, "`", "``") + "`"
	if recoverOffsets {
		if err := h.resetRoutineLoadOffsets(ctx, identifier, job, input.Action); err != nil {
			return nil, err
		}
	}

	command := "RESUME"
	if input.Action == routineLoadActionPause {
		command = "PAUSE"
	}

	query = command + " ROUTINE LOAD FOR " + identifier
	if _, err := h.client.Exec(ctx, query); err != nil {
		if recoverOffsets {
			return nil, fmt.Errorf("offsets were reset, but the routine could not resume: %w", err)
		}

		return nil, fmt.Errorf("could not %s Kafka routine: %w", input.Action, err)
	}

	return httpserver.NewJsonResponse(map[string]string{"action": input.Action}), nil
}

func (h *HandlerTables) resetRoutineLoadOffsets(ctx context.Context, identifier string, job RoutineLoad, action string) error {
	reason := strings.ToLower(job.Reason)
	if !strings.Contains(reason, "offset out of range") && !strings.Contains(reason, "requested offset not available") {
		return httpserver.NewErrorWithStatus(http.StatusConflict, errors.New("offset recovery requires a Kafka offset out of range error; reload the routine"))
	}

	var properties string
	var err error

	if properties, err = h.routineLoadResetProperties(job.Progress, action); err != nil {
		return httpserver.NewErrorWithStatus(http.StatusConflict, err)
	}

	if _, err = h.client.Exec(ctx, "ALTER ROUTINE LOAD FOR "+identifier+properties); err != nil {
		return fmt.Errorf("could not reset Kafka offsets; routine remains paused: %w", err)
	}

	return nil
}

func (h *HandlerTables) routineLoadResetProperties(progress string, action string) (string, error) {
	var partitions map[string]json.RawMessage
	if err := json.Unmarshal([]byte(progress), &partitions); err != nil || len(partitions) == 0 {
		return "", errors.New("cannot reset offsets without reported Kafka partitions")
	}

	ids := make([]int, 0, len(partitions))
	for partition := range partitions {
		var id uint64
		var err error

		if id, err = strconv.ParseUint(partition, 10, 31); err != nil {
			return "", fmt.Errorf("invalid Kafka partition %q: %w", partition, err)
		}

		ids = append(ids, int(id))
	}

	slices.Sort(ids)
	names := make([]string, len(ids))
	offsets := make([]string, len(ids))

	for i, id := range ids {
		names[i] = strconv.Itoa(id)
		offsets[i] = "OFFSET_BEGINNING"

		if action == "resume-latest" {
			offsets[i] = "OFFSET_END"
		}
	}

	return ` FROM KAFKA ("kafka_partitions" = "` + strings.Join(names, ",") + `", "kafka_offsets" = "` + strings.Join(offsets, ",") + `")`, nil
}

func (h *HandlerTables) GetRoutineLoadTasks(ctx context.Context, input *RoutineLoadTasksInput) (response httpserver.Response, err error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	jobs := make([]RoutineLoad, 0)
	tasks := make([]RoutineLoadTask, 0)
	query := `SELECT NAME AS name FROM information_schema.routine_load_jobs WHERE DB_NAME = ? AND ID = ? AND DATA_SOURCE_TYPE = 'KAFKA'`

	if err = h.client.Select(ctx, &jobs, query, input.Database, input.ID); err != nil {
		return nil, fmt.Errorf("could not find Kafka routine: %w", err)
	}

	if len(jobs) == 0 {
		return httpserver.NewJsonResponse(RoutineLoadTasksResponse{Tasks: tasks}), nil
	}

	// SHOW ROUTINE LOAD TASK does not support the prepared statement protocol.
	query = h.routineLoadTasksQuery(input.Database, jobs[0].Name)
	var rows *sqlc.Rows
	var columns []string

	if rows, err = h.client.Query(ctx, query); err != nil {
		return nil, fmt.Errorf("could not list routine load tasks: %w", err)
	}

	defer func() {
		var closeErr error
		if closeErr = rows.Close(); closeErr != nil && err == nil {
			response = nil
			err = fmt.Errorf("could not close routine task rows: %w", closeErr)
		}
	}()

	if columns, err = rows.Columns(); err != nil {
		return nil, fmt.Errorf("could not read routine task columns: %w", err)
	}

	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		destinations := make([]any, len(columns))

		for i := range values {
			destinations[i] = &values[i]
		}

		if err = rows.Scan(destinations...); err != nil {
			return nil, fmt.Errorf("could not read routine task: %w", err)
		}

		metadata := make(map[string]string, len(columns))
		for i, column := range columns {
			metadata[column] = values[i].String
		}

		task := h.routineLoadTaskFromMetadata(metadata)
		if task.JobID == input.ID {
			tasks = append(tasks, task)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate routine tasks: %w", err)
	}

	return httpserver.NewJsonResponse(RoutineLoadTasksResponse{Tasks: tasks}), nil
}

func (h *HandlerTables) routineLoadTasksQuery(database, name string) string {
	name = strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(name)

	return "SHOW ROUTINE LOAD TASK FROM `" + strings.ReplaceAll(database, "`", "``") + "` WHERE JobName = '" + name + "'"
}

func (h *HandlerTables) routineLoadTaskFromMetadata(m map[string]string) RoutineLoadTask {
	for key, value := range m {
		if value == "NULL" {
			m[key] = ""
		}
	}

	return RoutineLoadTask{
		ID: m["TaskId"], JobID: m["JobId"], TransactionID: m["TxnId"], TransactionStatus: m["TxnStatus"],
		Created: m["CreateTime"], LastScheduled: m["LastScheduledTime"], ExecuteStarted: m["ExecuteStartTime"],
		Timeout: m["Timeout"], NodeID: m["BeId"], Progress: m["DataSourceProperties"], Message: m["Message"], Warehouse: m["Warehouse"],
	}
}

func (h *HandlerTables) WatchRoutineLoadTasks(_ context.Context, request *http.Request, input *RoutineLoadTasksInput, writer *httpserver.SseWriter) error {
	// Tasks can execute and be replaced between the usual five-second snapshots.
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	return streamSnapshots(request.Context(), writer, ticker.C, func(ctx context.Context) (httpserver.Response, error) {
		return h.GetRoutineLoadTasks(ctx, input)
	})
}

func (h *HandlerTables) GetRoutineLoadSettings(ctx context.Context) (response httpserver.Response, err error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var rows *sqlc.Rows
	var columns []string

	if rows, err = h.client.Query(ctx, "ADMIN SHOW FRONTEND CONFIG LIKE '%routine_load%'"); err != nil {
		return nil, fmt.Errorf("could not read routine load cluster settings: %w", err)
	}

	defer func() {
		var closeErr error
		if closeErr = rows.Close(); closeErr != nil && err == nil {
			response = nil
			err = fmt.Errorf("could not close routine setting rows: %w", closeErr)
		}
	}()

	if columns, err = rows.Columns(); err != nil {
		return nil, fmt.Errorf("could not read routine setting columns: %w", err)
	}

	keyIndex, valueIndex := slices.Index(columns, "Key"), slices.Index(columns, "Value")
	if keyIndex < 0 || valueIndex < 0 {
		return nil, errors.New("routine setting metadata is missing Key or Value columns")
	}

	settings := RoutineLoadSettings{}
	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		destinations := make([]any, len(columns))

		for i := range values {
			destinations[i] = &values[i]
		}

		if err = rows.Scan(destinations...); err != nil {
			return nil, fmt.Errorf("could not read routine setting: %w", err)
		}

		switch values[keyIndex].String {
		case "max_routine_load_batch_size":
			settings.MaxBatchSize = values[valueIndex].String
		case "routine_load_task_consume_second":
			settings.TaskConsumeSecond = values[valueIndex].String
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate routine settings: %w", err)
	}

	return httpserver.NewJsonResponse(settings), nil
}

func (h *HandlerTables) WatchRoutineLoadSettings(_ context.Context, request *http.Request, writer *httpserver.SseWriter) error {
	return watchSnapshots(request, writer, h.GetRoutineLoadSettings)
}
