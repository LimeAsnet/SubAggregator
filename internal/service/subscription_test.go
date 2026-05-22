package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LimeAsnet/SubAggregator/internal/models"
	"github.com/LimeAsnet/SubAggregator/internal/service/mocks"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testUserID = uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440000"))

func TestSubscriptionService_Create(t *testing.T) {
	var captured *models.Subscription
	repo := &mocks.SubscriptionRepository{
		CreateFn: func(_ context.Context, sub *models.Subscription) (int64, error) {
			captured = sub
			return 10, nil
		},
	}
	svc := NewSubscriptionService(repo)

	id, err := svc.Create(context.Background(), models.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		MonthlyCost: 599,
		UserID:      testUserID,
		StartDate:   "07-2025",
		EndDate:     "12-2025",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(10), id)
	require.NotNil(t, captured)
	assert.Equal(t, time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC), captured.StartDate)
}

func TestSubscriptionService_Create_InvalidDate(t *testing.T) {
	svc := NewSubscriptionService(&mocks.SubscriptionRepository{})
	_, err := svc.Create(context.Background(), models.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		MonthlyCost: 599,
		UserID:      testUserID,
		StartDate:   "bad",
	})
	assert.ErrorIs(t, err, ErrInvalidDate)
}

func TestSubscriptionService_Create_WithoutEndDate(t *testing.T) {
	var captured *models.Subscription
	repo := &mocks.SubscriptionRepository{
		CreateFn: func(_ context.Context, sub *models.Subscription) (int64, error) {
			captured = sub
			return 42, nil
		},
	}
	svc := NewSubscriptionService(repo)

	id, err := svc.Create(context.Background(), models.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		MonthlyCost: 599,
		UserID:      testUserID,
		StartDate:   "07-2025",
		EndDate:     "",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(42), id)
	require.NotNil(t, captured)
	assert.Nil(t, captured.EndDate)
}

func TestSubscriptionService_List_EmptySlice(t *testing.T) {
	repo := &mocks.SubscriptionRepository{
		ListFn: func(context.Context, uuid.UUID) ([]models.Subscription, error) {
			return nil, nil
		},
	}
	svc := NewSubscriptionService(repo)

	subs, err := svc.ListByUserID(context.Background(), testUserID)
	require.NoError(t, err)
	assert.Equal(t, []models.Subscription{}, subs)
}

func TestSubscriptionService_List_ReturnsData(t *testing.T) {
	expected := []models.Subscription{
		{ID: 1, ServiceName: "Netflix", UserID: testUserID},
		{ID: 2, ServiceName: "Spotify", UserID: testUserID},
	}
	repo := &mocks.SubscriptionRepository{
		ListFn: func(_ context.Context, userID uuid.UUID) ([]models.Subscription, error) {
			assert.Equal(t, testUserID, userID)
			return expected, nil
		},
	}
	svc := NewSubscriptionService(repo)

	subs, err := svc.ListByUserID(context.Background(), testUserID)
	require.NoError(t, err)
	assert.Equal(t, expected, subs)
}

func TestSubscriptionService_Update_EmptyEndDate(t *testing.T) {
	svc := NewSubscriptionService(&mocks.SubscriptionRepository{})
	err := svc.Update(context.Background(), 1, models.PatchSubscriptionRequest{})
	assert.ErrorIs(t, err, ErrEmptyPatch)
}

func TestSubscriptionService_Update_InvalidDate(t *testing.T) {
	svc := NewSubscriptionService(&mocks.SubscriptionRepository{})
	err := svc.Update(context.Background(), 1, models.PatchSubscriptionRequest{EndDate: "bad-date"})
	assert.ErrorIs(t, err, ErrInvalidDate)
}

func TestSubscriptionService_Create_ActiveSubscription(t *testing.T) {
	repo := &mocks.SubscriptionRepository{
		CreateFn: func(context.Context, *models.Subscription) (int64, error) {
			return 0, ErrActiveSubscription
		},
	}
	svc := NewSubscriptionService(repo)

	_, err := svc.Create(context.Background(), models.CreateSubscriptionRequest{
		ServiceName: "Netflix",
		MonthlyCost: 599,
		UserID:      testUserID,
		StartDate:   "07-2025",
	})
	assert.ErrorIs(t, err, ErrActiveSubscription)
}

func TestSubscriptionService_Update_NotFound(t *testing.T) {
	repo := &mocks.SubscriptionRepository{
		UpdateFn: func(context.Context, int64, *string) error {
			return ErrNotFound
		},
	}
	svc := NewSubscriptionService(repo)

	err := svc.Update(context.Background(), 99, models.PatchSubscriptionRequest{EndDate: "12-2026"})
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSubscriptionService_Update_Success(t *testing.T) {
	var gotID int64
	var gotEnd *string
	repo := &mocks.SubscriptionRepository{
		UpdateFn: func(_ context.Context, id int64, endDate *string) error {
			gotID = id
			gotEnd = endDate
			return nil
		},
	}
	svc := NewSubscriptionService(repo)

	err := svc.Update(context.Background(), 7, models.PatchSubscriptionRequest{EndDate: "12-2026"})
	require.NoError(t, err)
	assert.Equal(t, int64(7), gotID)
	require.NotNil(t, gotEnd)
	assert.Equal(t, "2026-12-01", *gotEnd)
}

func TestSubscriptionService_TotalCost_NotFound(t *testing.T) {
	repo := &mocks.SubscriptionRepository{
		TotalCostFn: func(context.Context, *models.GetSubscriptionTotalAmountRequest) (models.GetSubscriptionTotalAmountResponse, error) {
			return models.GetSubscriptionTotalAmountResponse{}, pgx.ErrNoRows
		},
	}
	svc := NewSubscriptionService(repo)

	_, err := svc.TotalCost(context.Background(), testUserID, "Netflix", "01-2025", "12-2025")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestSubscriptionService_TotalCost_InvalidStartDate(t *testing.T) {
	svc := NewSubscriptionService(&mocks.SubscriptionRepository{})
	_, err := svc.TotalCost(context.Background(), testUserID, "Netflix", "bad-date", "12-2025")
	assert.ErrorIs(t, err, ErrInvalidDate)
}

func TestSubscriptionService_TotalCost_Success(t *testing.T) {
	repo := &mocks.SubscriptionRepository{
		TotalCostFn: func(_ context.Context, req *models.GetSubscriptionTotalAmountRequest) (models.GetSubscriptionTotalAmountResponse, error) {
			assert.Equal(t, "Netflix", req.ServiceName)
			return models.GetSubscriptionTotalAmountResponse{TotalAmount: 1000}, nil
		},
	}
	svc := NewSubscriptionService(repo)

	result, err := svc.TotalCost(context.Background(), testUserID, "Netflix", "01-2025", "12-2025")
	require.NoError(t, err)
	assert.Equal(t, 1000.0, result.TotalAmount)
}

func TestParseUserID(t *testing.T) {
	_, err := ParseUserID("invalid")
	assert.ErrorIs(t, err, ErrInvalidUserID)

	id, err := ParseUserID(testUserID.String())
	require.NoError(t, err)
	assert.Equal(t, testUserID, id)
}

func TestSubscriptionService_Delete_Error(t *testing.T) {
	repo := &mocks.SubscriptionRepository{
		DeleteFn: func(context.Context, int64) error {
			return errors.New("db")
		},
	}
	svc := NewSubscriptionService(repo)
	err := svc.Delete(context.Background(), 1)
	assert.Error(t, err)
}
