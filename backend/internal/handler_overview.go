package internal

import (
	"context"
	"database/sql"
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

// ClusterNode contains read-only metadata reported by the node SHOW commands.
type ClusterNode struct {
	ID             string            `json:"id"`
	Type           string            `json:"type"`
	Role           string            `json:"role"`
	Details        map[string]string `json:"details"`
	Host           string            `json:"host"`
	Alive          *bool             `json:"alive"`
	Decommissioned bool              `json:"decommissioned"`
	Status         string            `json:"status"`
	Warehouse      string            `json:"warehouse"`
	CPUCores       *int64            `json:"cpuCores"`
	MemoryLimit    string            `json:"memoryLimit"`
	CPUUsed        string            `json:"cpuUsed"`
	MemoryUsed     string            `json:"memoryUsed"`
	RunningQueries *int64            `json:"runningQueries"`
	Tablets        *int64            `json:"tablets"`
	LastStartTime  string            `json:"lastStartTime"`
	LastHeartbeat  string            `json:"lastHeartbeat"`
	Version        string            `json:"version"`
	Error          string            `json:"error"`
	DataCache      string            `json:"dataCache"`
}

type OverviewResponse struct {
	Nodes []ClusterNode `json:"nodes"`
}

type nodeSource struct {
	kind  string
	query string
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
	var nodes []ClusterNode
	if nodes, err = h.readNodes(ctx, "compute", "SHOW COMPUTE NODES"); err != nil {
		return nil, err
	}

	return httpserver.NewJsonResponse(OverviewResponse{Nodes: nodes}), nil
}

func (h *HandlerOverview) GetNodes(ctx context.Context) (httpserver.Response, error) {
	nodes := make([]ClusterNode, 0)
	var batch []ClusterNode
	var err error

	for _, source := range []nodeSource{
		{"frontend", "SHOW FRONTENDS"}, {"backend", "SHOW BACKENDS"}, {"compute", "SHOW COMPUTE NODES"},
	} {
		if batch, err = h.readNodes(ctx, source.kind, source.query); err != nil {
			return nil, err
		}

		nodes = append(nodes, batch...)
	}

	var metrics []frontendHeapMetrics
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err = h.client.Select(ctx, &metrics, frontendHeapQuery); err != nil {
		return nil, fmt.Errorf("could not read frontend heap metrics: %w", err)
	}

	h.applyFrontendHeap(nodes, metrics)

	return httpserver.NewJsonResponse(OverviewResponse{Nodes: nodes}), nil
}

func (h *HandlerOverview) WatchNodes(_ context.Context, request *http.Request, writer *httpserver.SseWriter) error {
	return watchSnapshots(request, writer, h.GetNodes)
}

func (h *HandlerOverview) readNodes(ctx context.Context, kind, query string) (nodes []ClusterNode, err error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var rows *sqlc.Rows
	var columns []string

	if rows, err = h.client.Query(ctx, query); err != nil {
		return nil, fmt.Errorf("could not list %s nodes: %w", kind, err)
	}

	defer func() {
		var closeErr error
		if closeErr = rows.Close(); closeErr != nil && err == nil {
			nodes = nil
			err = fmt.Errorf("could not close %s node rows: %w", kind, closeErr)
		}
	}()

	if columns, err = rows.Columns(); err != nil {
		return nil, fmt.Errorf("could not read %s node columns: %w", kind, err)
	}

	nodes = make([]ClusterNode, 0)

	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		destinations := make([]any, len(columns))

		for i := range values {
			destinations[i] = &values[i]
		}

		if err = rows.Scan(destinations...); err != nil {
			return nil, fmt.Errorf("could not read %s node: %w", kind, err)
		}

		metadata := make(map[string]string, len(columns))

		for i, column := range columns {
			metadata[column] = values[i].String
		}

		node := h.nodeFromMetadata(metadata)
		node.Type = kind
		nodes = append(nodes, node)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate %s nodes: %w", kind, err)
	}

	return nodes, nil
}

// Column names vary between StarRocks releases. Missing values remain unknown.
func (h *HandlerOverview) nodeFromMetadata(m map[string]string) ClusterNode {
	var alive *bool
	if value, err := strconv.ParseBool(m["Alive"]); err == nil {
		alive = &value
	}

	id := m["ComputeNodeId"]
	if id == "" {
		id = m["BackendId"]
	}

	if id == "" {
		id = m["Id"]
	}

	if id == "" {
		id = m["Name"]
	}

	host := m["IP"]
	if host == "" {
		host = m["Host"]
	}

	started := m["LastStartTime"]
	if started == "" {
		started = m["StartTime"]
	}

	return ClusterNode{
		ID: id, Host: host, Alive: alive, Role: m["Role"], Details: m,
		Decommissioned: m["SystemDecommissioned"] == "true" || m["ClusterDecommissioned"] == "true",
		Status:         m["StatusCode"], Warehouse: m["WarehouseName"],
		CPUCores: h.nodeInteger(m["CpuCores"]), MemoryLimit: m["MemLimit"],
		CPUUsed: m["CpuUsedPct"], MemoryUsed: m["MemUsedPct"],
		RunningQueries: h.nodeInteger(m["NumRunningQueries"]), Tablets: h.nodeInteger(m["TabletNum"]),
		LastStartTime: started, LastHeartbeat: m["LastHeartbeat"],
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

type frontendHeapMetrics struct {
	Name      string `db:"name"`
	UsedBytes *int64 `db:"used_bytes"`
	MaxBytes  *int64 `db:"max_bytes"`
}

const frontendHeapQuery = `SELECT FE_ID AS name,
 MAX(CASE WHEN get_json_string(LABELS, '$.type') = 'used' THEN VALUE END) AS used_bytes,
 MAX(CASE WHEN get_json_string(LABELS, '$.type') = 'max' THEN VALUE END) AS max_bytes
 FROM information_schema.fe_metrics
 WHERE NAME = 'jvm_heap_size_bytes'
 GROUP BY FE_ID`

func (h *HandlerOverview) applyFrontendHeap(nodes []ClusterNode, metrics []frontendHeapMetrics) {
	byName := make(map[string]frontendHeapMetrics, len(metrics))
	for _, metric := range metrics {
		byName[metric.Name] = metric
	}

	for i := range nodes {
		node := &nodes[i]

		metric, found := byName[node.Details["Name"]]
		if node.Type != "frontend" || !found || metric.UsedBytes == nil || metric.MaxBytes == nil || *metric.UsedBytes < 0 || *metric.MaxBytes <= 0 {
			continue
		}

		node.MemoryUsed = fmt.Sprintf("%.1f %%", 100*float64(*metric.UsedBytes)/float64(*metric.MaxBytes))
		node.MemoryLimit = fmt.Sprintf("%.2f GiB", float64(*metric.MaxBytes)/(1<<30))
		node.Details["JVM heap used (bytes)"] = strconv.FormatInt(*metric.UsedBytes, 10)
		node.Details["JVM heap max (bytes)"] = strconv.FormatInt(*metric.MaxBytes, 10)
	}
}

func (h *HandlerOverview) GetFrontendConfig(ctx context.Context) (response httpserver.Response, err error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var rows *sqlc.Rows
	var columns []string

	if rows, err = h.client.Query(ctx, "ADMIN SHOW FRONTEND CONFIG"); err != nil {
		return nil, fmt.Errorf("could not read frontend configuration: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			response = nil
			err = fmt.Errorf("could not close frontend configuration rows: %w", closeErr)
		}
	}()

	if columns, err = rows.Columns(); err != nil {
		return nil, fmt.Errorf("could not read frontend configuration columns: %w", err)
	}

	if !slices.Contains(columns, "Key") || !slices.Contains(columns, "Value") {
		return nil, errors.New("frontend configuration is missing Key or Value columns")
	}

	settings := make([]map[string]string, 0)
	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		destinations := make([]any, len(columns))

		for i := range values {
			destinations[i] = &values[i]
		}

		if err = rows.Scan(destinations...); err != nil {
			return nil, fmt.Errorf("could not read frontend configuration row: %w", err)
		}

		setting := make(map[string]string, len(columns))
		for i, column := range columns {
			setting[column] = values[i].String
		}

		h.redactFrontendConfig(setting)
		settings = append(settings, setting)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate frontend configuration: %w", err)
	}

	return httpserver.NewJsonResponse(FrontendConfigResponse{Columns: columns, Settings: settings}), nil
}

func (h *HandlerOverview) WatchFrontendConfig(_ context.Context, request *http.Request, writer *httpserver.SseWriter) error {
	return watchSnapshots(request, writer, h.GetFrontendConfig)
}

func (h *HandlerOverview) redactFrontendConfig(setting map[string]string) {
	switch strings.ToLower(setting["Type"]) {
	case "boolean", "int", "long", "double", "float", "short", "byte":
		return
	}

	if setting["Value"] == "" {
		return
	}

	name := strings.ToLower(setting["Key"] + " " + setting["AliasNames"])
	for _, sensitive := range []string{"password", "passwd", "pwd", "secret", "token", "credential", "private_key", "access_key", "keytab"} {
		if strings.Contains(name, sensitive) {
			setting["Value"] = "******"

			return
		}
	}
}
