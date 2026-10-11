package internal

import (
	"context"
	"fmt"

	"github.com/gosoline-project/sqlc"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

// ProvideStarRocksClient shares sqlc's connection pool with handlers and services.
func ProvideStarRocksClient(ctx context.Context, config cfg.Config, logger log.Logger) (sqlc.Client, error) {
	var err error
	var client sqlc.Client

	if client, err = sqlc.ProvideClient(ctx, config, logger, "default"); err != nil {
		return nil, fmt.Errorf("could not connect to StarRocks: %w", err)
	}

	return client, nil
}

// ModuleStarRocks owns the connection pool's application lifecycle.
type ModuleStarRocks struct {
	kernel.ServiceStage
	kernel.BackgroundModule

	client sqlc.Client
}

// NewModuleStarRocks connects and pings the configured cluster during application initialization.
func NewModuleStarRocks(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	var client sqlc.Client
	var err error
	if client, err = ProvideStarRocksClient(ctx, config, logger); err != nil {
		return nil, fmt.Errorf("could not initialize StarRocks connection: %w", err)
	}

	logger.Info(ctx, "StarRocks connection pool ready")

	return &ModuleStarRocks{client: client}, nil
}

// Run keeps the pool open until application modules have stopped, then closes it.
func (m *ModuleStarRocks) Run(ctx context.Context) error {
	<-ctx.Done()

	if err := m.client.Close(); err != nil {
		return fmt.Errorf("could not close StarRocks connection pool: %w", err)
	}

	return nil
}
