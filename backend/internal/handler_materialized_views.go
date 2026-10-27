package internal

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gosoline-project/httpserver"
	"github.com/gosoline-project/sqlc"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// MaterializedView describes a view in the configured cluster's internal catalog.
type MaterializedView struct {
	ID                              string   `json:"id" db:"id"`
	Database                        string   `json:"database" db:"database_name"`
	Name                            string   `json:"name" db:"name"`
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
}

func NewHandlerMaterializedViews(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerMaterializedViews, error) {
	var client sqlc.Client
	var err error

	if client, err = ProvideStarRocksClient(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not create materialized views handler: %w", err)
	}

	return &HandlerMaterializedViews{client: client}, nil
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

type MaterializedViewControlInput struct {
	Database string `uri:"database" binding:"required"`
	ID       string `uri:"id" binding:"required,numeric"`
	Action   string `uri:"action" binding:"required,oneof=pause resume"`
}

func (h *HandlerMaterializedViews) ControlMaterializedView(ctx context.Context, input *MaterializedViewControlInput) (httpserver.Response, error) {
	if input.Action != "pause" && input.Action != "resume" {
		return nil, httpserver.NewErrorWithStatus(http.StatusBadRequest, errors.New("unsupported materialized view action"))
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	views := make([]MaterializedView, 0)
	query := `SELECT TABLE_NAME AS name, COALESCE(REFRESH_TYPE, '') AS refresh_type,
 COALESCE(IS_ACTIVE, '') AS active FROM information_schema.materialized_views
 WHERE TABLE_SCHEMA = ? AND MATERIALIZED_VIEW_ID = ?`

	if err := h.client.Select(ctx, &views, query, input.Database, input.ID); err != nil {
		return nil, fmt.Errorf("could not find materialized view: %w", err)
	}

	if len(views) == 0 {
		return nil, httpserver.NewErrorWithStatus(http.StatusNotFound, errors.New("materialized view not found"))
	}

	view := views[0]
	expectedActive := "false"
	state := "ACTIVE"

	if input.Action == "pause" {
		expectedActive = "true"
		state = "INACTIVE"
	}

	if !strings.EqualFold(view.RefreshType, "ASYNC") || !strings.EqualFold(view.Active, expectedActive) {
		return nil, httpserver.NewErrorWithStatus(http.StatusConflict, fmt.Errorf("cannot %s this materialized view; reload it and check its refresh type and activation status", input.Action))
	}

	// ALTER requires identifiers, which cannot use query parameters.
	identifier := "`" + strings.ReplaceAll(input.Database, "`", "``") + "`.`" + strings.ReplaceAll(view.Name, "`", "``") + "`"
	if _, err := h.client.Exec(ctx, "ALTER MATERIALIZED VIEW "+identifier+" "+state); err != nil {
		return nil, fmt.Errorf("could not %s materialized view refresh: %w", input.Action, err)
	}

	return httpserver.NewJsonResponse(map[string]string{"action": input.Action}), nil
}
