package mocks

import (
	"context"

	"github.com/LimeAsnet/SubAggregator/internal/models"
	"github.com/gofrs/uuid"
)

type SubscriptionRepository struct {
	CreateFn    func(ctx context.Context, sub *models.Subscription) (int64, error)
	ListFn      func(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]models.Subscription, int64, error)
	UpdateFn    func(ctx context.Context, id int64, endDate *string) error
	DeleteFn    func(ctx context.Context, id int64) error
	TotalCostFn func(ctx context.Context, req *models.GetSubscriptionTotalAmountRequest) (models.GetSubscriptionTotalAmountResponse, error)
}

func (m *SubscriptionRepository) Create(ctx context.Context, sub *models.Subscription) (int64, error) {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, sub)
	}
	return 0, nil
}

func (m *SubscriptionRepository) ListByUserID(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]models.Subscription, int64, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, userID, page, pageSize)
	}
	return nil, 0, nil
}

func (m *SubscriptionRepository) Update(ctx context.Context, id int64, endDate *string) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, id, endDate)
	}
	return nil
}

func (m *SubscriptionRepository) Delete(ctx context.Context, id int64) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}

func (m *SubscriptionRepository) TotalCost(ctx context.Context, req *models.GetSubscriptionTotalAmountRequest) (models.GetSubscriptionTotalAmountResponse, error) {
	if m.TotalCostFn != nil {
		return m.TotalCostFn(ctx, req)
	}
	return models.GetSubscriptionTotalAmountResponse{}, nil
}
