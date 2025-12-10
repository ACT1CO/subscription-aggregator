package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"subscription-aggregator/internal/handler"
	"subscription-aggregator/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndToEnd(t *testing.T) {
	db, err := sql.Open("pgx", "host=localhost port=5433 user=testuser password=testpass dbname=testdb sslmode=disable")
	require.NoError(t, err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, db.PingContext(ctx))

	_, err = db.ExecContext(ctx, `
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		CREATE TABLE subscriptions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service_name TEXT NOT NULL,
			price INTEGER NOT NULL CHECK (price > 0),
			user_id UUID NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT
		);
	`)
	require.NoError(t, err)

	pgxConn, err := pgx.Connect(ctx, "host=localhost port=5433 user=testuser password=testpass dbname=testdb sslmode=disable")
	require.NoError(t, err)
	defer pgxConn.Close(ctx)

	repo := repository.NewPostgresSubscriptionRepo(pgxConn)
	h := handler.NewSubscriptionHandler(repo)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /subscriptions", h.CreateSubscription)
	mux.HandleFunc("GET /subscriptions/{id}", h.GetSubscription)
	mux.HandleFunc("GET /subscriptions", h.ListSubscriptions)
	mux.HandleFunc("PUT /subscriptions/{id}", h.UpdateSubscription)
	mux.HandleFunc("DELETE /subscriptions/{id}", h.DeleteSubscription)
	mux.HandleFunc("GET /subscriptions/total-cost", h.GetTotalCost)

	server := httptest.NewServer(mux)
	defer server.Close()

	userID := uuid.New().String()
	t.Run("Create subscription", func(t *testing.T) {
		body := map[string]interface{}{
			"service_name": "Yandex Plus", "price": 400,
			"user_id": userID, "start_date": "07-2025"}
		resp, err := http.Post(server.URL+"/subscriptions", "application/json", jsonBody(body))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var created map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&created))
		assert.NotEmpty(t, created["id"])
	})

	t.Log("✅ Тест пройден")
}

func jsonBody(v interface{}) *bytes.Reader {
	data, _ := json.Marshal(v)
	return bytes.NewReader(data)
}
