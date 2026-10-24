package internal

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gosoline-project/httpserver"
)

type snapshotWriter struct {
	events []httpserver.SseEvent
	cancel context.CancelFunc
}

func (w *snapshotWriter) SendEvent(event httpserver.SseEvent) error {
	w.events = append(w.events, event)
	if len(w.events) == 2 {
		w.cancel()
	}

	return nil
}

func TestSnapshotStreamRetriesAndStopsOnDisconnect(t *testing.T) {
	for _, payload := range []struct {
		name     string
		data     any
		expected string
	}{
		{name: "tasks", data: MaterializedViewTasksResponse{Tasks: []MaterializedViewTask{}}, expected: `{"tasks":[]}`},
		{name: "tables", data: TablesResponse{Tables: []Table{}}, expected: `{"tables":[]}`},
		{name: "routine loads", data: RoutineLoadsResponse{Jobs: []RoutineLoad{}}, expected: `{"jobs":[]}`},
		{name: "dictionaries", data: DictionariesResponse{Dictionaries: []Dictionary{}}, expected: `{"dictionaries":[]}`},
	} {
		t.Run(payload.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			writer := &snapshotWriter{cancel: cancel}
			ticks := make(chan time.Time, 1)
			ticks <- time.Now()
			calls := 0
			snapshot := func(ctx context.Context) (httpserver.Response, error) {
				if ctx.Err() != nil {
					t.Fatal("snapshot called after disconnect")
				}

				calls++
				if calls == 1 {
					return nil, errors.New("temporary metadata failure")
				}

				return httpserver.NewJsonResponse(payload.data), nil
			}

			if err := streamSnapshots(ctx, writer, ticks, snapshot); err != nil {
				t.Fatal(err)
			}

			if calls != 2 || len(writer.events) != 2 || writer.events[0].Event != "refresh-error" || writer.events[1].Data != payload.expected || writer.events[1].Retry != 5000 {
				t.Fatalf("expected an immediate error followed by a recovered snapshot, got calls=%d events=%+v", calls, writer.events)
			}
		})
	}
}
