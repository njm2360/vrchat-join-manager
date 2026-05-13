package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	oapimw "github.com/oapi-codegen/echo-middleware"

	"github.com/njm2360/vrchat-join-manager/server/internal/db"
	"github.com/njm2360/vrchat-join-manager/server/internal/gen"
	"github.com/njm2360/vrchat-join-manager/server/internal/handler"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	dbPath := envOr("DB_PATH", "data/vrchat.db")
	addr := envOr("LISTEN_ADDR", ":8080")

	conn, err := db.Open(dbPath)
	if err != nil {
		slog.Error("db open failed", "err", err)
		os.Exit(1)
	}
	defer conn.Close()

	srv := handler.New(conn)
	strict := gen.NewStrictHandler(srv, nil)

	spec, err := gen.GetSwagger()
	if err != nil {
		slog.Error("load openapi spec failed", "err", err)
		os.Exit(1)
	}
	spec.Servers = nil

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

	apiGroup := e.Group("")
	apiGroup.Use(oapimw.OapiRequestValidator(spec))
	gen.RegisterHandlersWithBaseURL(apiGroup, strict, "")

	e.GET("/openapi.json", func(c echo.Context) error {
		return c.JSON(http.StatusOK, spec)
	})
	e.File("/docs", "static/swagger.html")

	e.Static("/", "static")
	e.GET("/", func(c echo.Context) error {
		return c.File("static/index.html")
	})

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("listening", "addr", addr, "db", dbPath)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
