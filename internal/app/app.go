package app

import (
	"context"
	"errors"
	"github.com/fandasy/06.08.2025/internal/config"
	zips_download "github.com/fandasy/06.08.2025/internal/http/handlers/zips-download"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"log/slog"
	"net/http"
	"net/url"

	add_objects "github.com/fandasy/06.08.2025/internal/http/handlers/add-objects"
	get_status "github.com/fandasy/06.08.2025/internal/http/handlers/get-status"
	new_task "github.com/fandasy/06.08.2025/internal/http/handlers/new-task"

	"github.com/fandasy/06.08.2025/internal/http/middlewares/cors"
	"github.com/fandasy/06.08.2025/internal/http/middlewares/logger"

	"github.com/fandasy/06.08.2025/internal/models"

	local_zip_storage "github.com/fandasy/06.08.2025/internal/object-storage/local-zip-storage"
	"github.com/fandasy/06.08.2025/internal/services/archiver"
	"github.com/fandasy/06.08.2025/internal/services/archiver/utils"

	_ "github.com/fandasy/06.08.2025/docs"
	"github.com/gin-gonic/gin"
)

type App struct {
	server   *http.Server
	archiver archiver.Archiver
}

// @title           ZIP Archiver API
// @version         1.0.0
// @description     API for archiving files
func New(env string, cfg *config.Config, log *slog.Logger) (*App, error) {
	log.Debug("Config", slog.String("env", env), slog.Any("cfg", cfg))

	archiveObjectGetter := utils.NewArchiveObjectGetter(http.DefaultClient, cfg.Archiver.ArchiveObjectGetter.ValidContentType)

	zipsDownloadMethodPath := url.URL{
		Scheme: "http",
		Host:   cfg.HttpServer.Addr,
		Path:   "zips",
	}

	localZipStorage, err := local_zip_storage.New(zipsDownloadMethodPath.String(), cfg.LocalZipStorage.Dir)
	if err != nil {
		return nil, err
	}

	Archiver := archiver.New(archiver.Config{
		MaxTasks:   cfg.Archiver.MaxTasks,
		MaxObjects: cfg.Archiver.MaxObjects,
	}, archiveObjectGetter, localZipStorage, log)

	if env == models.EnvProd {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(cors.Middleware())
	router.Use(logger.Middleware(log))
	router.Use(gin.Recovery())

	router.GET("/task/new", new_task.New(Archiver, log))
	router.POST("/task/:id/add", add_objects.New(Archiver, cfg.Archiver.ValidExtension, log))
	router.GET("/task/:id/status", get_status.New(Archiver, log))

	router.GET("/zips/:filename", zips_download.New(cfg.LocalZipStorage.Dir, log))

	router.GET("/swagger/:any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	srv := &http.Server{
		Addr:        cfg.HttpServer.Addr,
		Handler:     router,
		IdleTimeout: cfg.HttpServer.IdleTimeout,
	}

	return &App{
		server:   srv,
		archiver: Archiver,
	}, nil
}

func MustNew(env string, cfg *config.Config, log *slog.Logger) *App {
	app, err := New(env, cfg, log)
	if err != nil {
		panic(err)
	}

	return app
}

func (app *App) Run(log *slog.Logger) error {
	log.Info("Server address", slog.String("addr", app.server.Addr))

	if err := app.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (app *App) MustRun(log *slog.Logger) {
	if err := app.Run(log); err != nil {
		panic(err)
	}
}

func (app *App) Shutdown(ctx context.Context, log *slog.Logger) error {
	if err := app.archiver.Stop(ctx); err != nil {
		return err
	}

	log.Info("Archiver service is stopped")

	if err := app.server.Shutdown(ctx); err != nil {
		return err
	}

	log.Info("Server is shutdown")

	return nil
}
