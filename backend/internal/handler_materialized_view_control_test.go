package internal

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/gosoline-project/httpserver"
	"github.com/gosoline-project/sqlc"
)

type materializedViewControlClient struct {
	materializedViewsClient
	command string
	execErr error
}

func (c *materializedViewControlClient) Exec(ctx context.Context, query string, _ ...any) (sqlc.Result, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("missing command deadline")
	}

	c.command = query

	return nil, c.execErr
}

func TestMaterializedViewControl(t *testing.T) {
	for _, test := range []struct{ action, active, state string }{
		{"pause", "true", "INACTIVE"}, {"resume", "false", "ACTIVE"},
	} {
		t.Run(test.action, func(t *testing.T) {
			client := &materializedViewControlClient{materializedViewsClient: materializedViewsClient{
				views: []MaterializedView{{Name: "view`name", RefreshType: "ASYNC", Active: test.active}},
			}}

			handler := &HandlerMaterializedViews{client: client}

			response, err := handler.ControlMaterializedView(t.Context(), &MaterializedViewControlInput{Database: "db`name", ID: "9007199254740993", Action: test.action})
			if err != nil || response == nil {
				t.Fatalf("expected successful command: %v, %v", response, err)
			}

			if client.command != "ALTER MATERIALIZED VIEW `db``name`.`view``name` "+test.state {
				t.Fatalf("unexpected command: %s", client.command)
			}

			if !strings.Contains(client.query, "WHERE TABLE_SCHEMA = ? AND MATERIALIZED_VIEW_ID = ?") ||
				!reflect.DeepEqual(client.args, []any{"db`name", "9007199254740993"}) {
				t.Fatalf("expected parameterized lookup: %s %v", client.query, client.args)
			}

			body, err := response.Body()
			if err != nil || string(body) != `{"action":"`+test.action+`"}` {
				t.Fatalf("unexpected response: %s, %v", body, err)
			}
		})
	}
}

func TestMaterializedViewControlRejectsInvalidTransitions(t *testing.T) {
	for _, test := range []struct {
		action, refreshType, active string
		empty                       bool
		status                      int
	}{
		{action: "drop", status: 400},
		{action: "pause", empty: true, status: 404},
		{action: "pause", refreshType: "SYNC", active: "true", status: 409},
		{action: "resume", refreshType: "SYNC", active: "false", status: 409},
		{action: "pause", refreshType: "ASYNC", active: "false", status: 409},
		{action: "resume", refreshType: "ASYNC", active: "true", status: 409},
		{action: "resume", refreshType: "ASYNC", active: "", status: 409},
	} {
		t.Run(test.action+"_"+test.refreshType+"_"+test.active, func(t *testing.T) {
			client := &materializedViewControlClient{}
			if !test.empty {
				client.views = []MaterializedView{{Name: "view", RefreshType: test.refreshType, Active: test.active}}
			}

			handler := &HandlerMaterializedViews{client: client}

			response, err := handler.ControlMaterializedView(t.Context(), &MaterializedViewControlInput{Database: "analytics", ID: "1", Action: test.action})
			if response != nil || err == nil || httpserver.GetErrorStatusCode(err) != test.status || client.command != "" {
				t.Fatalf("invalid transition must not execute: %v, %v, %s", response, err, client.command)
			}
		})
	}
}

func TestMaterializedViewControlErrors(t *testing.T) {
	failure := errors.New("permission denied")
	for _, client := range []*materializedViewControlClient{
		{materializedViewsClient: materializedViewsClient{err: failure}},
		{materializedViewsClient: materializedViewsClient{views: []MaterializedView{{Name: "view", RefreshType: "ASYNC", Active: "false"}}}, execErr: failure},
	} {
		handler := &HandlerMaterializedViews{client: client}

		response, err := handler.ControlMaterializedView(t.Context(), &MaterializedViewControlInput{Database: "analytics", ID: "1", Action: "resume"})
		if response != nil || !errors.Is(err, failure) {
			t.Fatalf("expected wrapped database error: %v, %v", response, err)
		}
	}
}
