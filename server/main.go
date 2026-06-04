package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	oapimw "github.com/oapi-codegen/echo-middleware"

	"github.com/njm2360/vrchat-join-manager/server/internal/config"
	"github.com/njm2360/vrchat-join-manager/server/internal/db"
	"github.com/njm2360/vrchat-join-manager/server/internal/gen"
	"github.com/njm2360/vrchat-join-manager/server/internal/handler"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}
	basePath := cfg.BasePath

	conn, err := db.Open(cfg.DBPath)
	if err != nil {
		slog.Error("db open failed", "err", err)
		os.Exit(1)
	}
	defer conn.Close()

	srv := handler.New(conn)
	strict := gen.NewStrictHandler(srv, nil)

	spec, err := gen.GetSpec()
	if err != nil {
		slog.Error("load openapi spec failed", "err", err)
		os.Exit(1)
	}
	if basePath != "" {
		spec.Servers = openapi3.Servers{{URL: basePath}}
	} else {
		spec.Servers = nil
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogMethod:   true,
		LogURI:      true,
		LogStatus:   true,
		LogLatency:  true,
		LogError:    true,
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []any{
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"latency", v.Latency,
				"remote_ip", c.RealIP(),
			}
			if v.Error != nil {
				slog.Error("request", append(attrs, "err", v.Error)...)
			} else {
				slog.Info("request", attrs...)
			}
			return nil
		},
	}))

	g := e.Group(basePath)

	validator := oapimw.OapiRequestValidatorWithOptions(spec, &oapimw.Options{SilenceServersWarning: true})
	opMiddlewares := map[string][]echo.MiddlewareFunc{}
	for _, item := range spec.Paths.Map() {
		for _, op := range item.Operations() {
			if op.OperationID != "" {
				opMiddlewares[op.OperationID] = []echo.MiddlewareFunc{validator}
			}
		}
	}
	gen.RegisterHandlersWithOptions(g, strict, gen.RegisterHandlersOptions{
		OperationMiddlewares: opMiddlewares,
	})

	g.GET("/openapi.json", func(c echo.Context) error {
		return c.JSON(http.StatusOK, spec)
	})
	g.File("/docs", filepath.Join(cfg.FrontendDir, "swagger.html"))

	static := handler.NewStaticHandler(cfg.FrontendDir, basePath)
	g.GET("/*", static.Serve)

	if basePath != "" {
		e.GET(basePath, func(c echo.Context) error {
			return c.Redirect(http.StatusMovedPermanently, basePath+"/")
		})
	}

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", cfg.ListenAddr, "db", cfg.DBPath)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	stop()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}
