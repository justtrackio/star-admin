package internal

import (
	"net/url"

	"github.com/gosoline-project/sqlc"
	"github.com/justtrackio/gosoline/pkg/log"
	_ "modernc.org/sqlite"
)

func init() {
	sqlc.AddDriverFactory("sqlite", func(_ log.Logger) (sqlc.Driver, error) { return &sqliteDriver{}, nil })
}

type sqliteDriver struct{}

func (s *sqliteDriver) GetDSN(settings *sqlc.Settings) string {
	uri := url.URL{Scheme: "file", Path: settings.Uri.Database, RawQuery: "_pragma=busy_timeout(5000)"}

	return uri.String()
}

func (s *sqliteDriver) GetPlaceholder() string { return "?" }

func (s *sqliteDriver) GetQuote() string { return "`" }
