package models

import (
	"time"

	"github.com/gofrs/uuid"
)

type Subscription struct {
	ID          int64      `json:"id"`
	ServiceName string     `json:"service_name"`
	MonthlyCost int64      `json:"monthly_cost"`
	UserID      uuid.UUID  `json:"user_id"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}

// CreateSubscriptionRequest — тело запроса POST /subscriptions
type CreateSubscriptionRequest struct {
	ServiceName string    `json:"service_name" binding:"required,min=1,max=255"`
	MonthlyCost int64     `json:"monthly_cost" binding:"required,gt=0"`
	UserID      uuid.UUID `json:"user_id" binding:"required"`
	StartDate   string    `json:"start_date" binding:"required"`
	EndDate     string    `json:"end_date,omitempty"`
}

// CreateSubscriptionResponse — ответ после создания подписки
type CreateSubscriptionResponse struct {
	ID int64 `json:"id" example:"1"`
}

// PatchSubscriptionRequest — тело запроса PATCH /subscriptions/:id
type PatchSubscriptionRequest struct {
	EndDate string `json:"end_date" binding:"required" example:"12-2026"`
}

// GetSubscriptionTotalAmountRequest — тело запроса Get /subscriptions/total
type GetSubscriptionTotalAmountRequest struct {
	ServiceName string    `json:"service_name"`
	UserID      uuid.UUID `json:"user_id"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
}

// GetSubscriptionTotalAmountResponse — ответ после получения общей стоимости подписок
type GetSubscriptionTotalAmountResponse struct {
	TotalAmount float64 `json:"total_amount"`
}

// ListSubscriptionsResponse — ответ GET /subscriptions
type ListSubscriptionsResponse struct {
	Items    []Subscription `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
	Total    int64          `json:"total"`
}

// ErrorResponse — тело ответа с ошибкой
type ErrorResponse struct {
	Error string `json:"error" example:"invalid request"`
}
