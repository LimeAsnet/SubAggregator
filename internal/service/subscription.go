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
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]models.Subscription, error)
	Update(ctx context.Context, id int64, endDate *string) error
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

func (s *SubscriptionService) ListByUserID(ctx context.Context, userID uuid.UUID) ([]models.Subscription, error) {
	subs, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	if subs == nil {
		return []models.Subscription{}, nil
	}
	return subs, nil
}

func (s *SubscriptionService) Update(ctx context.Context, id int64, req models.PatchSubscriptionRequest) error {
	if req.EndDate == "" {
		return ErrEmptyPatch
	}

	parsed, err := ParseMonthYear(req.EndDate)
	if err != nil {
		return err
	}
	formatted := FormatMonthYearDate(parsed)
	endDate := &formatted

	if err := s.repo.Update(ctx, id, endDate); err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrEndBeforeStart) {
			return err
		}
		return fmt.Errorf("update subscription: %w", err)
	}
	return nil
}

func (s *SubscriptionService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
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
