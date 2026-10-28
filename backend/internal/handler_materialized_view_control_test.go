package internal

import (
	"context"
	"database/sql"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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
	for _, test := range []struct{ action, active, trigger, policy, expected string }{
		{"pause", "true", "SCHEDULED", `START("2026-08-23 00:00:00") EVERY(INTERVAL 5 MINUTE)`, "REFRESH MANUAL"},
		{"pause", "true", "ON_BASE_TABLE_CHANGE", "ON_BASE_TABLE_CHANGE", "REFRESH MANUAL"},
		{"resume", "true", "MANUAL", `START("2026-08-23 00:00:00") EVERY(INTERVAL 5 MINUTE)`, "REFRESH ASYNC START ('2026-08-23 00:00:00') EVERY(INTERVAL 5 MINUTE)"},
		{"resume", "true", "MANUAL", "ON_BASE_TABLE_CHANGE", "REFRESH ASYNC"},
		{"resume", "false", "SCHEDULED", "EVERY(INTERVAL 5 MINUTE)", "ACTIVE"},
	} {
		t.Run(test.action+"_"+test.trigger, func(t *testing.T) {
			client := &materializedViewControlClient{materializedViewsClient: materializedViewsClient{
				views: []MaterializedView{{Name: "view`name", RefreshType: "ASYNC", Active: test.active, RefreshTrigger: test.trigger, RefreshPolicy: test.policy}},
			}}
			handler := &HandlerMaterializedViews{client: client, state: testLocalState(t, filepath.Join(t.TempDir(), "state.sqlite"))}

			if err := handler.saveRefreshPolicy(t.Context(), "db`name", "9007199254740993", test.policy); err != nil {
				t.Fatal(err)
			}

			response, err := handler.ControlMaterializedView(t.Context(), &MaterializedViewControlInput{Database: "db`name", ID: "9007199254740993", Action: test.action})
			if err != nil || response == nil {
				t.Fatalf("expected successful command: %v, %v", response, err)
			}

			if client.command != "ALTER MATERIALIZED VIEW `db``name`.`view``name` "+test.expected {
				t.Fatalf("unexpected command: %s", client.command)
			}

			if !strings.Contains(client.query, "WHERE TABLE_SCHEMA = ? AND MATERIALIZED_VIEW_ID = ?") ||
				!reflect.DeepEqual(client.args, []any{"db`name", "9007199254740993"}) {
				t.Fatalf("expected parameterized lookup: %s %v", client.query, client.args)
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

func TestMaterializedViewForcedRefresh(t *testing.T) {
	client := &materializedViewControlClient{materializedViewsClient: materializedViewsClient{
		views: []MaterializedView{{Name: "view`name", RefreshType: "ASYNC", Active: "true", RefreshTrigger: "MANUAL", PartitionType: "RANGE"}},
	}}
	handler := &HandlerMaterializedViews{client: client}

	response, err := handler.RefreshMaterializedView(t.Context(), &MaterializedViewRefreshInput{Database: "db`name", ID: "9007199254740993", StartDate: "2026-10-20", EndDate: "2026-10-23"})
	if err != nil || response == nil {
		t.Fatalf("expected successful submission: %v, %v", response, err)
	}

	if client.command != "REFRESH MATERIALIZED VIEW `db``name`.`view``name` PARTITION START ('2026-10-20') END ('2026-10-24') FORCE WITH ASYNC MODE" {
		t.Fatalf("unexpected refresh command: %s", client.command)
	}

	if !reflect.DeepEqual(client.args, []any{"db`name", "9007199254740993"}) {
		t.Fatalf("expected scoped metadata lookup: %v", client.args)
	}
}

func TestMaterializedViewForcedRefreshRejectsInvalidDates(t *testing.T) {
	for _, dates := range [][2]string{
		{"", "2026-10-23"},
		{"2026-10-20", ""},
		{"2026-10-23", "2026-10-20"},
		{"2026-10-20", "9999-12-31"},
		{"2026-02-30", "2026-03-01"},
		{"2026-10-20'); DROP TABLE events; --", "2026-10-23"},
		{"2026-10-20T12:00:00Z", "2026-10-23"},
		{"0000-01-01", "2026-10-23"},
	} {
		client := &materializedViewControlClient{}
		handler := &HandlerMaterializedViews{client: client}

		response, err := handler.RefreshMaterializedView(t.Context(), &MaterializedViewRefreshInput{Database: "analytics", ID: "1", StartDate: dates[0], EndDate: dates[1]})
		if response != nil || err == nil || httpserver.GetErrorStatusCode(err) != 400 || client.query != "" || client.command != "" {
			t.Fatalf("invalid dates must not query or execute: %v, %v, %s", response, err, client.command)
		}
	}
}

func TestMaterializedViewForcedRefreshRejectsUnsupportedViews(t *testing.T) {
	for _, view := range []*MaterializedView{
		nil,
		{RefreshType: "SYNC", Active: "true", PartitionType: "RANGE"},
		{RefreshType: "ASYNC", Active: "false", PartitionType: "RANGE"},
		{RefreshType: "ASYNC", Active: "true", PartitionType: "UNKNOWN"},
		{RefreshType: "ASYNC", Active: "true", PartitionType: "UNPARTITIONED"},
	} {
		client := &materializedViewControlClient{}
		expectedStatus := 404

		if view != nil {
			client.views = []MaterializedView{*view}
			expectedStatus = 409
		}

		handler := &HandlerMaterializedViews{client: client}

		response, err := handler.RefreshMaterializedView(t.Context(), &MaterializedViewRefreshInput{Database: "analytics", ID: "1", StartDate: "2026-10-20", EndDate: "2026-10-23"})
		if response != nil || err == nil || httpserver.GetErrorStatusCode(err) != expectedStatus || client.command != "" {
			t.Fatalf("unsupported view must not refresh: %v, %v, %s", response, err, client.command)
		}
	}
}

func TestMaterializedViewForcedRefreshErrors(t *testing.T) {
	failure := errors.New("permission denied")
	for _, client := range []*materializedViewControlClient{
		{materializedViewsClient: materializedViewsClient{err: failure}},
		{materializedViewsClient: materializedViewsClient{views: []MaterializedView{{Name: "view", RefreshType: "ASYNC", Active: "true", PartitionType: "RANGE"}}}, execErr: failure},
	} {
		handler := &HandlerMaterializedViews{client: client}

		response, err := handler.RefreshMaterializedView(t.Context(), &MaterializedViewRefreshInput{Database: "analytics", ID: "1", StartDate: "2026-10-20", EndDate: "2026-10-23"})
		if response != nil || !errors.Is(err, failure) {
			t.Fatalf("expected wrapped refresh error: %v, %v", response, err)
		}
	}
}

func TestMaterializedViewRefreshRequestBinding(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/api/materialized-views/analytics/1/refresh", strings.NewReader(`{"startDate":"2026-10-20","endDate":"2026-10-23","database":"other","id":"2"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Params = gin.Params{{Key: "database", Value: "analytics"}, {Key: "id", Value: "1"}}

	input, err := httpserver.BindHandleRequest[MaterializedViewRefreshInput](ctx, []string{"uri", "json"}, nil)
	if err != nil || input.Database != "analytics" || input.ID != "1" || input.StartDate != "2026-10-20" || input.EndDate != "2026-10-23" {
		t.Fatalf("expected dates from JSON and identity from route: %v, %v", input, err)
	}
}

func TestDatePartitionValues(t *testing.T) {
	start := time.Date(2026, time.October, 20, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.October, 24, 0, 0, 0, 0, time.UTC)
	handler := &HandlerMaterializedViews{}

	values, err := handler.datePartitionValues(`[["2026-10-19 00:00:00"],["2026-10-20 00:00:00"],["2026-10-22"],["2026-10-23 23:59:59"],["2026-10-24 00:00:00"]]`, start, end)
	if err != nil || !reflect.DeepEqual(values, []string{"'2026-10-20 00:00:00'", "'2026-10-22'", "'2026-10-23 23:59:59'"}) {
		t.Fatalf("expected values inside the half-open date range: %v, %v", values, err)
	}

	for _, invalid := range []string{`null`, `[]`, `[["2026-02-30"]]`, `[["2026-10-20", "other"]]`, `[["2026-10-20'); DROP TABLE events; --"]]`, `not json`} {
		if _, err := handler.datePartitionValues(invalid, start, end); err == nil {
			t.Fatalf("unsafe partition values must fail: %s", invalid)
		}
	}
}

func TestAutomaticRefreshClause(t *testing.T) {
	handler := &HandlerMaterializedViews{}
	for _, policy := range []string{
		"MANUAL", "", "EVERY(INTERVAL 0 MINUTE)", "EVERY(INTERVAL 1 YEAR)",
		"EVERY(INTERVAL 999999999999999999999999 DAY)",
		`START("2026-02-30 00:00:00") EVERY(INTERVAL 5 MINUTE)`,
		"EVERY(INTERVAL 5 MINUTE); DROP TABLE events",
	} {
		if _, err := handler.automaticRefreshClause(policy); err == nil {
			t.Fatalf("unsafe or invalid policy must fail: %s", policy)
		}
	}
}

func TestManualRefreshPolicySurvivesRestart(t *testing.T) {
	directory := t.TempDir()
	client := &materializedViewControlClient{materializedViewsClient: materializedViewsClient{
		views: []MaterializedView{{Name: "view", Database: "analytics", ID: "1", RefreshType: "ASYNC", Active: "true", RefreshTrigger: "MANUAL", RefreshPolicy: "MANUAL"}},
	}}
	handler := &HandlerMaterializedViews{client: client, state: testLocalState(t, filepath.Join(directory, "state.sqlite"))}
	client.views[0].RefreshTrigger = "SCHEDULED"
	client.views[0].RefreshPolicy = "EVERY(INTERVAL 5 MINUTE)"

	if _, err := handler.ControlMaterializedView(t.Context(), &MaterializedViewControlInput{Database: "analytics", ID: "1", Action: "pause"}); err != nil {
		t.Fatal(err)
	}

	client.views[0].RefreshTrigger = "MANUAL"
	client.views[0].RefreshPolicy = "MANUAL"

	if err := handler.state.Close(); err != nil {
		t.Fatal(err)
	}

	restarted := &HandlerMaterializedViews{client: client, state: testLocalState(t, filepath.Join(directory, "state.sqlite"))}
	var response httpserver.Response
	var err error

	if response, err = restarted.GetMaterializedViews(t.Context()); err != nil {
		t.Fatal(err)
	}

	body, err := response.Body()
	if err != nil || !strings.Contains(string(body), `"canResume":true`) {
		t.Fatalf("saved policy must enable Resume: %s, %v", body, err)
	}

	if _, err := restarted.ControlMaterializedView(t.Context(), &MaterializedViewControlInput{Database: "analytics", ID: "1", Action: "resume"}); err != nil {
		t.Fatal(err)
	}

	if client.command != "ALTER MATERIALIZED VIEW `analytics`.`view` REFRESH ASYNC EVERY(INTERVAL 5 MINUTE)" {
		t.Fatalf("resume must restore the saved policy: %s", client.command)
	}

	if handler.refreshPolicyKey("analytics", "1") == handler.refreshPolicyKey("other", "1") || handler.refreshPolicyKey("analytics", "1") == handler.refreshPolicyKey("analytics", "2") {
		t.Fatal("saved policies must be scoped to database and MV ID")
	}
}

func TestPauseDoesNotChangeViewWhenPolicyCannotBeSaved(t *testing.T) {
	state := testLocalState(t, filepath.Join(t.TempDir(), "state.sqlite"))
	if err := state.Close(); err != nil {
		t.Fatal(err)
	}

	client := &materializedViewControlClient{materializedViewsClient: materializedViewsClient{
		views: []MaterializedView{{Name: "view", RefreshType: "ASYNC", Active: "true", RefreshTrigger: "SCHEDULED", RefreshPolicy: "EVERY(INTERVAL 5 MINUTE)"}},
	}}
	handler := &HandlerMaterializedViews{client: client, state: state}

	if _, err := handler.ControlMaterializedView(t.Context(), &MaterializedViewControlInput{Database: "analytics", ID: "1", Action: "pause"}); err == nil || client.command != "" {
		t.Fatalf("failed policy save must prevent pausing: %v, %s", err, client.command)
	}
}

func TestManualRefreshResumeKeepsPolicyOnDatabaseFailure(t *testing.T) {
	failure := errors.New("permission denied")
	client := &materializedViewControlClient{materializedViewsClient: materializedViewsClient{
		views: []MaterializedView{{Name: "view", RefreshType: "ASYNC", Active: "true", RefreshTrigger: "MANUAL", RefreshPolicy: "MANUAL"}},
	}, execErr: failure}
	handler := &HandlerMaterializedViews{client: client, state: testLocalState(t, filepath.Join(t.TempDir(), "state.sqlite"))}

	if err := handler.saveRefreshPolicy(t.Context(), "analytics", "1", "EVERY(INTERVAL 5 MINUTE)"); err != nil {
		t.Fatal(err)
	}

	input := &MaterializedViewControlInput{Database: "analytics", ID: "1", Action: "resume"}
	if _, err := handler.ControlMaterializedView(t.Context(), input); !errors.Is(err, failure) {
		t.Fatalf("expected database error: %v", err)
	}

	policy, err := handler.loadRefreshPolicy(t.Context(), "analytics", "1")
	if err != nil || policy != "EVERY(INTERVAL 5 MINUTE)" {
		t.Fatalf("failed resume must preserve policy: %s, %v", policy, err)
	}

	client.execErr = nil
	if _, err := handler.ControlMaterializedView(t.Context(), input); err != nil {
		t.Fatal(err)
	}

	if _, err := handler.loadRefreshPolicy(t.Context(), "analytics", "1"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("successful resume must clear the saved policy: %v", err)
	}
}

func TestForcedRefreshInclusiveDateBoundaries(t *testing.T) {
	for _, dates := range [][2]string{
		{"2026-10-20", "2026-10-21"},
		{"2026-10-31", "2026-11-01"},
		{"2026-12-31", "2027-01-01"},
		{"2024-02-29", "2024-03-01"},
	} {
		client := &materializedViewControlClient{materializedViewsClient: materializedViewsClient{
			views: []MaterializedView{{Name: "view", RefreshType: "ASYNC", Active: "true", RefreshTrigger: "MANUAL", PartitionType: "RANGE"}},
		}}
		handler := &HandlerMaterializedViews{client: client}

		if _, err := handler.RefreshMaterializedView(t.Context(), &MaterializedViewRefreshInput{Database: "analytics", ID: "1", StartDate: dates[0], EndDate: dates[0]}); err != nil {
			t.Fatal(err)
		}

		if !strings.Contains(client.command, "START ('"+dates[0]+"') END ('"+dates[1]+"')") {
			t.Fatalf("single-day refresh must include the full day: %s", client.command)
		}
	}
}
