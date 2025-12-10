package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	_ "subscription-aggregator/docs"

	"subscription-aggregator/internal/db"
	"subscription-aggregator/internal/handler"
	"subscription-aggregator/internal/repository"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func main() {
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	if err := db.InitDB(); err != nil {
		slog.Error("❌ Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.GetConn().Close(context.Background()); err != nil {
			slog.Warn("Failed to close DB connection", "error", err)
		}
	}()

	if err := db.RunMigrations(); err != nil {
		slog.Error("❌ Failed to run migrations", "error", err)
		os.Exit(1)
	}

	repo := repository.NewPostgresSubscriptionRepo(db.GetConn())
	h := handler.NewSubscriptionHandler(repo)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /subscriptions", h.CreateSubscription)
	mux.HandleFunc("GET /subscriptions/{id}", h.GetSubscription)
	mux.HandleFunc("GET /subscriptions", h.ListSubscriptions)
	mux.HandleFunc("PUT /subscriptions/{id}", h.UpdateSubscription)
	mux.HandleFunc("DELETE /subscriptions/{id}", h.DeleteSubscription)
	mux.HandleFunc("GET /subscriptions/total-cost", h.GetTotalCost)

	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("🚀 Starting HTTP server", "port", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		slog.Error("❌ Server crashed", "error", err)
		os.Exit(1)
	}
}
