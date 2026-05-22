package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/LimeAsnet/SubAggregator/internal/service"
	"github.com/gin-gonic/gin"
)

func mapServiceError(err error) (status int, message string, ok bool) {
	switch {
	case errors.Is(err, service.ErrInvalidDate):
		return http.StatusBadRequest, "date must be in MM-YYYY format", true
	case errors.Is(err, service.ErrEmptyPatch):
		return http.StatusBadRequest, "end_date is required", true
	case errors.Is(err, service.ErrInvalidUserID):
		return http.StatusBadRequest, "user_id is required and must be a valid UUID", true
	case errors.Is(err, service.ErrActiveSubscription):
		return http.StatusConflict, err.Error(), true
	case errors.Is(err, service.ErrEndBeforeStart):
		return http.StatusBadRequest, err.Error(), true
	case errors.Is(err, service.ErrNotFound):
		return http.StatusNotFound, err.Error(), true
	default:
		return 0, "", false
	}
}

func writeServiceError(c *gin.Context, log *slog.Logger, err error, fallback string) {
	if status, message, ok := mapServiceError(err); ok {
		if status >= http.StatusInternalServerError {
			log.Error(fallback, slog.String("error", err.Error()))
		}
		c.JSON(status, gin.H{"error": message})
		return
	}

	log.Error(fallback, slog.String("error", err.Error()))
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
