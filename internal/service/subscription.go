package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/LimeAsnet/SubAggregator/internal/models"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, sub *models.Subscription) (int64, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]models.Subscription, int64, error)
	Update(ctx context.Context, id int64, endDate *string) (models.Subscription, error)
	Delete(ctx context.Context, id int64) error
	TotalCost(ctx context.Context, req *models.GetSubscriptionTotalAmountRequest) (models.GetSubscriptionTotalAmountResponse, error)
}

type SubscriptionService struct {
	repo SubscriptionRepository
}

func NewSubscriptionService(repo SubscriptionRepository) *SubscriptionService {
	return &SubscriptionService{repo: repo}
}

func (s *SubscriptionService) Create(ctx context.Context, req models.CreateSubscriptionRequest) (int64, error) {
	startDate, err := ParseMonthYear(req.StartDate)
	if err != nil {
		return 0, err
	}

	var endDate *time.Time
	if req.EndDate != "" {
		parsed, err := ParseMonthYear(req.EndDate)
		if err != nil {
			return 0, err
		}
		endDate = &parsed
	}

	sub := &models.Subscription{
		ServiceName: req.ServiceName,
		MonthlyCost: req.MonthlyCost,
		UserID:      req.UserID,
		StartDate:   startDate,
		EndDate:     endDate,
	}

	id, err := s.repo.Create(ctx, sub)
	if err != nil {
		if errors.Is(err, ErrActiveSubscription) || errors.Is(err, ErrEndBeforeStart) {
			return 0, err
		}
		return 0, fmt.Errorf("create subscription: %w", err)
	}
	return id, nil
}

func (s *SubscriptionService) ListByUserID(ctx context.Context, userID uuid.UUID, p Pagination) (models.ListSubscriptionsResponse, error) {
	subs, total, err := s.repo.ListByUserID(ctx, userID, p.Page, p.PageSize)
	if err != nil {
		return models.ListSubscriptionsResponse{}, fmt.Errorf("list subscriptions: %w", err)
	}
	if total == 0 {
		return models.ListSubscriptionsResponse{}, ErrNotFound
	}
	if subs == nil {
		subs = []models.Subscription{}
	}
	return models.ListSubscriptionsResponse{
		Items:    subs,
		Page:     p.Page,
		PageSize: p.PageSize,
		Total:    total,
	}, nil
}

func (s *SubscriptionService) Update(ctx context.Context, id int64, req models.PatchSubscriptionRequest) (models.Subscription, error) {
	if req.EndDate == "" {
		return models.Subscription{}, ErrEmptyPatch
	}

	parsed, err := ParseMonthYear(req.EndDate)
	if err != nil {
		return models.Subscription{}, err
	}
	formatted := FormatMonthYearDate(parsed)
	endDate := &formatted

	sub, err := s.repo.Update(ctx, id, endDate)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrEndBeforeStart) {
			return models.Subscription{}, err
		}
		return models.Subscription{}, fmt.Errorf("update subscription: %w", err)
	}
	return sub, nil
}

func (s *SubscriptionService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}
		return fmt.Errorf("delete subscription: %w", err)
	}
	return nil
}

func (s *SubscriptionService) TotalCost(ctx context.Context, userID uuid.UUID, serviceName, startDateStr, endDateStr string) (models.GetSubscriptionTotalAmountResponse, error) {
	startDate, err := ParseMonthYear(startDateStr)
	if err != nil {
		return models.GetSubscriptionTotalAmountResponse{}, err
	}

	endDate, err := ParseMonthYear(endDateStr)
	if err != nil {
		return models.GetSubscriptionTotalAmountResponse{}, err
	}

	req := &models.GetSubscriptionTotalAmountRequest{
		UserID:      userID,
		ServiceName: serviceName,
		StartDate:   startDate,
		EndDate:     endDate,
	}

	result, err := s.repo.TotalCost(ctx, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.GetSubscriptionTotalAmountResponse{}, ErrNotFound
		}
		return models.GetSubscriptionTotalAmountResponse{}, fmt.Errorf("total cost: %w", err)
	}
	return result, nil
}

func ParseUserID(value string) (uuid.UUID, error) {
	id, err := uuid.FromString(value)
	if err != nil {
		return uuid.Nil, ErrInvalidUserID
	}
	return id, nil
}
