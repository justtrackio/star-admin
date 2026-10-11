package internal

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/gosoline-project/httpserver"
	"github.com/gosoline-project/sqlc"
)

type materializedViewsClient struct {
	sqlc.Client
	views []MaterializedView
	err   error
}

func (c *materializedViewsClient) Select(ctx context.Context, dest any, _ string, _ ...any) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("metadata read must have a deadline")
	}
	if c.err != nil {
		return c.err
	}
	views := dest.(*[]MaterializedView)
	*views = append(*views, c.views...)

	return nil
}

func TestMaterializedViewsResponse(t *testing.T) {
	for _, tc := range []struct {
		name  string
		views []MaterializedView
	}{
		{name: "empty cluster"},
		{name: "nullable metadata and large ID", views: []MaterializedView{{ID: "9007199254740993", Database: "analytics", Name: "events"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := &HandlerMaterializedViews{client: &materializedViewsClient{views: tc.views}}
			var response httpserver.Response
			var body []byte
			var err error
			if response, err = handler.GetMaterializedViews(t.Context()); err != nil {
				t.Fatal(err)
			}
			if body, err = response.Body(); err != nil {
				t.Fatal(err)
			}
			var result struct {
				Views []MaterializedView `json:"views"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				t.Fatal(err)
			}
			if result.Views == nil || len(result.Views) != len(tc.views) {
				t.Fatalf("expected a non-null views array, got %s", body)
			}
			if len(tc.views) > 0 && (result.Views[0].ID != tc.views[0].ID || result.Views[0].Rows != nil) {
				t.Fatalf("large ID or unknown row count lost: %s", body)
			}
		})
	}
}

func TestMaterializedViewsQueryError(t *testing.T) {
	queryError := errors.New("metadata unavailable")
	handler := &HandlerMaterializedViews{client: &materializedViewsClient{err: queryError}}
	response, err := handler.GetMaterializedViews(t.Context())
	if response != nil || !errors.Is(err, queryError) {
		t.Fatalf("expected wrapped query error and no response, got %v, %v", response, err)
	}
}
