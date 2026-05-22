package service

import "errors"

var (
	ErrInvalidUserID       = errors.New("invalid user id")
	ErrInvalidDate         = errors.New("invalid date format")
	ErrEmptyPatch          = errors.New("end_date is required")
	ErrNotFound            = errors.New("subscription not found")
	ErrActiveSubscription  = errors.New("active subscription already exists for this user and service")
	ErrEndBeforeStart      = errors.New("end_date must be after start_date")
)
