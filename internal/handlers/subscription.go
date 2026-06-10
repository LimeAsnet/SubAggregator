package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/LimeAsnet/SubAggregator/internal/models"
	"github.com/LimeAsnet/SubAggregator/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

type subscriptionService interface {
	Create(ctx context.Context, req models.CreateSubscriptionRequest) (int64, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, p service.Pagination) (models.ListSubscriptionsResponse, error)
	Update(ctx context.Context, id int64, req models.PatchSubscriptionRequest) error
	Delete(ctx context.Context, id int64) error
	TotalCost(ctx context.Context, userID uuid.UUID, serviceName, startDate, endDate string) (models.GetSubscriptionTotalAmountResponse, error)
}

type SubscriptionHandler struct {
	svc subscriptionService
	log *slog.Logger
}

func NewSubscriptionHandler(svc *service.SubscriptionService, log *slog.Logger) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc, log: log}
}

func (h *SubscriptionHandler) Register(r gin.IRouter) {
	subGroup := r.Group("/subscriptions")
	subGroup.POST("", h.Create)
	subGroup.GET("", h.List)
	subGroup.GET("/total", h.TotalCost)
	subGroup.PATCH("/:id", h.Update)
	subGroup.DELETE("/:id", h.Delete)
}

// @Summary      Создать подписку
// @Description  Создаёт новую подписку. Даты в формате MM-YYYY (например 07-2025).
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        body  body      models.CreateSubscriptionRequest  true  "Данные подписки"
// @Success      201   {object}  models.CreateSubscriptionResponse
// @Failure      400   {object}  models.ErrorResponse
// @Failure      409   {object}  models.ErrorResponse
// @Failure      500   {object}  models.ErrorResponse
// @Router       /subscriptions [post]
func (h *SubscriptionHandler) Create(c *gin.Context) {
	var req models.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("invalid create subscription request", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, h.log, err, "failed to create subscription")
		return
	}

	h.log.Info("subscription created", slog.Int64("id", id), slog.String("user_id", req.UserID.String()))
	c.JSON(http.StatusCreated, models.CreateSubscriptionResponse{ID: id})
}

// @Summary      Список подписок пользователя
// @Description  Возвращает подписки пользователя с пагинацией, отсортированные по дате начала (убывание).
// @Tags         subscriptions
// @Produce      json
// @Param        user_id    query     string  true   "UUID пользователя"  format(uuid)
// @Param        page       query     int     false  "Номер страницы"     default(1)
// @Param        page_size  query     int     false  "Размер страницы"    default(20)
// @Success      200        {object}  models.ListSubscriptionsResponse
// @Failure      400        {object}  models.ErrorResponse
// @Failure      500        {object}  models.ErrorResponse
// @Router       /subscriptions [get]
func (h *SubscriptionHandler) List(c *gin.Context) {
	userID, err := service.ParseUserID(c.Query("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required and must be a valid UUID"})
		return
	}

	pagination, err := service.ParsePagination(c.Query("page"), c.Query("page_size"))
	if err != nil {
		writeServiceError(c, h.log, err, "invalid pagination parameters")
		return
	}

	result, err := h.svc.ListByUserID(c.Request.Context(), userID, pagination)
	if err != nil {
		writeServiceError(c, h.log, err, "failed to list subscriptions")
		return
	}

	c.JSON(http.StatusOK, result)
}

// @Summary      Обновить дату окончания подписки
// @Description  Обновляет только end_date подписки в формате MM-YYYY (например 12-2026).
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        id    path      int                             true  "ID подписки"
// @Param        body  body      models.PatchSubscriptionRequest true  "Новая дата окончания"
// @Success      204   "No Content"
// @Failure      400   {object}  models.ErrorResponse
// @Failure      404   {object}  models.ErrorResponse
// @Failure      500   {object}  models.ErrorResponse
// @Router       /subscriptions/{id} [patch]
func (h *SubscriptionHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.log.Warn("invalid subscription id", slog.String("id", c.Param("id")))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
		return
	}

	var req models.PatchSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warn("invalid patch subscription request", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.Update(c.Request.Context(), id, req); err != nil {
		writeServiceError(c, h.log, err, "failed to update subscription")
		return
	}

	h.log.Info("subscription updated", slog.Int64("id", id))
	c.Status(http.StatusNoContent)
}

// @Summary      Удалить подписку
// @Tags         subscriptions
// @Produce      json
// @Param        id  path  int  true  "ID подписки"
// @Success      204  "No Content"
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /subscriptions/{id} [delete]
func (h *SubscriptionHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		h.log.Warn("invalid subscription id", slog.String("id", c.Param("id")))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscription id"})
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		writeServiceError(c, h.log, err, "failed to delete subscription")
		return
	}

	h.log.Info("subscription deleted", slog.Int64("id", id))
	c.Status(http.StatusNoContent)
}

// @Summary      Суммарная стоимость подписок
// @Description  Считает суммарную стоимость подписок пользователя на сервис за период. Даты периода в формате MM-YYYY.
// @Tags         subscriptions
// @Produce      json
// @Param        user_id       query  string  true  "UUID пользователя"  format(uuid)
// @Param        service_name  query  string  true  "Название сервиса"   example(Netflix)
// @Param        start_date    query  string  true  "Начало периода"     example(01-2025)
// @Param        end_date      query  string  true  "Конец периода"      example(12-2025)
// @Success      200  {object}  models.GetSubscriptionTotalAmountResponse
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /subscriptions/total [get]
func (h *SubscriptionHandler) TotalCost(c *gin.Context) {
	userID, err := service.ParseUserID(c.Query("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required and must be a valid UUID"})
		return
	}

	if c.Query("service_name") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "service_name is required"})
		return
	}

	result, err := h.svc.TotalCost(
		c.Request.Context(),
		userID,
		c.Query("service_name"),
		c.Query("start_date"),
		c.Query("end_date"),
	)
	if err != nil {
		writeServiceError(c, h.log, err, "failed to calculate total cost")
		return
	}

	c.JSON(http.StatusOK, result)
}
