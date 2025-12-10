package repository

import (
	"context"
	"subscription-aggregator/internal/model"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, sub *model.Subscription) error
	GetByID(ctx context.Context, id string) (*model.Subscription, error)
	ListByUserID(ctx context.Context, userID string) ([]model.Subscription, error)
	Update(ctx context.Context, id string, sub *model.Subscription) error
	Delete(ctx context.Context, id string) error
	TotalCost(ctx context.Context, userID, serviceName, from, to string) (int, error)
}
