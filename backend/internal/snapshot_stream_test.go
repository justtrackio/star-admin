package internal

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gosoline-project/httpserver"
)

func TestSnapshotStreamCancelsMetadataRead(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	request := httptest.NewRequestWithContext(ctx, "GET", "/api/tables", nil)
	writer := &snapshotWriter{cancel: cancel}
	called := false
	snapshot := func(queryContext context.Context) (httpserver.Response, error) {
		called = true
		cancel()

		if queryContext.Err() == nil {
			t.Fatal("metadata read must use the canceled request context")
		}

		return nil, queryContext.Err()
	}

	if err := watchSnapshots(request, writer, snapshot); err != nil {
		t.Fatal(err)
	}

	if !called || len(writer.events) != 0 {
		t.Fatalf("disconnect must cancel the metadata read without sending an error: called=%v events=%+v", called, writer.events)
	}
}
