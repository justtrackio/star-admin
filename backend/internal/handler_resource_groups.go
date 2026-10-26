package internal

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gosoline-project/httpserver"
	"github.com/gosoline-project/sqlc"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// ResourceGroup contains read-only configuration reported by StarRocks.
type ResourceGroup struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Properties  map[string]string `json:"properties"`
	Classifiers []string          `json:"classifiers"`
}

type ResourceGroupsResponse struct {
	Groups []ResourceGroup `json:"groups"`
}

type HandlerResourceGroups struct {
	client sqlc.Client
}

func NewHandlerResourceGroups(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerResourceGroups, error) {
	var client sqlc.Client
	var err error

	if client, err = ProvideStarRocksClient(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not create resource groups handler: %w", err)
	}

	return &HandlerResourceGroups{client: client}, nil
}

func (h *HandlerResourceGroups) GetResourceGroups(ctx context.Context) (response httpserver.Response, err error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var rows *sqlc.Rows
	var columns []string

	if rows, err = h.client.Query(ctx, "SHOW RESOURCE GROUPS ALL"); err != nil {
		return nil, fmt.Errorf("could not list resource groups: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			response = nil
			err = fmt.Errorf("could not close resource group rows: %w", closeErr)
		}
	}()

	if columns, err = rows.Columns(); err != nil {
		return nil, fmt.Errorf("could not read resource group columns: %w", err)
	}

	metadata := make([]map[string]string, 0)

	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		destinations := make([]any, len(columns))

		for i := range values {
			destinations[i] = &values[i]
		}

		if err = rows.Scan(destinations...); err != nil {
			return nil, fmt.Errorf("could not read resource group: %w", err)
		}

		row := make(map[string]string, len(columns))

		for i, column := range columns {
			row[column] = values[i].String
		}

		metadata = append(metadata, row)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate resource groups: %w", err)
	}

	return httpserver.NewJsonResponse(ResourceGroupsResponse{Groups: h.resourceGroupsFromMetadata(metadata)}), nil
}

func (h *HandlerResourceGroups) WatchResourceGroups(_ context.Context, request *http.Request, writer *httpserver.SseWriter) error {
	return watchSnapshots(request, writer, h.GetResourceGroups)
}

func (h *HandlerResourceGroups) resourceGroupsFromMetadata(rows []map[string]string) []ResourceGroup {
	groups := make([]ResourceGroup, 0)
	indices := make(map[string]int)

	for _, row := range rows {
		index, found := indices[row["name"]]
		if !found {
			properties := make(map[string]string)

			for key, value := range row {
				if key != "name" && key != "id" && key != "classifiers" {
					if value == "null" {
						value = ""
					}

					properties[key] = value
				}
			}

			index = len(groups)
			indices[row["name"]] = index
			groups = append(groups, ResourceGroup{ID: row["id"], Name: row["name"], Properties: properties, Classifiers: []string{}})
		}

		// SHOW returns one row per classifier for the same group.
		if classifier := row["classifiers"]; classifier != "" {
			groups[index].Classifiers = append(groups[index].Classifiers, classifier)
		}
	}

	return groups
}
