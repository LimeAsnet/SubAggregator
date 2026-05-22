CREATE TABLE subscriptions (
    id            BIGSERIAL PRIMARY KEY,
    service_name  VARCHAR(255) NOT NULL,
    monthly_cost  BIGINT NOT NULL CHECK (monthly_cost >= 0),
    user_id       UUID NOT NULL,
    start_date    DATE NOT NULL,
    end_date      DATE,
    CONSTRAINT end_after_start CHECK (end_date IS NULL OR end_date > start_date)
);

CREATE INDEX idx_subscriptions_user_service_dates
    ON subscriptions (user_id, service_name, start_date, end_date);