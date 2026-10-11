package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/gosoline-project/httpserver"
	"github.com/gosoline-project/sqlc"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// MaterializedView describes a view in the configured cluster's internal catalog.
type MaterializedView struct {
	ID                            string   `json:"id" db:"id"`
	Database                      string   `json:"database" db:"database_name"`
	Name                          string   `json:"name" db:"name"`
	RefreshType                   string   `json:"refreshType" db:"refresh_type"`
	RefreshTrigger                string   `json:"refreshTrigger" db:"refresh_trigger"`
	RefreshPolicy                 string   `json:"refreshPolicy" db:"refresh_policy"`
	Active                        string   `json:"active" db:"active"`
	InactiveReason                string   `json:"inactiveReason" db:"inactive_reason"`
	LastRefreshState              string   `json:"lastRefreshState" db:"last_refresh_state"`
	LastRefreshFinished           string   `json:"lastRefreshFinished" db:"last_refresh_finished"`
	LastSuccessfulRefreshFinished string   `json:"lastSuccessfulRefreshFinished" db:"last_successful_refresh_finished"`
	LastRefreshDuration           *float64 `json:"lastRefreshDuration" db:"last_refresh_duration"`
	LastRefreshError              string   `json:"lastRefreshError" db:"last_refresh_error"`
	Rows                          *int64   `json:"rows" db:"table_rows"`
	Definition                    string   `json:"definition" db:"definition"`
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

const materializedViewsQuery = `SELECT
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
ORDER BY mv.TABLE_SCHEMA, mv.TABLE_NAME, mv.MATERIALIZED_VIEW_ID`

// GetMaterializedViews performs only a metadata read, without refreshing any views.
func (h *HandlerMaterializedViews) GetMaterializedViews(ctx context.Context) (httpserver.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	views := make([]MaterializedView, 0)
	if err := h.client.Select(ctx, &views, materializedViewsQuery); err != nil {
		return nil, fmt.Errorf("could not list materialized views: %w", err)
	}

	return httpserver.NewJsonResponse(struct {
		Views []MaterializedView `json:"views"`
	}{Views: views}), nil
}
