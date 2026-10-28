package internal

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gosoline-project/httpserver"
	"github.com/gosoline-project/sqlc"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// MaterializedView describes a view in the configured cluster's internal catalog.
type MaterializedView struct {
	CanResume                       bool     `json:"canResume" db:"-"`
	ID                              string   `json:"id" db:"id"`
	Database                        string   `json:"database" db:"database_name"`
	Name                            string   `json:"name" db:"name"`
	PartitionType                   string   `json:"partitionType" db:"partition_type"`
	RefreshType                     string   `json:"refreshType" db:"refresh_type"`
	RefreshTrigger                  string   `json:"refreshTrigger" db:"refresh_trigger"`
	RefreshPolicy                   string   `json:"refreshPolicy" db:"refresh_policy"`
	Active                          string   `json:"active" db:"active"`
	InactiveReason                  string   `json:"inactiveReason" db:"inactive_reason"`
	LastRefreshState                string   `json:"lastRefreshState" db:"last_refresh_state"`
	LastRefreshFinished             string   `json:"lastRefreshFinished" db:"last_refresh_finished"`
	LastSuccessfulRefreshFinished   string   `json:"lastSuccessfulRefreshFinished" db:"last_successful_refresh_finished"`
	LastSuccessfulRefreshAgeSeconds *float64 `json:"lastSuccessfulRefreshAgeSeconds" db:"last_successful_refresh_age_seconds"`
	LastRefreshDuration             *float64 `json:"lastRefreshDuration" db:"last_refresh_duration"`
	LastRefreshError                string   `json:"lastRefreshError" db:"last_refresh_error"`
	Rows                            *int64   `json:"rows" db:"table_rows"`
	Definition                      string   `json:"definition" db:"definition"`
}

// MaterializedViewsResponse contains the materialized view snapshot.
type MaterializedViewsResponse struct {
	Views []MaterializedView `json:"views"`
}

// HandlerMaterializedViews lists materialized views using the shared pool.
type HandlerMaterializedViews struct {
	client sqlc.Client
	state  sqlc.Client
	// shortcut: local state coordinates one backend; use shared storage and locking before adding control replicas.
	controlMu sync.Mutex
}

func NewHandlerMaterializedViews(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerMaterializedViews, error) {
	var client sqlc.Client
	var err error
	var state sqlc.Client

	if client, err = ProvideStarRocksClient(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not create materialized views handler: %w", err)
	}

	if state, err = ProvideLocalState(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not initialize local state: %w", err)
	}

	return &HandlerMaterializedViews{client: client, state: state}, nil
}

// GetMaterializedViews performs only a metadata read, without refreshing any views.
func (h *HandlerMaterializedViews) GetMaterializedViews(ctx context.Context) (httpserver.Response, error) {
	return h.getMaterializedViews(ctx, nil)
}

func (h *HandlerMaterializedViews) getMaterializedViews(ctx context.Context, input *MaterializedViewTasksInput) (httpserver.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	query := `SELECT
    CAST(mv.MATERIALIZED_VIEW_ID AS CHAR) AS id,
    mv.TABLE_SCHEMA AS database_name,
    mv.TABLE_NAME AS name,
    COALESCE(mv.REFRESH_TYPE, '') AS refresh_type,
    COALESCE(mv.PARTITION_TYPE, '') AS partition_type,
    COALESCE(mv.REFRESH_TRIGGER, '') AS refresh_trigger,
    COALESCE(mv.REFRESH_POLICY, '') AS refresh_policy,
    COALESCE(mv.IS_ACTIVE, '') AS active,
    COALESCE(mv.INACTIVE_REASON, '') AS inactive_reason,
    COALESCE(mv.LAST_REFRESH_STATE, '') AS last_refresh_state,
    COALESCE(CAST(mv.LAST_REFRESH_FINISHED_TIME AS CHAR), '') AS last_refresh_finished,
    COALESCE(CAST(CASE WHEN mv.LAST_REFRESH_STATE = 'SUCCESS'
        THEN mv.LAST_REFRESH_FINISHED_TIME END AS CHAR),
        CAST(success.last_finished AS CHAR), '') AS last_successful_refresh_finished,
    TIMESTAMPDIFF(SECOND, COALESCE(CASE WHEN mv.LAST_REFRESH_STATE = 'SUCCESS'
        THEN mv.LAST_REFRESH_FINISHED_TIME END, success.last_finished), NOW()) AS last_successful_refresh_age_seconds,
    mv.LAST_REFRESH_DURATION AS last_refresh_duration,
    COALESCE(mv.LAST_REFRESH_ERROR_MESSAGE, '') AS last_refresh_error,
    mv.TABLE_ROWS AS table_rows,
    COALESCE(mv.MATERIALIZED_VIEW_DEFINITION, '') AS definition
FROM information_schema.materialized_views mv
LEFT JOIN (
    SELECT TABLE_SCHEMA, MATERIALIZED_VIEW_ID, MAX(FINISH_TIME) AS last_finished
    FROM information_schema.materialized_view_refresh_jobs
    WHERE REFRESH_STATE = 'SUCCESS'
    GROUP BY TABLE_SCHEMA, MATERIALIZED_VIEW_ID
) success ON success.TABLE_SCHEMA = mv.TABLE_SCHEMA
    AND success.MATERIALIZED_VIEW_ID = mv.MATERIALIZED_VIEW_ID
`
	var args []any

	if input != nil {
		query += " WHERE mv.TABLE_SCHEMA = ? AND mv.MATERIALIZED_VIEW_ID = ?"
		args = []any{input.Database, input.ID}
	}

	query += " ORDER BY mv.TABLE_SCHEMA, mv.TABLE_NAME, mv.MATERIALIZED_VIEW_ID"

	views := make([]MaterializedView, 0)
	if err := h.client.Select(ctx, &views, query, args...); err != nil {
		return nil, fmt.Errorf("could not list materialized views: %w", err)
	}

	for i := range views {
		view := &views[i]
		manual := strings.EqualFold(view.RefreshTrigger, "MANUAL")
		view.CanResume = strings.EqualFold(view.Active, "false") && !manual

		if manual && strings.EqualFold(view.Active, "true") {
			if policy, err := h.loadRefreshPolicy(ctx, view.Database, view.ID); err == nil {
				_, policyErr := h.automaticRefreshClause(policy)
				view.CanResume = policyErr == nil
			}
		}
	}

	return httpserver.NewJsonResponse(MaterializedViewsResponse{Views: views}), nil
}

// WatchMaterializedViews refreshes metadata only while this request is connected.
func (h *HandlerMaterializedViews) WatchMaterializedViews(_ context.Context, request *http.Request, writer *httpserver.SseWriter) error {
	return watchSnapshots(request, writer, h.GetMaterializedViews)
}

// WatchMaterializedView streams only the selected view, or an empty snapshot if it was removed.
func (h *HandlerMaterializedViews) WatchMaterializedView(_ context.Context, request *http.Request, input *MaterializedViewTasksInput, writer *httpserver.SseWriter) error {
	return watchSnapshots(request, writer, func(ctx context.Context) (httpserver.Response, error) {
		return h.getMaterializedViews(ctx, input)
	})
}

// GetMaterializedViewTasks reads refresh history without triggering a refresh.
func (h *HandlerMaterializedViews) GetMaterializedViewTasks(ctx context.Context, input *MaterializedViewTasksInput) (httpserver.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	query := `SELECT
 COALESCE(QUERY_ID, '') AS query_id,
 COALESCE(STATE, '') AS state,
 COALESCE(CAST(CREATE_TIME AS CHAR), '') AS created,
 COALESCE(CAST(FINISH_TIME AS CHAR), '') AS finished,
 COALESCE(PROGRESS, '') AS progress,
 COALESCE(ERROR_MESSAGE, '') AS error_message,
 COALESCE(EXTRA_MESSAGE, '') AS extra_message,
 CAST(UNIX_TIMESTAMP(CREATE_TIME) * 1000 AS SIGNED) AS created_millis,
 CAST(UNIX_TIMESTAMP(FINISH_TIME) * 1000 AS SIGNED) AS finished_millis,
 CAST(UNIX_TIMESTAMP(NOW()) * 1000 AS SIGNED) AS observed_millis
FROM information_schema.task_runs
WHERE ` + "`DATABASE`" + ` = ? AND TASK_NAME = ?
ORDER BY CREATE_TIME DESC, QUERY_ID DESC
LIMIT 100`

	tasks := make([]MaterializedViewTask, 0)
	if err := h.client.Select(ctx, &tasks, query, input.Database, "mv-"+input.ID); err != nil {
		return nil, fmt.Errorf("could not list materialized view refresh tasks: %w", err)
	}

	for i := range tasks {
		tasks[i].enrich()
	}

	return httpserver.NewJsonResponse(MaterializedViewTasksResponse{Tasks: tasks}), nil
}

// WatchMaterializedViewTasks streams retained refresh task runs until the client disconnects.
func (h *HandlerMaterializedViews) WatchMaterializedViewTasks(_ context.Context, request *http.Request, input *MaterializedViewTasksInput, writer *httpserver.SseWriter) error {
	return watchSnapshots(request, writer, func(ctx context.Context) (httpserver.Response, error) {
		return h.GetMaterializedViewTasks(ctx, input)
	})
}

const (
	materializedViewActionPause  = "pause"
	materializedViewActionResume = "resume"
)

type MaterializedViewControlInput struct {
	Database string `uri:"database" binding:"required"`
	ID       string `uri:"id" binding:"required,numeric"`
	Action   string `uri:"action" binding:"required,oneof=pause resume"`
}

func (h *HandlerMaterializedViews) ControlMaterializedView(ctx context.Context, input *MaterializedViewControlInput) (httpserver.Response, error) {
	if input.Action != materializedViewActionPause && input.Action != materializedViewActionResume {
		return nil, httpserver.NewErrorWithStatus(http.StatusBadRequest, errors.New("unsupported materialized view action"))
	}

	h.controlMu.Lock()
	defer h.controlMu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	views := make([]MaterializedView, 0)
	query := `SELECT TABLE_NAME AS name, COALESCE(REFRESH_TYPE, '') AS refresh_type,
 COALESCE(IS_ACTIVE, '') AS active, COALESCE(REFRESH_TRIGGER, '') AS refresh_trigger,
 COALESCE(REFRESH_POLICY, '') AS refresh_policy FROM information_schema.materialized_views
 WHERE TABLE_SCHEMA = ? AND MATERIALIZED_VIEW_ID = ?`

	if err := h.client.Select(ctx, &views, query, input.Database, input.ID); err != nil {
		return nil, fmt.Errorf("could not find materialized view: %w", err)
	}

	if len(views) == 0 {
		return nil, httpserver.NewErrorWithStatus(http.StatusNotFound, errors.New("materialized view not found"))
	}

	view := views[0]
	var clause string
	var err error

	if clause, err = h.refreshControlClause(ctx, input, view); err != nil {
		return nil, err
	}

	identifier := "`" + strings.ReplaceAll(input.Database, "`", "``") + "`.`" + strings.ReplaceAll(view.Name, "`", "``") + "`"
	if _, err := h.client.Exec(ctx, "ALTER MATERIALIZED VIEW "+identifier+" "+clause); err != nil {
		return nil, fmt.Errorf("could not %s materialized view refresh: %w", input.Action, err)
	}

	if input.Action == materializedViewActionResume && clause != "ACTIVE" {
		if _, err := h.state.Exec(ctx, "DELETE FROM materialized_view_refresh_policies WHERE policy_key = ?", h.refreshPolicyKey(input.Database, input.ID)); err != nil {
			return nil, fmt.Errorf("automatic refresh was restored, but saved policy cleanup failed: %w", err)
		}
	}

	return httpserver.NewJsonResponse(map[string]string{"action": input.Action}), nil
}

func (h *HandlerMaterializedViews) refreshControlClause(ctx context.Context, input *MaterializedViewControlInput, view MaterializedView) (string, error) {
	manual := strings.EqualFold(view.RefreshTrigger, "MANUAL")
	if !strings.EqualFold(view.RefreshType, "ASYNC") {
		return "", httpserver.NewErrorWithStatus(http.StatusConflict, errors.New("refresh control requires an asynchronous materialized view"))
	}

	if input.Action == materializedViewActionResume && strings.EqualFold(view.Active, "false") && !manual {
		return "ACTIVE", nil
	}

	if !strings.EqualFold(view.Active, "true") || (input.Action == materializedViewActionPause && manual) || (input.Action == materializedViewActionResume && !manual) {
		return "", httpserver.NewErrorWithStatus(http.StatusConflict, fmt.Errorf("cannot %s this view; reload its activation and refresh mode", input.Action))
	}

	if input.Action == materializedViewActionPause {
		if _, err := h.automaticRefreshClause(view.RefreshPolicy); err != nil {
			return "", httpserver.NewErrorWithStatus(http.StatusConflict, err)
		}

		if err := h.saveRefreshPolicy(ctx, input.Database, input.ID, view.RefreshPolicy); err != nil {
			return "", fmt.Errorf("could not save automatic refresh policy; view was not changed: %w", err)
		}

		return "REFRESH MANUAL", nil
	}

	var policy string
	var err error

	if policy, err = h.loadRefreshPolicy(ctx, input.Database, input.ID); err != nil {
		return "", httpserver.NewErrorWithStatus(http.StatusConflict, fmt.Errorf("original automatic refresh policy is unavailable; view remains manual: %w", err))
	}

	return h.automaticRefreshClause(policy)
}

func (h *HandlerMaterializedViews) refreshPolicyKey(database, id string) string {
	key := sha256.Sum256([]byte(database + "\x00" + id))

	return fmt.Sprintf("%x", key)
}

func (h *HandlerMaterializedViews) saveRefreshPolicy(ctx context.Context, database, id, policy string) error {
	_, err := h.state.Exec(ctx, `INSERT INTO materialized_view_refresh_policies (policy_key, policy) VALUES (?, ?) ON CONFLICT(policy_key) DO UPDATE SET policy = excluded.policy`, h.refreshPolicyKey(database, id), policy)

	return err
}

func (h *HandlerMaterializedViews) loadRefreshPolicy(ctx context.Context, database, id string) (string, error) {
	var policy string
	if err := h.state.Get(ctx, &policy, "SELECT policy FROM materialized_view_refresh_policies WHERE policy_key = ?", h.refreshPolicyKey(database, id)); err != nil {
		return "", err
	}

	return policy, nil
}

var scheduledRefreshPolicyPattern = regexp.MustCompile(`^(?:START\(["']([0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2})["']\)\s*)?EVERY\(INTERVAL ([0-9]+) (SECOND|MINUTE|HOUR|DAY)\)$`)

func (h *HandlerMaterializedViews) automaticRefreshClause(policy string) (string, error) {
	if policy == "ON_BASE_TABLE_CHANGE" {
		return "REFRESH ASYNC", nil
	}

	parts := scheduledRefreshPolicyPattern.FindStringSubmatch(policy)
	if parts == nil {
		return "", errors.New("automatic refresh policy is unsupported; refusing to change the view")
	}

	interval, err := strconv.ParseUint(parts[2], 10, 63)
	if err != nil || interval == 0 {
		return "", errors.New("invalid automatic refresh interval")
	}

	clause := "REFRESH ASYNC"
	if parts[1] != "" {
		if _, err := time.Parse(time.DateTime, parts[1]); err != nil {
			return "", fmt.Errorf("invalid automatic refresh start time: %w", err)
		}

		clause += " START ('" + parts[1] + "')"
	}

	return clause + fmt.Sprintf(" EVERY(INTERVAL %d %s)", interval, parts[3]), nil
}

// MaterializedViewRefreshInput includes both calendar dates in the cluster partition key's time zone.
type MaterializedViewRefreshInput struct {
	Database  string `uri:"database" json:"-"`
	ID        string `uri:"id" json:"-"`
	StartDate string `json:"startDate" binding:"required"`
	EndDate   string `json:"endDate" binding:"required"`
}

func (h *HandlerMaterializedViews) RefreshMaterializedView(ctx context.Context, input *MaterializedViewRefreshInput) (httpserver.Response, error) {
	id, idErr := strconv.ParseUint(input.ID, 10, 64)
	start, startErr := time.Parse(time.DateOnly, input.StartDate)
	end, endErr := time.Parse(time.DateOnly, input.EndDate)
	// StarRocks range bounds and LIST filtering use an exclusive upper bound.
	end = end.AddDate(0, 0, 1)

	if input.Database == "" || idErr != nil || id == 0 || startErr != nil || endErr != nil || start.Year() < 1 || end.Year() > 9999 || !start.Before(end) {
		return nil, httpserver.NewErrorWithStatus(http.StatusBadRequest, errors.New("provide a database, numeric view ID, and valid supported YYYY-MM-DD dates with start on or before end (both included)"))
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	views := make([]MaterializedView, 0)
	query := `SELECT TABLE_NAME AS name, COALESCE(REFRESH_TYPE, '') AS refresh_type,
 COALESCE(IS_ACTIVE, '') AS active, COALESCE(PARTITION_TYPE, '') AS partition_type
 FROM information_schema.materialized_views WHERE TABLE_SCHEMA = ? AND MATERIALIZED_VIEW_ID = ?`

	if err := h.client.Select(ctx, &views, query, input.Database, input.ID); err != nil {
		return nil, fmt.Errorf("could not find materialized view: %w", err)
	}

	if len(views) == 0 {
		return nil, httpserver.NewErrorWithStatus(http.StatusNotFound, errors.New("materialized view not found"))
	}

	view := views[0]
	if !strings.EqualFold(view.RefreshType, "ASYNC") || !strings.EqualFold(view.Active, "true") || (!strings.EqualFold(view.PartitionType, "RANGE") && !strings.EqualFold(view.PartitionType, "LIST")) {
		return nil, httpserver.NewErrorWithStatus(http.StatusConflict, errors.New("forced date-range refresh requires an active asynchronous materialized view with date partitions"))
	}

	// Identifiers are quoted; date literals are parsed and formatted before interpolation.
	identifier := "`" + strings.ReplaceAll(input.Database, "`", "``") + "`.`" + strings.ReplaceAll(view.Name, "`", "``") + "`"
	clause := fmt.Sprintf("START ('%s') END ('%s')", start.Format(time.DateOnly), end.Format(time.DateOnly))

	if strings.EqualFold(view.PartitionType, "LIST") {
		var err error
		if clause, err = h.listPartitionRefreshClause(ctx, identifier, start, end); err != nil {
			return nil, err
		}
	}

	query = "REFRESH MATERIALIZED VIEW " + identifier + " PARTITION " + clause + " FORCE WITH ASYNC MODE"

	if _, err := h.client.Exec(ctx, query); err != nil {
		return nil, fmt.Errorf("could not submit forced materialized view refresh: %w", err)
	}

	return httpserver.NewJsonResponse(map[string]string{"status": "submitted"}), nil
}

func (h *HandlerMaterializedViews) listPartitionRefreshClause(ctx context.Context, identifier string, start, end time.Time) (clause string, err error) {
	var rows *sqlc.Rows
	var columns []string

	if rows, err = h.client.Query(ctx, "SHOW PARTITIONS FROM "+identifier); err != nil {
		return "", fmt.Errorf("could not list materialized view partitions: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			clause = ""
			err = fmt.Errorf("could not close partition metadata: %w", closeErr)
		}
	}()

	if columns, err = rows.Columns(); err != nil {
		return "", fmt.Errorf("could not read partition columns: %w", err)
	}

	listIndex := slices.Index(columns, "List")
	if listIndex < 0 {
		return "", errors.New("partition metadata does not expose LIST values")
	}

	values := make([]string, 0)
	for rows.Next() {
		fields := make([]sql.NullString, len(columns))
		destinations := make([]any, len(columns))

		for i := range fields {
			destinations[i] = &fields[i]
		}

		if err = rows.Scan(destinations...); err != nil {
			return "", fmt.Errorf("could not read partition values: %w", err)
		}

		var selected []string
		if selected, err = h.datePartitionValues(fields[listIndex].String, start, end); err != nil {
			return "", httpserver.NewErrorWithStatus(http.StatusConflict, err)
		}

		values = append(values, selected...)
	}

	if err = rows.Err(); err != nil {
		return "", fmt.Errorf("could not finish reading partition values: %w", err)
	}

	if len(values) == 0 {
		return "", httpserver.NewErrorWithStatus(http.StatusBadRequest, errors.New("no existing date partitions match the selected range"))
	}

	slices.Sort(values)
	values = slices.Compact(values)

	return "(" + strings.Join(values, ", ") + ")", nil
}

func (h *HandlerMaterializedViews) datePartitionValues(encoded string, start, end time.Time) ([]string, error) {
	var tuples [][]string
	var value time.Time
	var err error

	if err := json.Unmarshal([]byte(encoded), &tuples); err != nil || len(tuples) == 0 {
		return nil, errors.New("date-range refresh requires readable single-column DATE or DATETIME partition values")
	}

	values := make([]string, 0)
	for _, tuple := range tuples {
		if len(tuple) != 1 {
			return nil, errors.New("date-range refresh does not support multiple partition columns")
		}

		layout := time.DateOnly
		if value, err = time.Parse(layout, tuple[0]); err != nil {
			layout = "2006-01-02 15:04:05"
			value, err = time.Parse(layout, tuple[0])
		}

		if err != nil || value.Year() < 1 {
			return nil, errors.New("date-range refresh requires DATE or DATETIME partition values")
		}

		if !value.Before(start) && value.Before(end) {
			values = append(values, "'"+value.Format(layout)+"'")
		}
	}

	return values, nil
}
