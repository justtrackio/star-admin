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

type routineControlClient struct {
	sqlc.Client
	state       string
	progress    string
	reason      string
	commands    []string
	failCommand int
	name        string
	empty       bool
	selectErr   error
	execErr     error
	query       string
	args        []any
	command     string
}

func (c *routineControlClient) Select(ctx context.Context, dest any, query string, args ...any) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("missing metadata deadline")
	}

	c.query, c.args = query, args
	if c.selectErr != nil {
		return c.selectErr
	}

	if !c.empty {
		*dest.(*[]RoutineLoad) = []RoutineLoad{{Name: c.name, State: c.state, Progress: c.progress, Reason: c.reason}}
	}

	return nil
}

func (c *routineControlClient) Exec(ctx context.Context, query string, _ ...any) (sqlc.Result, error) {
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("missing command deadline")
	}

	c.command = query
	c.commands = append(c.commands, query)

	if c.failCommand > 0 && len(c.commands) != c.failCommand {
		return nil, nil
	}

	return nil, c.execErr
}

func TestRoutineLoadControl(t *testing.T) {
	for _, test := range []struct{ action, state string }{
		{"pause", "RUNNING"}, {"pause", "NEED_SCHEDULE"}, {"pause", "UNSTABLE"}, {"resume", "PAUSED"},
	} {
		t.Run(test.action+"_"+test.state, func(t *testing.T) {
			client := &routineControlClient{state: test.state, name: "job`name"}
			handler := &HandlerTables{client: client}
			var response httpserver.Response
			var err error

			if response, err = handler.ControlRoutineLoad(t.Context(), &RoutineLoadControlInput{Database: "db`name", ID: "9007199254740993", Action: test.action}); err != nil {
				t.Fatal(err)
			}

			if response.StatusCode() != 200 {
				t.Fatalf("unexpected status: %d", response.StatusCode())
			}

			if client.command != strings.ToUpper(test.action)+" ROUTINE LOAD FOR `db``name`.`job``name`" {
				t.Fatalf("unsafe command: %s", client.command)
			}

			if !strings.Contains(client.query, "DB_NAME = ? AND ID = ? AND DATA_SOURCE_TYPE = 'KAFKA'") || !reflect.DeepEqual(client.args, []any{"db`name", "9007199254740993"}) {
				t.Fatalf("job lookup must bind database and ID: %s, %v", client.query, client.args)
			}
		})
	}
}

func TestRoutineLoadControlRejectsInvalidTransitions(t *testing.T) {
	for _, test := range []struct {
		action, state string
		empty         bool
		status        int
	}{
		{"stop", "RUNNING", false, 400},
		{"pause", "PAUSED", false, 409},
		{"resume", "RUNNING", false, 409},
		{"pause", "CANCELLED", false, 409},
		{"resume", "STOPPED", false, 409},
		{"pause", "UNKNOWN", false, 409},
		{"resume", "", true, 404},
	} {
		t.Run(test.action+"_"+test.state, func(t *testing.T) {
			client := &routineControlClient{state: test.state, empty: test.empty}
			handler := &HandlerTables{client: client}

			response, err := handler.ControlRoutineLoad(t.Context(), &RoutineLoadControlInput{Database: "analytics", ID: "1", Action: test.action})
			if response != nil || err == nil || httpserver.GetErrorStatusCode(err) != test.status || client.command != "" {
				t.Fatalf("invalid transition must not execute: %v, %v, %s", response, err, client.command)
			}
		})
	}
}

func TestRoutineLoadControlErrors(t *testing.T) {
	failure := errors.New("permission denied")
	for _, client := range []*routineControlClient{
		{selectErr: failure}, {state: "PAUSED", name: "job", execErr: failure},
	} {
		handler := &HandlerTables{client: client}

		response, err := handler.ControlRoutineLoad(t.Context(), &RoutineLoadControlInput{Database: "analytics", ID: "1", Action: "resume"})
		if response != nil || !errors.Is(err, failure) {
			t.Fatalf("expected wrapped database error: %v, %v", response, err)
		}
	}
}

func TestRoutineLoadOffsetRecovery(t *testing.T) {
	for _, test := range []struct{ action, offset string }{
		{"resume-earliest", "OFFSET_BEGINNING"}, {"resume-latest", "OFFSET_END"},
	} {
		t.Run(test.action, func(t *testing.T) {
			client := &routineControlClient{state: "PAUSED", name: "job`name", reason: "Broker: Offset out of range (broker 0)", progress: `{"10":"9007199254740993","0":"5","2":"9"}`}
			handler := &HandlerTables{client: client}
			var err error

			if _, err = handler.ControlRoutineLoad(t.Context(), &RoutineLoadControlInput{Database: "db`name", ID: "1", Action: test.action}); err != nil {
				t.Fatal(err)
			}

			expected := []string{
				`ALTER ROUTINE LOAD FOR ` + "`db``name`.`job``name`" + ` FROM KAFKA ("kafka_partitions" = "0,2,10", "kafka_offsets" = "` + test.offset + `,` + test.offset + `,` + test.offset + `")`,
				"RESUME ROUTINE LOAD FOR `db``name`.`job``name`",
			}
			if !reflect.DeepEqual(client.commands, expected) {
				t.Fatalf("expected ordered reset then resume: %v", client.commands)
			}
		})
	}
}

func TestRoutineLoadOffsetRecoveryRejectsUnsafeRequests(t *testing.T) {
	for _, test := range []struct{ state, reason, progress string }{
		{"RUNNING", "offset out of range", `{"0":"1"}`},
		{"PAUSED", "permission denied", `{"0":"1"}`},
		{"PAUSED", "offset out of range", ""},
		{"PAUSED", "offset out of range", "{}"},
		{"PAUSED", "offset out of range", "null"},
		{"PAUSED", "offset out of range", "[]"},
		{"PAUSED", "offset out of range", `{"0\"; DROP TABLE events; --":"1"}`},
		{"PAUSED", "offset out of range", `{"-1":"1"}`},
	} {
		t.Run(test.state+"_"+test.progress, func(t *testing.T) {
			client := &routineControlClient{state: test.state, reason: test.reason, progress: test.progress}
			handler := &HandlerTables{client: client}

			response, err := handler.ControlRoutineLoad(t.Context(), &RoutineLoadControlInput{Database: "analytics", ID: "1", Action: "resume-earliest"})
			if response != nil || err == nil || httpserver.GetErrorStatusCode(err) != 409 || len(client.commands) != 0 {
				t.Fatalf("unsafe recovery must not execute: %v, %v, %v", response, err, client.commands)
			}
		})
	}
}

func TestRoutineLoadOffsetRecoveryFailures(t *testing.T) {
	failure := errors.New("permission denied")
	for _, failCommand := range []int{1, 2} {
		client := &routineControlClient{state: "PAUSED", name: "job", reason: "requested offset not available", progress: `{"0":"1"}`, execErr: failure, failCommand: failCommand}
		handler := &HandlerTables{client: client}

		response, err := handler.ControlRoutineLoad(t.Context(), &RoutineLoadControlInput{Database: "analytics", ID: "1", Action: "resume-latest"})
		if response != nil || !errors.Is(err, failure) || len(client.commands) != failCommand {
			t.Fatalf("failed reset must prevent resume: %v, %v, %v", response, err, client.commands)
		}

		if failCommand == 2 && !strings.Contains(err.Error(), "offsets were reset") {
			t.Fatalf("partial recovery must be explicit: %v", err)
		}
	}
}
