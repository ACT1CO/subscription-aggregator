package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"

	"subscription-aggregator/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PostgresSubscriptionRepo struct {
	conn *pgx.Conn
}

func NewPostgresSubscriptionRepo(conn *pgx.Conn) *PostgresSubscriptionRepo {
	return &PostgresSubscriptionRepo{conn: conn}
}

func (r *PostgresSubscriptionRepo) Create(ctx context.Context, sub *model.Subscription) error {
	if _, err := uuid.Parse(sub.UserID); err != nil {
		return fmt.Errorf("invalid user_id UUID: %w", err)
	}
	if !isValidMonthYear(sub.StartDate) {
		return fmt.Errorf("start_date must be in MM-YYYY format")
	}

	query := `
		INSERT INTO subscriptions (service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`

	var id uuid.UUID
	err := r.conn.QueryRow(ctx, query,
		sub.ServiceName,
		sub.Price,
		sub.UserID,
		sub.StartDate,
		sub.EndDate,
	).Scan(&id)
	if err != nil {
		slog.Error("Failed to create subscription", "error", err)
		return fmt.Errorf("database insert failed: %w", err)
	}

	sub.ID = id.String()
	slog.Debug("Subscription created", "id", sub.ID)
	return nil
}

func (r *PostgresSubscriptionRepo) GetByID(ctx context.Context, id string) (*model.Subscription, error) {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid subscription ID format")
	}

	query := `
		SELECT id, service_name, price, user_id, start_date, end_date
		FROM subscriptions
		WHERE id = $1`

	var sub model.Subscription
	var endDate sql.NullString

	err = r.conn.QueryRow(ctx, query, parsedID).Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.Price,
		&sub.UserID,
		&sub.StartDate,
		&endDate,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("subscription not found")
		}
		slog.Error("Failed to get subscription by ID", "id", id, "error", err)
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	if endDate.Valid {
		sub.EndDate = &endDate.String
	}

	return &sub, nil
}

func (r *PostgresSubscriptionRepo) ListByUserID(ctx context.Context, userID string) ([]model.Subscription, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return nil, fmt.Errorf("invalid user_id UUID: %w", err)
	}

	query := `
		SELECT id, service_name, price, user_id, start_date, end_date
		FROM subscriptions
		WHERE user_id = $1
		ORDER BY start_date DESC`

	rows, err := r.conn.Query(ctx, query, userID)
	if err != nil {
		slog.Error("Failed to list subscriptions", "user_id", userID, "error", err)
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	var subs []model.Subscription
	for rows.Next() {
		var sub model.Subscription
		var endDate sql.NullString

		err := rows.Scan(
			&sub.ID,
			&sub.ServiceName,
			&sub.Price,
			&sub.UserID,
			&sub.StartDate,
			&endDate,
		)
		if err != nil {
			slog.Error("Failed to scan subscription row", "error", err)
			continue
		}

		if endDate.Valid {
			sub.EndDate = &endDate.String
		}

		subs = append(subs, sub)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return subs, nil
}

func (r *PostgresSubscriptionRepo) Update(ctx context.Context, id string, sub *model.Subscription) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid subscription ID: %w", err)
	}
	if _, err := uuid.Parse(sub.UserID); err != nil {
		return fmt.Errorf("invalid user_id UUID: %w", err)
	}
	if !isValidMonthYear(sub.StartDate) {
		return fmt.Errorf("start_date must be in MM-YYYY format")
	}

	query := `
		UPDATE subscriptions
		SET service_name = $1, price = $2, user_id = $3, start_date = $4, end_date = $5
		WHERE id = $6`

	commandTag, err := r.conn.Exec(ctx, query,
		sub.ServiceName,
		sub.Price,
		sub.UserID,
		sub.StartDate,
		sub.EndDate,
		parsedID,
	)
	if err != nil {
		slog.Error("Failed to update subscription", "id", id, "error", err)
		return fmt.Errorf("database update failed: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("subscription not found")
	}

	slog.Debug("Subscription updated", "id", id)
	return nil
}

func (r *PostgresSubscriptionRepo) Delete(ctx context.Context, id string) error {
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid subscription ID: %w", err)
	}

	query := `DELETE FROM subscriptions WHERE id = $1`
	commandTag, err := r.conn.Exec(ctx, query, parsedID)
	if err != nil {
		slog.Error("Failed to delete subscription", "id", id, "error", err)
		return fmt.Errorf("database delete failed: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("subscription not found")
	}

	slog.Debug("Subscription deleted", "id", id)
	return nil
}

func (r *PostgresSubscriptionRepo) TotalCost(
	ctx context.Context,
	userID, serviceName, from, to string,
) (int, error) {
	if _, err := uuid.Parse(userID); err != nil {
		return 0, fmt.Errorf("invalid user_id UUID: %w", err)
	}

	if !isValidMonthYear(from) || !isValidMonthYear(to) {
		return 0, fmt.Errorf("dates must be in MM-YYYY format")
	}

	query := `
		SELECT COALESCE(SUM(price), 0)
		FROM subscriptions
		WHERE user_id = $1
		  AND start_date <= $3
		  AND (end_date IS NULL OR end_date >= $2)`

	args := []any{userID, from, to}
	argIndex := 4

	if serviceName != "" {
		query += fmt.Sprintf(" AND service_name = $%d", argIndex)
		args = append(args, serviceName)
	}

	var total int
	err := r.conn.QueryRow(ctx, query, args...).Scan(&total)
	if err != nil {
		slog.Error("Failed to calculate total cost", "user_id", userID, "error", err)
		return 0, fmt.Errorf("database aggregation failed: %w", err)
	}

	return total, nil
}

func isValidMonthYear(s string) bool {
	if len(s) != 7 || s[2] != '-' {
		return false
	}
	month, err1 := strconv.Atoi(s[0:2])
	year, err2 := strconv.Atoi(s[3:7])
	if err1 != nil || err2 != nil {
		return false
	}
	return month >= 1 && month <= 12 && year >= 1900 && year <= 2100
}
