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

type tablesClient struct {
	sqlc.Client
	query string
	err   error
	empty bool
}

func (c *tablesClient) Select(ctx context.Context, dest any, query string, _ ...any) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("metadata read must have a deadline")
	}

	c.query = query
	if c.err != nil {
		return c.err
	}

	if c.empty {
		return nil
	}

	switch rows := dest.(type) {
	case *[]Table:
		*rows = append(*rows, Table{Database: "analytics", Name: "events"})
	case *[]RoutineLoad:
		*rows = append(*rows, RoutineLoad{ID: "9007199254740993", Database: "analytics", Table: "events", State: "PAUSED", Reason: "too many error rows", ErrorRows: "10", TaskConsumeSecond: "30", Progress: `{"0":"9007199254740993"}`})
	}

	return nil
}

func TestTablesMetadata(t *testing.T) {
	client := &tablesClient{}
	handler := &HandlerTables{client: client}
	var response httpserver.Response
	var body []byte
	var err error

	if response, err = handler.GetTables(t.Context()); err != nil {
		t.Fatal(err)
	}

	if body, err = response.Body(); err != nil {
		t.Fatal(err)
	}

	var result TablesResponse
	if err = json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}

	if len(result.Tables) != 1 || result.Tables[0].Rows != nil || result.Tables[0].DataBytes != nil {
		t.Fatalf("unknown table statistics must stay null: %s", body)
	}

	if !strings.Contains(client.query, "mv.MATERIALIZED_VIEW_ID IS NULL") || !strings.Contains(client.query, "'BASE TABLE', 'TABLE'") {
		t.Fatalf("table query must exclude views and support both table type names: %s", client.query)
	}
}

func TestRoutineLoadMetadata(t *testing.T) {
	client := &tablesClient{}
	handler := &HandlerTables{client: client}
	var response httpserver.Response
	var body []byte
	var err error

	if response, err = handler.GetRoutineLoads(t.Context()); err != nil {
		t.Fatal(err)
	}

	if body, err = response.Body(); err != nil {
		t.Fatal(err)
	}

	var result RoutineLoadsResponse
	if err = json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}

	if len(result.Jobs) != 1 || result.Jobs[0].ID != "9007199254740993" || result.Jobs[0].State != "PAUSED" || result.Jobs[0].Reason != "too many error rows" || result.Jobs[0].ErrorRows != "10" || result.Jobs[0].TaskConsumeSecond != "30" || result.Jobs[0].Progress != `{"0":"9007199254740993"}` {
		t.Fatalf("lost ingestion identity, errors or offset precision: %s", body)
	}

	if strings.Contains(client.query, "CUSTOM_PROPERTIES") || !strings.Contains(client.query, "get_json_string(DATA_SOURCE_PROPERTIES, '$.topic')") || !strings.Contains(client.query, "get_json_string(JOB_PROPERTIES, '$.taskConsumeSecond')") {
		t.Fatalf("only select the topic, not Kafka connection properties: %s", client.query)
	}
}

func TestTablesMetadataErrors(t *testing.T) {
	queryError := errors.New("metadata unavailable")
	handler := &HandlerTables{client: &tablesClient{err: queryError}}

	for _, get := range []func(context.Context) (httpserver.Response, error){handler.GetTables, handler.GetRoutineLoads} {
		response, err := get(t.Context())
		if response != nil || !errors.Is(err, queryError) {
			t.Fatalf("expected wrapped query error and no response: %v, %v", response, err)
		}
	}
}

func TestTablesEmptyResponses(t *testing.T) {
	handler := &HandlerTables{client: &tablesClient{empty: true}}

	for _, get := range []func(context.Context) (httpserver.Response, error){handler.GetTables, handler.GetRoutineLoads} {
		var response httpserver.Response
		var body []byte
		var err error

		if response, err = get(t.Context()); err != nil {
			t.Fatal(err)
		}

		if body, err = response.Body(); err != nil {
			t.Fatal(err)
		}

		if string(body) != `{"tables":[]}` && string(body) != `{"jobs":[]}` {
			t.Fatalf("empty metadata must return an array: %s", body)
		}
	}
}
