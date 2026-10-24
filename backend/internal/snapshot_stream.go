package internal

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gosoline-project/httpserver"
)

type snapshotEventWriter interface {
	SendEvent(httpserver.SseEvent) error
}

func watchSnapshots(request *http.Request, writer snapshotEventWriter, snapshot func(context.Context) (httpserver.Response, error)) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Use the HTTP request context so a disconnect cancels in-flight SQL.
	return streamSnapshots(request.Context(), writer, ticker.C, snapshot)
}

func streamSnapshots(ctx context.Context, writer snapshotEventWriter, ticks <-chan time.Time, snapshot func(context.Context) (httpserver.Response, error)) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		if err := sendSnapshot(ctx, writer, snapshot); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticks:
		}
	}
}

func sendSnapshot(ctx context.Context, writer snapshotEventWriter, snapshot func(context.Context) (httpserver.Response, error)) error {
	response, err := snapshot(ctx)
	event := httpserver.SseEvent{Retry: 5000}

	if err != nil {
		if ctx.Err() != nil {
			return nil
		}

		// Keep the connection alive and try again on the next tick.
		event.Event = "refresh-error"
		event.Data = err.Error()
	} else {
		var body []byte
		if body, err = response.Body(); err != nil {
			return fmt.Errorf("could not encode metadata snapshot: %w", err)
		}

		event.Data = string(body)
	}

	if err := writer.SendEvent(event); err != nil {
		return fmt.Errorf("could not send metadata snapshot: %w", err)
	}

	return nil
}
