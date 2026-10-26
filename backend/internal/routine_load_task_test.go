package internal

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gosoline-project/httpserver"
	"github.com/gosoline-project/sqlc"
)

func TestRoutineLoadTaskMetadata(t *testing.T) {
	handler := &HandlerTables{}
	metadata := map[string]string{
		"TaskId": "task-1", "JobId": "9007199254740993", "TxnId": "9007199254740994", "TxnStatus": "UNKNOWN",
		"CreateTime": "2026-10-24 14:51:38", "LastScheduledTime": "2026-10-24 14:51:48", "ExecuteStartTime": "NULL",
		"Timeout": "60", "BeId": "10002", "DataSourceProperties": `Progress:{"0":9007199254740993},LatestOffset:null`,
		"Message": "task submitted to execute", "Warehouse": "default_warehouse", "FutureColumn": "ignored",
	}

	task := handler.routineLoadTaskFromMetadata(metadata)
	if task.ID != "task-1" || task.JobID != metadata["JobId"] || task.TransactionID != metadata["TxnId"] || task.Progress != metadata["DataSourceProperties"] {
		t.Fatalf("lost task identity or offset precision: %+v", task)
	}

	if task.TransactionStatus != "UNKNOWN" || task.NodeID != "10002" || task.Timeout != "60" || task.Created != metadata["CreateTime"] || task.LastScheduled != metadata["LastScheduledTime"] || task.ExecuteStarted != "" || task.Message != metadata["Message"] || task.Warehouse != metadata["Warehouse"] {
		t.Fatalf("lost task execution metadata: %+v", task)
	}

	if handler.routineLoadTaskFromMetadata(nil) != (RoutineLoadTask{}) {
		t.Fatal("missing task metadata must remain unknown")
	}
}

func TestRoutineLoadTasksQueryEscaping(t *testing.T) {
	handler := &HandlerTables{}

	query := handler.routineLoadTasksQuery("db`name", `job'\name`)
	if query != "SHOW ROUTINE LOAD TASK FROM `db``name` WHERE JobName = 'job\\'\\\\name'" {
		t.Fatalf("unsafe SHOW query: %s", query)
	}
}

func TestRoutineLoadTasksMissingJob(t *testing.T) {
	client := &tablesClient{empty: true}
	handler := &HandlerTables{client: client}
	var response httpserver.Response
	var body []byte
	var err error

	if response, err = handler.GetRoutineLoadTasks(t.Context(), &RoutineLoadTasksInput{Database: "analytics", ID: "9007199254740993"}); err != nil {
		t.Fatal(err)
	}

	if body, err = response.Body(); err != nil {
		t.Fatal(err)
	}

	var result RoutineLoadTasksResponse
	if err = json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}

	if result.Tasks == nil || len(result.Tasks) != 0 || !strings.Contains(client.query, "DB_NAME = ? AND ID = ? AND DATA_SOURCE_TYPE = 'KAFKA'") {
		t.Fatalf("missing job must return empty tasks and bind identity: %s, %s", body, client.query)
	}
}

type routineTasksQueryClient struct {
	tablesClient
	taskError error
}

func (c *routineTasksQueryClient) Query(ctx context.Context, _ string, _ ...any) (*sqlc.Rows, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("missing task query deadline")
	}

	return nil, c.taskError
}

func TestRoutineLoadTasksErrors(t *testing.T) {
	failure := errors.New("metadata unavailable")
	for _, client := range []sqlc.Client{
		&tablesClient{err: failure}, &routineTasksQueryClient{taskError: failure},
	} {
		handler := &HandlerTables{client: client}

		response, err := handler.GetRoutineLoadTasks(t.Context(), &RoutineLoadTasksInput{Database: "analytics", ID: "1"})
		if response != nil || !errors.Is(err, failure) {
			t.Fatalf("task reads must preserve errors: %v, %v", response, err)
		}
	}
}
