package main

import (
	"context"
	"embed"

	"github.com/gin-contrib/cors"
	"github.com/gosoline-project/httpserver"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/star-admin/internal"
)

//go:embed config.dist.yml
var configDist []byte

//go:embed public
var publicFs embed.FS

func main() {
	application.New(
		application.WithConfigBytes(configDist, "yml"),
		application.WithConfigEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
		application.WithConfigFileFlag,
		application.WithConfigSanitizers(cfg.TimeSanitizer),
		application.WithLoggerHandlersFromConfig,
		application.WithUTCClock(true),
		application.WithModuleFactory("starrocks", internal.NewModuleStarRocks),
		application.WithModuleFactory("http", httpserver.NewServer("default", func(ctx context.Context, config cfg.Config, logger log.Logger, router *httpserver.Router) error {
			router.Use(cors.Default())
			router.UseFactory(httpserver.CreateEmbeddedStaticServe(publicFs, "public", "/api"))
			router.Group("/api").HandleWith(httpserver.With(internal.NewHandlerHealth, func(r *httpserver.Router, handler *internal.HandlerHealth) {
				r.GET("/health", httpserver.BindN(handler.GetHealth))
			}))
			router.Group("/api").HandleWith(httpserver.With(internal.NewHandlerMaterializedViews, func(r *httpserver.Router, handler *internal.HandlerMaterializedViews) {
				r.GET("/materialized-views", httpserver.BindN(handler.GetMaterializedViews))
				r.GET("/materialized-views/watch", httpserver.BindSseNR(handler.WatchMaterializedViews))
			}))

			return nil
		})),
	).Run()
}
