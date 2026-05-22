package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/LimeAsnet/SubAggregator/internal/database"
	"github.com/LimeAsnet/SubAggregator/internal/models"
	"github.com/LimeAsnet/SubAggregator/internal/service"
	"github.com/gofrs/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var _ service.SubscriptionRepository = (*SubscriptionRepository)(nil)

type SubscriptionRepository struct {
	pool *database.Pool
}

func New(pool *database.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{pool: pool}
}

func (subRep *SubscriptionRepository) activeSubscription(ctx context.Context, sub *models.Subscription) error {
	var id int64
	query := `
		SELECT id FROM subscriptions
		WHERE user_id = $1
		  AND service_name = $2
		  AND (end_date IS NULL OR end_date > $3)
		LIMIT 1
	`
	err := subRep.pool.QueryRow(ctx, query,
		sub.UserID,
		sub.ServiceName,
		sub.StartDate,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check active subscription: %w", err)
	}
	return fmt.Errorf("%w (existing id: %d)", service.ErrActiveSubscription, id)
}

func (subRep *SubscriptionRepository) Create(ctx context.Context, sub *models.Subscription) (int64, error) {

	err := subRep.activeSubscription(ctx, sub)
	if err != nil {
		return 0, err
	}

	var activeSubId int64
	err = subRep.pool.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		query := `
        INSERT INTO subscriptions (service_name, monthly_cost, user_id, start_date, end_date)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `
		if err := tx.QueryRow(ctx, query,
			sub.ServiceName,
			sub.MonthlyCost,
			sub.UserID,
			sub.StartDate,
			sub.EndDate,
		).Scan(&activeSubId); err != nil {
			return fmt.Errorf("insert subscription: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("insert subscription: %w", err)
	}
	return activeSubId, nil
}

func (r *SubscriptionRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]models.Subscription, error) {
	query := `
        SELECT id, service_name, monthly_cost, user_id, start_date, end_date
        FROM subscriptions
        WHERE user_id = $1
        ORDER BY start_date DESC
    `
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("select subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []models.Subscription
	for rows.Next() {
		var s models.Subscription
		if err := rows.Scan(&s.ID, &s.ServiceName, &s.MonthlyCost, &s.UserID, &s.StartDate, &s.EndDate); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("select subscriptions: %w", err)
	}
	return subs, nil
}

func (r *SubscriptionRepository) Update(ctx context.Context, id int64, endDate *string) error {
	return r.pool.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		query := `
        UPDATE subscriptions
        SET end_date = COALESCE($1::date, end_date)
        WHERE id = $2
    `
		tag, err := tx.Exec(ctx, query, endDate, id)
		if err != nil {
			if isEndBeforeStartConstraint(err) {
				return service.ErrEndBeforeStart
			}
			return fmt.Errorf("update subscription: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return service.ErrNotFound
		}
		return nil
	})
}

func isEndBeforeStartConstraint(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23514" && pgErr.ConstraintName == "end_after_start"
}

func (r *SubscriptionRepository) Delete(ctx context.Context, id int64) error {
	return r.pool.WithTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		query := `DELETE FROM subscriptions WHERE id = $1`
		_, err := tx.Exec(ctx, query, id)
		return err
	})
}

func (r *SubscriptionRepository) TotalCost(ctx context.Context, totalAmountReq *models.GetSubscriptionTotalAmountRequest) (models.GetSubscriptionTotalAmountResponse, error) {
	subTotalAmount := models.GetSubscriptionTotalAmountResponse{}
	query := `
        SELECT 		   
		SUM(
		(
		((EXTRACT(YEAR FROM COALESCE(end_date, CURRENT_DATE)) - EXTRACT(YEAR FROM start_date)) * 12 
		+ 
		(EXTRACT(MONTH FROM COALESCE(end_date, CURRENT_DATE)) - EXTRACT(MONTH FROM start_date)))::INT 
		* monthly_cost
		)) AS total_cost
		FROM subscriptions
		WHERE user_id = $1
		AND service_name = $2
		AND start_date >= $3
		AND (end_date <= $4 OR end_date is null)
		group by user_id, service_name;
    `
	err := r.pool.QueryRow(ctx, query,
		totalAmountReq.UserID,
		totalAmountReq.ServiceName,
		totalAmountReq.StartDate,
		totalAmountReq.EndDate).Scan(&subTotalAmount.TotalAmount)

	if err != nil {
		return models.GetSubscriptionTotalAmountResponse{}, fmt.Errorf("select total amount: %w", err)
	}

	return subTotalAmount, nil
}
