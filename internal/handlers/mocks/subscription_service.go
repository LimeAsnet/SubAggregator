package mocks

import (
	"context"

	"github.com/LimeAsnet/SubAggregator/internal/models"
	"github.com/LimeAsnet/SubAggregator/internal/service"
	"github.com/gofrs/uuid"
)

type SubscriptionService struct {
	CreateFn    func(ctx context.Context, req models.CreateSubscriptionRequest) (int64, error)
	ListFn      func(ctx context.Context, userID uuid.UUID, p service.Pagination) (models.ListSubscriptionsResponse, error)
	UpdateFn    func(ctx context.Context, id int64, req models.PatchSubscriptionRequest) (models.Subscription, error)
	DeleteFn    func(ctx context.Context, id int64) error
	TotalCostFn func(ctx context.Context, userID uuid.UUID, serviceName, startDate, endDate string) (models.GetSubscriptionTotalAmountResponse, error)
}

func (m *SubscriptionService) Create(ctx context.Context, req models.CreateSubscriptionRequest) (int64, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, req)
	}
	return 0, nil
}

func (m *SubscriptionService) ListByUserID(ctx context.Context, userID uuid.UUID, p service.Pagination) (models.ListSubscriptionsResponse, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, userID, p)
	}
	return models.ListSubscriptionsResponse{}, nil
}

func (m *SubscriptionService) Update(ctx context.Context, id int64, req models.PatchSubscriptionRequest) (models.Subscription, error) {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, id, req)
	}
	return models.Subscription{}, nil
}

func (m *SubscriptionService) Delete(ctx context.Context, id int64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *SubscriptionService) TotalCost(ctx context.Context, userID uuid.UUID, serviceName, startDate, endDate string) (models.GetSubscriptionTotalAmountResponse, error) {
	if m.TotalCostFn != nil {
		return m.TotalCostFn(ctx, userID, serviceName, startDate, endDate)
	}
	return models.GetSubscriptionTotalAmountResponse{}, nil
}
