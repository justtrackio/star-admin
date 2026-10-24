package internal

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/gosoline-project/httpserver"
	"github.com/gosoline-project/sqlc"
)

type materializedViewTasksClient struct {
	sqlc.Client
	args []any
	err  error
}

func (c *materializedViewTasksClient) Select(ctx context.Context, _ any, _ string, args ...any) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("metadata read must have a deadline")
	}

	c.args = args

	return c.err
}

func TestMaterializedViewTasksScopeAndEmptyResponse(t *testing.T) {
	var response httpserver.Response
	var body []byte
	var err error
	var result MaterializedViewTasksResponse

	client := &materializedViewTasksClient{}
	handler := &HandlerMaterializedViews{client: client}

	if response, err = handler.GetMaterializedViewTasks(t.Context(), &MaterializedViewTasksInput{Database: "analytics'", ID: "2019067"}); err != nil {
		t.Fatal(err)
	}

	if len(client.args) != 2 || client.args[0] != "analytics'" || client.args[1] != "mv-2019067" {
		t.Fatalf("expected bound database and view task name, got %v", client.args)
	}

	if body, err = response.Body(); err != nil {
		t.Fatal(err)
	}

	if err = json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}

	if result.Tasks == nil || len(result.Tasks) != 0 {
		t.Fatalf("expected empty tasks array, got %s", body)
	}
}

func TestMaterializedViewTasksQueryError(t *testing.T) {
	queryError := errors.New("metadata unavailable")
	handler := &HandlerMaterializedViews{client: &materializedViewTasksClient{err: queryError}}

	response, err := handler.GetMaterializedViewTasks(t.Context(), &MaterializedViewTasksInput{Database: "analytics", ID: "2019067"})
	if response != nil || !errors.Is(err, queryError) {
		t.Fatalf("expected wrapped query error, got %v, %v", response, err)
	}
}

func TestTaskRefreshMetadata(t *testing.T) {
	created := int64(1000)
	finished := int64(7000)

	for _, tc := range []struct {
		name          string
		state         string
		extra         string
		finished      *int64
		queue         float64
		processing    float64
		hasQueue      bool
		hasProcessing bool
		partitions    int
	}{
		{name: "completed", state: "SUCCESS", extra: `{"processStartTime":3000,"mvPartitionsToRefresh":["p1","p2"]}`, finished: &finished, queue: 2, processing: 4, hasQueue: true, hasProcessing: true, partitions: 2},
		{name: "pending", state: "PENDING", extra: `{}`, queue: 9, hasQueue: true},
		{name: "running", state: "RUNNING", extra: `{"processStartTime":3000}`, queue: 2, processing: 7, hasQueue: true, hasProcessing: true},
		{name: "missing metadata", state: "SUCCESS", extra: "", finished: &finished},
		{name: "invalid metadata", state: "SUCCESS", extra: "invalid", finished: &finished},
		{name: "invalid timing", state: "SUCCESS", extra: `{"processStartTime":500}`, finished: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			task := MaterializedViewTask{State: tc.state, ExtraMessage: tc.extra, CreatedMillis: &created, FinishedMillis: tc.finished, ObservedMillis: 10000}
			task.enrich()

			if (task.QueueTime != nil) != tc.hasQueue || (task.ProcessingTime != nil) != tc.hasProcessing || len(task.Partitions) != tc.partitions {
				t.Fatalf("unexpected metadata: %+v", task)
			}

			if task.QueueTime != nil && *task.QueueTime != tc.queue {
				t.Fatalf("queue time: got %v want %v", *task.QueueTime, tc.queue)
			}

			if task.ProcessingTime != nil && *task.ProcessingTime != tc.processing {
				t.Fatalf("processing time: got %v want %v", *task.ProcessingTime, tc.processing)
			}
		})
	}
}
