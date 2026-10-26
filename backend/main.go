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

			api := router.Group("/api")

			api.HandleWith(httpserver.With(internal.NewHandlerHealth, func(r *httpserver.Router, handler *internal.HandlerHealth) {
				r.GET("/health", httpserver.BindN(handler.GetHealth))
			}))
			api.HandleWith(httpserver.With(internal.NewHandlerOverview, func(r *httpserver.Router, handler *internal.HandlerOverview) {
				r.GET("/overview", httpserver.BindN(handler.GetOverview))
				r.GET("/nodes", httpserver.BindSseNR(handler.WatchNodes))
				r.GET("/frontend-config", httpserver.BindSseNR(handler.WatchFrontendConfig))
			}))
			api.HandleWith(httpserver.With(internal.NewHandlerTables, func(r *httpserver.Router, handler *internal.HandlerTables) {
				r.GET("/tables", httpserver.BindSseNR(handler.WatchTables))
				r.GET("/routine-loads", httpserver.BindSseNR(handler.WatchRoutineLoads))
				r.GET("/routine-load-settings", httpserver.BindSseNR(handler.WatchRoutineLoadSettings))
				r.GET("/routine-loads/:database/:id/tasks", httpserver.BindSseR(handler.WatchRoutineLoadTasks))
				r.POST("/routine-loads/:database/:id/:action", httpserver.Bind(handler.ControlRoutineLoad))
			}))
			api.HandleWith(httpserver.With(internal.NewHandlerResourceGroups, func(r *httpserver.Router, handler *internal.HandlerResourceGroups) {
				r.GET("/resource-groups", httpserver.BindSseNR(handler.WatchResourceGroups))
			}))
			api.HandleWith(httpserver.With(internal.NewHandlerDictionaries, func(r *httpserver.Router, handler *internal.HandlerDictionaries) {
				r.GET("/dictionaries", httpserver.BindSseNR(handler.WatchDictionaries))
			}))
			api.HandleWith(httpserver.With(internal.NewHandlerMaterializedViews, func(r *httpserver.Router, handler *internal.HandlerMaterializedViews) {
				r.GET("/materialized-views/:database/:id", httpserver.BindSseR(handler.WatchMaterializedView))
				r.GET("/materialized-views/:database/:id/tasks", httpserver.BindSseR(handler.WatchMaterializedViewTasks))
				r.GET("/materialized-views", httpserver.BindSseNR(handler.WatchMaterializedViews))
			}))

			return nil
		})),
	).Run()
}
