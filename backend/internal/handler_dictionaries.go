package internal

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gosoline-project/httpserver"
	"github.com/gosoline-project/sqlc"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// Dictionary contains read-only metadata reported by SHOW DICTIONARY.
type Dictionary struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	Catalog             string `json:"catalog"`
	Database            string `json:"database"`
	Source              string `json:"source"`
	Keys                string `json:"keys"`
	Values              string `json:"values"`
	Status              string `json:"status"`
	LastRefreshStarted  string `json:"lastRefreshStarted"`
	LastRefreshFinished string `json:"lastRefreshFinished"`
	NextRefresh         string `json:"nextRefresh"`
	Error               string `json:"error"`
	MemoryUsage         string `json:"memoryUsage"`
}

type DictionariesResponse struct {
	Dictionaries []Dictionary `json:"dictionaries"`
}

type HandlerDictionaries struct {
	client sqlc.Client
}

func NewHandlerDictionaries(ctx context.Context, config cfg.Config, logger log.Logger) (*HandlerDictionaries, error) {
	var client sqlc.Client
	var err error

	if client, err = ProvideStarRocksClient(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not create dictionaries handler: %w", err)
	}

	return &HandlerDictionaries{client: client}, nil
}

func (h *HandlerDictionaries) GetDictionaries(ctx context.Context) (response httpserver.Response, err error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var rows *sqlc.Rows
	var columns []string

	if rows, err = h.client.Query(ctx, "SHOW DICTIONARY"); err != nil {
		return nil, fmt.Errorf("could not list dictionaries: %w", err)
	}

	defer func() {
		var closeErr error
		if closeErr = rows.Close(); closeErr != nil && err == nil {
			response = nil
			err = fmt.Errorf("could not close dictionary rows: %w", closeErr)
		}
	}()

	if columns, err = rows.Columns(); err != nil {
		return nil, fmt.Errorf("could not read dictionary columns: %w", err)
	}

	dictionaries := make([]Dictionary, 0)

	for rows.Next() {
		values := make([]sql.NullString, len(columns))
		destinations := make([]any, len(columns))

		for i := range values {
			destinations[i] = &values[i]
		}

		if err = rows.Scan(destinations...); err != nil {
			return nil, fmt.Errorf("could not read dictionary: %w", err)
		}

		metadata := make(map[string]string, len(columns))

		for i, column := range columns {
			metadata[column] = values[i].String
		}

		dictionaries = append(dictionaries, h.dictionaryFromMetadata(metadata))
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("could not iterate dictionaries: %w", err)
	}

	return httpserver.NewJsonResponse(DictionariesResponse{Dictionaries: dictionaries}), nil
}

// Support column aliases used by different StarRocks releases.
func (h *HandlerDictionaries) dictionaryFromMetadata(m map[string]string) Dictionary {
	source := m["dictionaryObject"]
	if source == "" {
		source = m["dictionarySource"]
	}

	status := m["status"]
	if status == "" {
		status = m["state"]
	}

	return Dictionary{
		ID: m["DictionaryId"], Name: m["DictionaryName"], Catalog: m["CatalogName"],
		Database: m["DbName"], Source: source, Keys: m["dictionaryKeys"], Values: m["dictionaryValues"],
		Status: status, LastRefreshStarted: m["lastSuccessRefreshTime"],
		LastRefreshFinished: m["lastSuccessFinishedTime"], NextRefresh: m["nextSchedulableTime"],
		Error: m["ErrorMessage"], MemoryUsage: m["approximated dictionaryMemoryUsage (Bytes)"],
	}
}

func (h *HandlerDictionaries) WatchDictionaries(_ context.Context, request *http.Request, writer *httpserver.SseWriter) error {
	return watchSnapshots(request, writer, h.GetDictionaries)
}
