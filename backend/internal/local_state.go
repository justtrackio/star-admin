package internal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gosoline-project/sqlc"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type localStateContextKey struct{}

func ProvideLocalState(ctx context.Context, config cfg.Config, logger log.Logger) (sqlc.Client, error) {
	return appctx.Provide(ctx, localStateContextKey{}, func() (sqlc.Client, error) {
		var err error
		var settings *sqlc.Settings

		if settings, err = sqlc.ReadSettings(config, "local_state"); err != nil {
			return nil, fmt.Errorf("could not read local state settings: %w", err)
		}

		return newLocalState(ctx, config, logger, settings)
	})
}

func newLocalState(ctx context.Context, config cfg.Config, logger log.Logger, settings *sqlc.Settings) (sqlc.Client, error) {
	var err error
	var file *os.File
	var path string

	if settings.Uri.Database == "" {
		return nil, errors.New("sqlc.local_state.uri.database is empty")
	}

	if path, err = filepath.Abs(settings.Uri.Database); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("could not create local state directory: %w", err)
	}

	if file, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600); err != nil {
		return nil, fmt.Errorf("could not create local state database: %w", err)
	}

	if err := file.Close(); err != nil {
		return nil, err
	}

	settings.Uri.Database = path

	return sqlc.NewClientWithSettings(ctx, config, logger, "local_state", settings)
}

func NewModuleLocalState(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	var err error
	var client sqlc.Client

	if client, err = ProvideLocalState(ctx, config, logger); err != nil {
		return nil, err
	}

	return &ModuleLocalState{client: client}, nil
}

type ModuleLocalState struct {
	kernel.ServiceStage
	kernel.BackgroundModule
	client sqlc.Client
}

func (m *ModuleLocalState) Run(ctx context.Context) error {
	<-ctx.Done()
	if err := m.client.Close(); err != nil {
		return fmt.Errorf("could not close local state database: %w", err)
	}

	return nil
}
