package internal

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/gosoline-project/httpserver"
	"github.com/gosoline-project/sqlc"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// ComputeNode contains the metadata reported by SHOW COMPUTE NODES.
type ComputeNode struct {
	ID             string `json:"id"`
	Host           string `json:"host"`
	Alive          *bool  `json:"alive"`
	Decommissioned bool   `json:"decommissioned"`
	Status         string `json:"status"`
	Warehouse      string `json:"warehouse"`
	CPUCores       *int64 `json:"cpuCores"`
	MemoryLimit    string `json:"memoryLimit"`
	CPUUsed        string `json:"cpuUsed"`
	MemoryUsed     string `json:"memoryUsed"`
	RunningQueries *int64 `json:"runningQueries"`
	Tablets        *int64 `json:"tablets"`
	LastStartTime  string `json:"lastStartTime"`
	LastHeartbeat  string `json:"lastHeartbeat"`
	Version        string `json:"version"`
	Error          string `json:"error"`
	DataCache      string `json:"dataCache"`
}

type OverviewResponse struct {
	Nodes []ComputeNode `json:"nodes"`
}

type HandlerOverview struct {
	client sqlc.Client
}

func NewHandlerOverview(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerOverview, error) {
	var client sqlc.Client
	var err error

	if client, err = ProvideStarRocksClient(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not create overview handler: %w", err)
	}

	return &HandlerOverview{client: client}, nil
}

func (h *HandlerOverview) GetOverview(ctx context.Context) (response httpserver.Response, err error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var rows *sqlc.Rows
	var columns []string

	if rows, err = h.client.Query(ctx, "SHOW COMPUTE NODES"); err != nil {
		return nil, fmt.Errorf("could not list compute nodes: %w", err)
	}

	defer func() {
		var closeErr error
		if closeErr = rows.Close(); closeErr != nil && err == nil {
			response = nil
			err = fmt.Errorf("could not close compute node rows: %w", closeErr)
		}
	}()

	if columns, err = rows.Columns(); err != nil {
		return nil, fmt.Errorf("could not read compute node columns: %w", err)
	}

	nodes := make([]ComputeNode, 0)

	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		destinations := make([]any, len(columns))

		for i := range values {
			destinations[i] = &values[i]
		}

		if err = rows.Scan(destinations...); err != nil {
			return nil, fmt.Errorf("could not read compute node: %w", err)
		}

		metadata := make(map[string]string, len(columns))

		for i, column := range columns {
			metadata[column] = values[i].String
		}

		nodes = append(nodes, h.computeNodeFromMetadata(metadata))
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate compute nodes: %w", err)
	}

	return httpserver.NewJsonResponse(OverviewResponse{Nodes: nodes}), nil
}

// Column names vary between StarRocks releases. Missing values remain unknown.
func (h *HandlerOverview) computeNodeFromMetadata(m map[string]string) ComputeNode {
	var alive *bool
	if value, err := strconv.ParseBool(m["Alive"]); err == nil {
		alive = &value
	}

	return ComputeNode{
		ID: m["ComputeNodeId"], Host: m["IP"], Alive: alive,
		Decommissioned: m["SystemDecommissioned"] == "true" || m["ClusterDecommissioned"] == "true",
		Status:         m["StatusCode"], Warehouse: m["WarehouseName"],
		CPUCores: h.nodeInteger(m["CpuCores"]), MemoryLimit: m["MemLimit"],
		CPUUsed: m["CpuUsedPct"], MemoryUsed: m["MemUsedPct"],
		RunningQueries: h.nodeInteger(m["NumRunningQueries"]), Tablets: h.nodeInteger(m["TabletNum"]),
		LastStartTime: m["LastStartTime"], LastHeartbeat: m["LastHeartbeat"],
		Version: m["Version"], Error: m["ErrMsg"], DataCache: m["DataCacheMetrics"],
	}
}

func (h *HandlerOverview) nodeInteger(value string) *int64 {
	var number int64
	var err error

	if number, err = strconv.ParseInt(value, 10, 64); err != nil {
		return nil
	}

	return &number
}
